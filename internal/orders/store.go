package orders

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	dyn "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/imrishuroy/go-idempotent-orderflow/internal/aws"
)

// Store encapsulates operations on the orders table.
type Store struct {
	client    aws.DynamoDBAPI
	tableName string
	nowFunc   func() time.Time
}

// NewStore creates a new orders Store.
func NewStore(client aws.DynamoDBAPI, tableName string) *Store {
	return &Store{
		client:    client,
		tableName: tableName,
		nowFunc:   time.Now,
	}
}

// CreateWithIdempotencyTransaction atomically creates:
//   - idempotency record in idempotencyTable (with ConditionExpression attribute_not_exists(idempotency_key))
//   - order record in orders table
//
// It marshals both items and issues a TransactWriteItems call.
// idempotencyItem must be a serializable struct with attribute idempotency_key present.
// order is the Order struct to persist; order.OrderID must be set by caller.
func (s *Store) CreateWithIdempotencyTransaction(ctx context.Context, dynamo aws.DynamoDBAPI, idempotencyTable string, idempotencyItem interface{}, order Order, ttlWindow time.Duration) error {
	// marshal idempotency item
	idempMap, err := attributevalue.MarshalMap(idempotencyItem)
	if err != nil {
		return fmt.Errorf("marshal idempotency item: %w", err)
	}
	// ensure idempotency TTL if needed: caller can include expires_at field; if not present, add it
	if _, ok := idempMap["expires_at"]; !ok && ttlWindow > 0 {
		expires := time.Now().Add(ttlWindow).Unix()
		idempMap["expires_at"] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", expires)}
	}

	// marshal order item
	// set CreatedAt/UpdatedAt if empty
	now := s.nowFunc()
	if order.CreatedAt.IsZero() {
		order.CreatedAt = now
	}
	order.UpdatedAt = now

	orderMap, err := attributevalue.MarshalMap(order)
	if err != nil {
		return fmt.Errorf("marshal order item: %w", err)
	}

	// build transact items: Put idempotency with condition, Put order in orders table
	transactItems := []types.TransactWriteItem{
		{
			Put: &types.Put{
				TableName:           &idempotencyTable,
				Item:                idempMap,
				ConditionExpression: awsString("attribute_not_exists(idempotency_key)"),
			},
		},
		{
			Put: &types.Put{
				TableName: &s.tableName,
				Item:      orderMap,
				// we could guard here if needed: ConditionExpression attribute_not_exists(order_id)
			},
		},
	}

	input := &dyn.TransactWriteItemsInput{
		TransactItems: transactItems,
	}

	_, err = dynamo.TransactWriteItems(ctx, input)
	if err != nil {
		// detect transaction canceled / conditional failure
		var tce *types.TransactionCanceledException
		if errors.As(err, &tce) {
			return fmt.Errorf("transaction canceled (likely idempotency key exists): %w", err)
		}
		return fmt.Errorf("transact write: %w", err)
	}
	return nil
}

// Get fetches an order by order_id. Returns (nil, nil) if not found.
func (s *Store) Get(ctx context.Context, orderID string) (*Order, error) {
	key := map[string]types.AttributeValue{
		"order_id": &types.AttributeValueMemberS{Value: orderID},
	}
	out, err := s.client.GetItem(ctx, &dyn.GetItemInput{
		TableName: &s.tableName,
		Key:       key,
	})
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}
	if len(out.Item) == 0 {
		return nil, nil
	}
	var o Order
	if err := attributevalue.UnmarshalMap(out.Item, &o); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}
	return &o, nil
}

// UpdateStatus conditionally updates the order status from expected -> newStatus.
// Returns nil on success, ErrStatusMismatch if condition failed.
var ErrStatusMismatch = errors.New("status mismatch/conditional failed")

func (s *Store) UpdateStatus(ctx context.Context, orderID, expectedStatus, newStatus string) error {
	now := s.nowFunc()
	// Update expression: SET #s = :new, updated_at = :ua, attempts = if_not_exists(attempts, :zero) + :inc
	updateExpr := "SET #s = :new, updated_at = :ua"
	// we will not change attempts here; caller can call IncrementAttempts
	input := &dyn.UpdateItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"order_id": &types.AttributeValueMemberS{Value: orderID},
		},
		UpdateExpression:          &updateExpr,
		ExpressionAttributeNames:  map[string]string{"#s": "status"},
		ExpressionAttributeValues: map[string]types.AttributeValue{":new": &types.AttributeValueMemberS{Value: newStatus}, ":ua": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)}},
		ConditionExpression:       awsString("#s = :expected"),
	}
	// add expected value
	input.ExpressionAttributeValues[":expected"] = &types.AttributeValueMemberS{Value: expectedStatus}

	_, err := s.client.UpdateItem(ctx, input)
	if err != nil {
		// detect conditional check failing
		var sc *types.ConditionalCheckFailedException
		if errors.As(err, &sc) {
			return ErrStatusMismatch
		}
		return fmt.Errorf("update item: %w", err)
	}
	return nil
}

// IncrementAttempts increases the attempts counter by 1 (useful for worker retries)
func (s *Store) IncrementAttempts(ctx context.Context, orderID string) error {
	now := s.nowFunc()
	input := &dyn.UpdateItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"order_id": &types.AttributeValueMemberS{Value: orderID},
		},
		UpdateExpression:          awsString("SET attempts = if_not_exists(attempts, :zero) + :inc, updated_at = :ua"),
		ExpressionAttributeValues: map[string]types.AttributeValue{":zero": &types.AttributeValueMemberN{Value: "0"}, ":inc": &types.AttributeValueMemberN{Value: "1"}, ":ua": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)}},
		ReturnValues:              types.ReturnValueUpdatedNew,
	}
	_, err := s.client.UpdateItem(ctx, input)
	if err != nil {
		return fmt.Errorf("increment attempts: %w", err)
	}
	return nil
}

func awsString(s string) *string { return &s }

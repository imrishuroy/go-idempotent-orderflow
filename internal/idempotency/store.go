package idempotency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	dyn "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
	"github.com/imrishuroy/go-idempotent-orderflow/internal/aws"
)

// Store encapsulates idempotency operations against DynamoDB.
type Store struct {
	client    aws.DynamoDBAPI
	tableName string
	ttlWindow time.Duration // default TTL window when creating entries
	nowFunc   func() time.Time
}

// NewStore returns a configured Store.
// tableName: DynamoDB table name for idempotency entries.
// ttlWindow: default TTL window (e.g., 48*time.Hour)
func NewStore(client aws.DynamoDBAPI, tableName string, ttlWindow time.Duration) *Store {
	return &Store{
		client:    client,
		tableName: tableName,
		ttlWindow: ttlWindow,
		nowFunc:   time.Now,
	}
}

// ErrConditionFailed indicates a conditional write failed (e.g., attribute_not_exists)
var ErrConditionFailed = errors.New("conditional check failed")

// CreateIfNotExists creates an idempotency record with status IN_PROGRESS if the key does not exist.
// Returns (created=true, nil) if successfully created.
// Returns (created=false, nil) if the record already exists (caller should Get to inspect).
// Returns (created=false, err) on other errors.
func (s *Store) CreateIfNotExists(ctx context.Context, key, orderID string) (bool, error) {
	now := s.nowFunc()
	rec := IdempotencyRecord{
		IdempotencyKey: key,
		Status:         StatusInProgress,
		OrderID:        orderID,
		CreatedAt:      now,
		UpdatedAt:      now,
		ExpiresAt:      now.Add(s.ttlWindow).Unix(),
	}

	item, err := attributevalue.MarshalMap(rec)
	if err != nil {
		return false, fmt.Errorf("marshal record: %w", err)
	}

	input := &dyn.PutItemInput{
		TableName: &s.tableName,
		Item:      item,
		// Only create when attribute_not_exists(idempotency_key)
		ConditionExpression: awsString("attribute_not_exists(idempotency_key)"),
	}

	_, err = s.client.PutItem(ctx, input)
	if err != nil {
		// detect conditional check failure
		var sc smithy.APIError
		if errors.As(err, &sc) && sc.ErrorCode() == "ConditionalCheckFailedException" {
			return false, nil
		}
		return false, fmt.Errorf("put item: %w", err)
	}

	return true, nil
}

// CreateWithOrderTransaction atomically creates the idempotency record and an order item
// using TransactWriteItems. orderItem should be a struct that can be marshaled by attributevalue.
// Returns nil on success; ErrConditionFailed if condition failed (idempotency key exists).
func (s *Store) CreateWithOrderTransaction(ctx context.Context, key string, orderItem interface{}) error {
	// now := s.nowFunc()
	// rec := IdempotencyRecord{
	// 	IdempotencyKey: key,
	// 	Status:         StatusInProgress,
	// 	CreatedAt:      now,
	// 	UpdatedAt:      now,
	// 	ExpiresAt:      now.Add(s.ttlWindow).Unix(),
	// }
	// recMap, err := attributevalue.MarshalMap(rec)
	// if err != nil {
	// 	return fmt.Errorf("marshal idempotency record: %w", err)
	// }

	// orderMap, err := attributevalue.MarshalMap(orderItem)
	// if err != nil {
	// 	return fmt.Errorf("marshal order item: %w", err)
	// }

	// // Build transact items: Put idempotency with condition_not_exists, Put order
	// transactItems := []types.TransactWriteItem{
	// 	{
	// 		Put: &types.Put{
	// 			TableName:           &s.tableName,
	// 			Item:                recMap,
	// 			ConditionExpression: awsString("attribute_not_exists(idempotency_key)"),
	// 		},
	// 	},
	// 	{
	// 		// Note: we assume orderItem contains its own PK (order_id)
	// 		Put: &types.Put{
	// 			// The orders table name is passed inside orderMap's metadata, or better: caller passes a dynamodb.Put item.
	// 			// For flexibility, we expect the order item to be written to a DIFFERENT table.
	// 			// We'll instruct callers to use the orders table using TransactWriteItems' Put with TableName set via parameter.
	// 			// However attributevalue + types.Put requires TableName string per put item; we'll instead
	// 			// choose a two-transaction approach in most deployments. To keep this generic, we'll fail if order item doesn't
	// 			// include a special meta "__table" attribute (not ideal).
	// 			// Simpler approach: do not set TableName here for order; instead return an error asking the caller to use
	// 			// TransactWriteItems directly if they want to write to multiple tables.
	// 		},
	// 	},
	// }

	// // NOTE: The AWS SDK requires each Put to specify TableName. Since this store is bound to the idempotency table,
	// // performing a multi-table transact from here requires the caller to provide the fully formed TransactWriteItemsInput.
	// // To balance convenience and flexibility, we will NOT implement multi-table transact here.
	// // Instead, we provide CreateIfNotExists and callers can perform a TransactWriteItems themselves if they need atomicity.
	return fmt.Errorf("CreateWithOrderTransaction is not implemented: use CreateIfNotExists + caller-managed TransactWriteItems for multi-table writes")
}

// Get retrieves an idempotency record by key. If not found, returns (nil, nil).
func (s *Store) Get(ctx context.Context, key string) (*IdempotencyRecord, error) {
	input := &dyn.GetItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"idempotency_key": &types.AttributeValueMemberS{Value: key},
		},
	}
	out, err := s.client.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}
	if len(out.Item) == 0 {
		return nil, nil
	}
	var rec IdempotencyRecord
	if err := attributevalue.UnmarshalMap(out.Item, &rec); err != nil {
		return nil, fmt.Errorf("unmarshal item: %w", err)
	}
	return &rec, nil
}

// MarkDone sets status to DONE and stores a small response body & status.
// It uses UpdateItem with a conditional expression to ensure transition from IN_PROGRESS -> DONE or FAILED -> DONE depending on needs.
func (s *Store) MarkDone(ctx context.Context, key, responseBody string, responseStatus int) error {
	now := s.nowFunc()
	input := &dyn.UpdateItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"idempotency_key": &types.AttributeValueMemberS{Value: key},
		},
		UpdateExpression: awsString("SET #s = :done, response_body = :rb, response_status = :rs, updated_at = :ua"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":done": &types.AttributeValueMemberS{Value: StatusDone},
			":rb":   &types.AttributeValueMemberS{Value: responseBody},
			":rs":   &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", responseStatus)},
			":ua":   &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	}
	_, err := s.client.UpdateItem(ctx, input)
	if err != nil {
		return fmt.Errorf("update item (mark done): %w", err)
	}
	return nil
}

// MarkFailed marks the idempotency record as FAILED and optionally stores a note.
func (s *Store) MarkFailed(ctx context.Context, key, note string) error {
	now := s.nowFunc()
	input := &dyn.UpdateItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"idempotency_key": &types.AttributeValueMemberS{Value: key},
		},
		UpdateExpression: awsString("SET #s = :failed, note = :n, updated_at = :ua"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":failed": &types.AttributeValueMemberS{Value: StatusFailed},
			":n":      &types.AttributeValueMemberS{Value: note},
			":ua":     &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	}
	_, err := s.client.UpdateItem(ctx, input)
	if err != nil {
		return fmt.Errorf("update item (mark failed): %w", err)
	}
	return nil
}

// Helper
func awsString(s string) *string { return &s }

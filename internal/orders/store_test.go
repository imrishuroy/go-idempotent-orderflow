package orders

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	dyn "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// mockDynamo is a simple mock that supports TransactWriteItems, PutItem, GetItem, UpdateItem.
// It stores items per table in a nested map: table -> pkValue -> item map.
type mockDynamo struct {
	mu     sync.Mutex
	tables map[string]map[string]map[string]types.AttributeValue
}

func newMockDynamo() *mockDynamo {
	return &mockDynamo{
		tables: map[string]map[string]map[string]types.AttributeValue{},
	}
}

func (m *mockDynamo) ensureTable(tbl string) {
	if _, ok := m.tables[tbl]; !ok {
		m.tables[tbl] = map[string]map[string]types.AttributeValue{}
	}
}

func (m *mockDynamo) PutItem(ctx context.Context, params *dyn.PutItemInput, optFns ...func(*dyn.Options)) (*dyn.PutItemOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	table := *params.TableName
	m.ensureTable(table)
	// find primary key either order_id or idempotency_key
	var pk string
	if v, ok := params.Item["order_id"]; ok {
		pk = v.(*types.AttributeValueMemberS).Value
	} else if v, ok := params.Item["idempotency_key"]; ok {
		pk = v.(*types.AttributeValueMemberS).Value
	} else {
		return nil, errors.New("no primary key in put item")
	}
	// handle conditional expression attribute_not_exists(idempotency_key)
	if params.ConditionExpression != nil && *params.ConditionExpression == "attribute_not_exists(idempotency_key)" {
		if _, exists := m.tables[table][pk]; exists {
			// simulate conditional failure
			return nil, &types.ConditionalCheckFailedException{}
		}
	}
	// write
	m.tables[table][pk] = params.Item
	return &dyn.PutItemOutput{}, nil
}

func (m *mockDynamo) GetItem(ctx context.Context, params *dyn.GetItemInput, optFns ...func(*dyn.Options)) (*dyn.GetItemOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	table := *params.TableName
	m.ensureTable(table)
	var pk string
	if v, ok := params.Key["order_id"]; ok {
		pk = v.(*types.AttributeValueMemberS).Value
	} else if v, ok := params.Key["idempotency_key"]; ok {
		pk = v.(*types.AttributeValueMemberS).Value
	} else {
		return nil, errors.New("no key attribute")
	}
	item, ok := m.tables[table][pk]
	if !ok {
		return &dyn.GetItemOutput{}, nil
	}
	return &dyn.GetItemOutput{Item: item}, nil
}

func (m *mockDynamo) UpdateItem(ctx context.Context, params *dyn.UpdateItemInput, optFns ...func(*dyn.Options)) (*dyn.UpdateItemOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	table := *params.TableName
	m.ensureTable(table)
	var pk string
	if v, ok := params.Key["order_id"]; ok {
		pk = v.(*types.AttributeValueMemberS).Value
	} else if v, ok := params.Key["idempotency_key"]; ok {
		pk = v.(*types.AttributeValueMemberS).Value
	} else {
		return nil, errors.New("no key attribute")
	}
	item, exists := m.tables[table][pk]
	if !exists {
		return nil, errors.New("item not found")
	}
	// naive apply: if ExpressionAttributeValues present, copy them into item
	for k, v := range params.ExpressionAttributeValues {
		// avoid overwriting reserved keys, simply set
		item[k] = v
	}
	// If ConditionExpression is present and of form "#s = :expected", check existing
	if params.ConditionExpression != nil && *params.ConditionExpression == "#s = :expected" {
		// check status
		if curr, ok := item["status"].(*types.AttributeValueMemberS); ok {
			expected := params.ExpressionAttributeValues[":expected"].(*types.AttributeValueMemberS).Value
			if curr.Value != expected {
				return nil, &types.ConditionalCheckFailedException{}
			}
		} else {
			return nil, &types.ConditionalCheckFailedException{}
		}
	}
	// perform update (this is simplistic)
	// set updated_at if provided
	if v, ok := params.ExpressionAttributeValues[":ua"]; ok {
		item["updated_at"] = v
	}
	if v, ok := params.ExpressionAttributeValues[":new"]; ok {
		item["status"] = v
	}
	// store back
	m.tables[table][pk] = item
	return &dyn.UpdateItemOutput{Attributes: item}, nil
}

func (m *mockDynamo) TransactWriteItems(ctx context.Context, params *dyn.TransactWriteItemsInput, optFns ...func(*dyn.Options)) (*dyn.TransactWriteItemsOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// First pass: verify condition expressions
	for _, it := range params.TransactItems {
		if p := it.Put; p != nil {
			if p.ConditionExpression != nil && *p.ConditionExpression == "attribute_not_exists(idempotency_key)" {
				// ensure idempotency pk exists in its table?
				table := *p.TableName
				m.ensureTable(table)
				kattr := p.Item["idempotency_key"]
				if kattr == nil {
					return nil, errors.New("missing idempotency_key in put")
				}
				pk := kattr.(*types.AttributeValueMemberS).Value
				if _, exists := m.tables[table][pk]; exists {
					// simulate transaction canceled
					return nil, &types.TransactionCanceledException{}
				}
			}
		}
	}
	// Second pass: apply all puts
	for _, it := range params.TransactItems {
		if p := it.Put; p != nil {
			table := *p.TableName
			m.ensureTable(table)
			// determine pk (order_id or idempotency_key)
			var pk string
			if v, ok := p.Item["order_id"]; ok {
				pk = v.(*types.AttributeValueMemberS).Value
			} else if v, ok := p.Item["idempotency_key"]; ok {
				pk = v.(*types.AttributeValueMemberS).Value
			} else {
				return nil, errors.New("no pk found in transact put")
			}
			m.tables[table][pk] = p.Item
		}
	}
	return &dyn.TransactWriteItemsOutput{}, nil
}

func TestCreateWithIdempotencyTransaction_Success(t *testing.T) {
	mock := newMockDynamo()
	ordersTable := "orders"
	idempTable := "idempotency"

	store := NewStore(mock, ordersTable)

	// build idempotency item struct
	now := time.Now()
	idemp := map[string]interface{}{
		"idempotency_key": "key-1",
		"status":          "IN_PROGRESS",
		"created_at":      now.Format(time.RFC3339),
		"updated_at":      now.Format(time.RFC3339),
	}

	order := Order{
		OrderID:    "order-1",
		CustomerID: "cust-1",
		Status:     StatusPending,
		Amount:     123.45,
		Items:      []map[string]interface{}{{"sku": "sku-1", "qty": 1}},
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err := store.CreateWithIdempotencyTransaction(context.Background(), mock, idempTable, idemp, order, 48*time.Hour)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// verify both tables contain items
	// idempotency
	idempItem, ok := mock.tables[idempTable]["key-1"]
	if !ok {
		t.Fatalf("idempotency item not stored")
	}
	if _, ok := idempItem["idempotency_key"]; !ok {
		t.Fatalf("idempotency_key missing in stored item")
	}
	// orders
	orderItem, ok := mock.tables[ordersTable]["order-1"]
	if !ok {
		t.Fatalf("order item not stored")
	}
	// unmarshal order item to Order
	var got Order
	if err := attributevalue.UnmarshalMap(orderItem, &got); err != nil {
		t.Fatalf("unmarshal order: %v", err)
	}
	if got.OrderID != order.OrderID {
		t.Fatalf("order id mismatch")
	}
}

func TestCreateWithIdempotencyTransaction_ExistingIdempotency_Fails(t *testing.T) {
	mock := newMockDynamo()
	ordersTable := "orders"
	idempTable := "idempotency"

	// pre-insert idempotency key
	mock.ensureTable(idempTable)
	mock.tables[idempTable]["key-2"] = map[string]types.AttributeValue{
		"idempotency_key": &types.AttributeValueMemberS{Value: "key-2"},
		"status":          &types.AttributeValueMemberS{Value: "DONE"},
	}

	store := NewStore(mock, ordersTable)

	idemp := map[string]interface{}{
		"idempotency_key": "key-2",
		"status":          "IN_PROGRESS",
	}
	order := Order{
		OrderID:    "order-2",
		CustomerID: "cust-2",
		Status:     StatusPending,
		Amount:     10.0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := store.CreateWithIdempotencyTransaction(context.Background(), mock, idempTable, idemp, order, 48*time.Hour)
	if err == nil {
		t.Fatalf("expected transaction canceled error, got nil")
	}
}

func TestUpdateStatus_Condition_SuccessAndFail(t *testing.T) {
	mock := newMockDynamo()
	tbl := "orders"
	mock.ensureTable(tbl)
	now := time.Now()
	// insert order
	item, _ := attributevalue.MarshalMap(Order{
		OrderID:    "order-10",
		CustomerID: "c10",
		Status:     StatusPending,
		Amount:     1.0,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	mock.tables[tbl]["order-10"] = item

	store := NewStore(mock, tbl)

	// success: PENDING -> PROCESSING
	err := store.UpdateStatus(context.Background(), "order-10", StatusPending, StatusProcessing)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	// failure: PENDING -> COMPLETED (but current is PROCESSING)
	err = store.UpdateStatus(context.Background(), "order-10", StatusPending, StatusCompleted)
	if err == nil {
		t.Fatalf("expected ErrStatusMismatch, got nil")
	}
	if !errors.Is(err, ErrStatusMismatch) {
		t.Fatalf("expected ErrStatusMismatch, got %v", err)
	}
}

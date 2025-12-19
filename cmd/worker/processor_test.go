package worker

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	awsDynamo "github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/imrishuroy/go-idempotent-orderflow/internal/aws"
	"github.com/imrishuroy/go-idempotent-orderflow/internal/idempotency"
	"github.com/imrishuroy/go-idempotent-orderflow/internal/orders"
)

// --- mock implementations ---

type mockDynamo struct {
	tables map[string]map[string]map[string]types.AttributeValue
}

func newMockDynamo() *mockDynamo {
	return &mockDynamo{
		tables: map[string]map[string]map[string]types.AttributeValue{
			"idempotency": {},
			"orders":      {},
		},
	}
}

func (m *mockDynamo) PutItem(ctx context.Context, in *awsDynamo.PutItemInput, optFns ...func(*awsDynamo.Options)) (*awsDynamo.PutItemOutput, error) {
	// minimal implementation enough for tests...
	return &awsDynamo.PutItemOutput{}, nil
}
func (m *mockDynamo) GetItem(ctx context.Context, in *awsDynamo.GetItemInput, optFns ...func(*awsDynamo.Options)) (*awsDynamo.GetItemOutput, error) {
	table := *in.TableName
	key := in.Key["order_id"]
	if key == nil && in.Key["idempotency_key"] != nil {
		key = in.Key["idempotency_key"]
	}
	k := key.(*types.AttributeValueMemberS).Value
	item, ok := m.tables[table][k]
	if !ok {
		return &awsDynamo.GetItemOutput{}, nil
	}
	return &awsDynamo.GetItemOutput{Item: item}, nil
}
func (m *mockDynamo) UpdateItem(ctx context.Context, in *awsDynamo.UpdateItemInput, optFns ...func(*awsDynamo.Options)) (*awsDynamo.UpdateItemOutput, error) {
	// handle conditional status transitions
	table := *in.TableName
	key := in.Key["order_id"]
	k := key.(*types.AttributeValueMemberS).Value

	_, ok := m.tables[table][k]
	if !ok {
		return nil, &types.ConditionalCheckFailedException{}
	}

	// update status immediately for tests
	m.tables[table][k]["status"] = in.ExpressionAttributeValues[":new"]
	return &awsDynamo.UpdateItemOutput{}, nil
}
func (m *mockDynamo) TransactWriteItems(ctx context.Context, in *awsDynamo.TransactWriteItemsInput, optFns ...func(*awsDynamo.Options)) (*awsDynamo.TransactWriteItemsOutput, error) {
	return &awsDynamo.TransactWriteItemsOutput{}, nil
}

// --- test cases ---

func TestWorkerProcess_Success(t *testing.T) {
	mock := newMockDynamo()

	// Insert order PENDING
	order := orders.Order{
		OrderID:    "o1",
		CustomerID: "c1",
		Status:     orders.StatusPending,
		Amount:     10,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	item, _ := attributevalue.MarshalMap(order)
	mock.tables["orders"]["o1"] = item

	// Insert idempotency record
	idemp := idempotency.IdempotencyRecord{
		IdempotencyKey: "k1",
		Status:         idempotency.StatusInProgress,
		OrderID:        "o1",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	idmap, _ := attributevalue.MarshalMap(idemp)
	mock.tables["idempotency"]["k1"] = idmap

	clients := &aws.AWSClients{DynamoDB: mock}
	p := NewProcessor(clients, "idempotency", "orders")

	msg := WorkerMessage{
		OrderID:        "o1",
		IdempotencyKey: "k1",
	}
	body, _ := json.Marshal(msg)
	ev := events.SQSEvent{
		Records: []events.SQSMessage{
			{Body: string(body)},
		},
	}

	err := p.Handle(context.Background(), ev)
	if err != nil {
		t.Fatalf("unexpected worker error: %v", err)
	}
}

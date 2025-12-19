package idempotency

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
)

func TestCreateIfNotExists_Get_MarkDone_MarkFailed(t *testing.T) {
	mock := newSimpleMock()
	s := NewStore(mock, "idempotency-table", 48*time.Hour)

	ctx := context.Background()
	key := "test-key-1"
	orderID := "order-123"

	created, err := s.CreateIfNotExists(ctx, key, orderID)
	if err != nil {
		t.Fatalf("CreateIfNotExists error: %v", err)
	}
	if !created {
		t.Fatalf("expected created=true")
	}

	// second create should return created=false (exists)
	created2, err := s.CreateIfNotExists(ctx, key, orderID)
	if err != nil {
		t.Fatalf("second CreateIfNotExists error: %v", err)
	}
	if created2 {
		t.Fatalf("expected created=false on duplicate create")
	}

	// Get the record
	rec, err := s.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if rec == nil {
		t.Fatalf("expected record, got nil")
	}
	if rec.Status != StatusInProgress {
		t.Fatalf("expected IN_PROGRESS, got %s", rec.Status)
	}
	if rec.OrderID != orderID {
		t.Fatalf("order id mismatch")
	}

	// Mark done
	err = s.MarkDone(ctx, key, "{\"ok\":true}", 201)
	if err != nil {
		t.Fatalf("MarkDone error: %v", err)
	}

	// Read raw item from mock to assert updated fields
	item := mock.table[key]
	if item == nil {
		t.Fatalf("mock item missing")
	}
	// verify status
	if st, ok := item["status"].(*types.AttributeValueMemberS); !ok || st.Value != StatusDone {
		t.Fatalf("status not updated to DONE, got %+v", item["status"])
	}
	// test response body
	if rb, ok := item["response_body"].(*types.AttributeValueMemberS); !ok || rb.Value != "{\"ok\":true}" {
		t.Fatalf("response_body not set correctly: %+v", item["response_body"])
	}

	// MarkFailed (should overwrite status)
	err = s.MarkFailed(ctx, key, "failed-reason")
	if err != nil {
		t.Fatalf("MarkFailed error: %v", err)
	}
	item2 := mock.table[key]
	if item2 == nil {
		t.Fatalf("mock item missing after mark failed")
	}
	if st, ok := item2["status"].(*types.AttributeValueMemberS); !ok || st.Value != StatusFailed {
		t.Fatalf("status not updated to FAILED, got %+v", item2["status"])
	}
	if n, ok := item2["note"].(*types.AttributeValueMemberS); !ok || n.Value != "failed-reason" {
		t.Fatalf("note not set, got %+v", item2["note"])
	}
}

func TestAttributevalueMarshal_Unmarshal(t *testing.T) {
	// ensure our types marshal/unmarshal cleanly
	rec := IdempotencyRecord{
		IdempotencyKey: "k1",
		Status:         StatusInProgress,
		OrderID:        "o1",
		CreatedAt:      time.Now().Round(time.Second),
		UpdatedAt:      time.Now().Round(time.Second),
		ExpiresAt:      time.Now().Add(24 * time.Hour).Unix(),
	}
	m, err := attributevalue.MarshalMap(rec)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out IdempotencyRecord
	if err := attributevalue.UnmarshalMap(m, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.IdempotencyKey != rec.IdempotencyKey {
		t.Fatalf("unmarshal mismatch")
	}
}

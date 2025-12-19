package validation

import (
	"testing"
	"time"
)

func TestCreateOrderRequest_Valid(t *testing.T) {
	v := New()

	now := time.Now()
	req := CreateOrderRequest{
		CustomerID: "cust-123",
		Items: []Item{
			{SKU: "sku-1", Quantity: 2, Price: 10.0},
			{SKU: "sku-2", Quantity: 1, Price: 5.5},
		},
		Amount:    25.5, // 2*10 + 1*5.5 = 25.5
		Metadata:  map[string]interface{}{"note": "test"},
		CreatedAt: &now,
	}

	if err := v.Struct(req); err != nil {
		t.Fatalf("expected valid, got error: %v", err)
	}
}

func TestCreateOrderRequest_InvalidAmountMismatch(t *testing.T) {
	v := New()

	req := CreateOrderRequest{
		CustomerID: "cust-123",
		Items: []Item{
			{SKU: "sku-1", Quantity: 1, Price: 10.0},
		},
		Amount: 9.99, // mismatch
	}

	if err := v.Struct(req); err == nil {
		t.Fatal("expected validation error for amount mismatch, got nil")
	}
}

func TestCreateOrderRequest_MissingFields(t *testing.T) {
	v := New()

	req := CreateOrderRequest{
		// CustomerID missing
		Items:  []Item{},
		Amount: 0,
	}

	if err := v.Struct(req); err == nil {
		t.Fatal("expected validation errors for missing required fields, got nil")
	}
}

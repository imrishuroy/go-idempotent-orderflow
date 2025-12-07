package validation

import "time"

// Item represents a single order line item.
type Item struct {
	SKU      string  `json:"sku" validate:"required"`            // stock keeping unit
	Quantity int     `json:"quantity" validate:"required,min=1"` // must be >= 1
	Price    float64 `json:"price" validate:"required,gt=0"`     // price per unit
}

// CreateOrderRequest is the payload for POST /orders
type CreateOrderRequest struct {
	CustomerID string                 `json:"customer_id" validate:"required"`      // business id for customer
	Items      []Item                 `json:"items" validate:"required,min=1,dive"` // at least one item
	Amount     float64                `json:"amount" validate:"required,gt=0"`      // total amount client claims
	Metadata   map[string]interface{} `json:"metadata,omitempty"`                   // optional free-form metadata
	CreatedAt  *time.Time             `json:"created_at,omitempty"`                 // optional client timestamp
}

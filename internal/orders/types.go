package orders

import "time"

// Order statuses
const (
	StatusPending    = "PENDING"
	StatusProcessing = "PROCESSING"
	StatusCompleted  = "COMPLETED"
	StatusFailed     = "FAILED"
)

// Order represents the item stored in the Orders DynamoDB table.
type Order struct {
	OrderID    string                   `dynamodbav:"order_id"`              // PK
	CustomerID string                   `dynamodbav:"customer_id,omitempty"` // customer reference
	Status     string                   `dynamodbav:"status"`                // PENDING | PROCESSING | COMPLETED | FAILED
	Amount     float64                  `dynamodbav:"amount"`
	Items      []map[string]interface{} `dynamodbav:"items,omitempty"` // flexible storage; can be refined
	Metadata   map[string]interface{}   `dynamodbav:"metadata,omitempty"`
	CreatedAt  time.Time                `dynamodbav:"created_at"`
	UpdatedAt  time.Time                `dynamodbav:"updated_at"`
	Attempts   int                      `dynamodbav:"attempts,omitempty"`
}

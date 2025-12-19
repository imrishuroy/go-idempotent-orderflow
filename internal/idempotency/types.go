package idempotency

import "time"

// Status values for idempotency entries
const (
	StatusInProgress = "IN_PROGRESS"
	StatusDone       = "DONE"
	StatusFailed     = "FAILED"
)

// IdempotencyRecord is the shape persisted in the idempotency DynamoDB table.
type IdempotencyRecord struct {
	IdempotencyKey string    `dynamodbav:"idempotency_key"` // PK
	Status         string    `dynamodbav:"status"`
	OrderID        string    `dynamodbav:"order_id,omitempty"`
	ResponseBody   string    `dynamodbav:"response_body,omitempty"`   // small responses only; else use S3 pointer
	ResponseStatus int       `dynamodbav:"response_status,omitempty"` // e.g., 201
	CreatedAt      time.Time `dynamodbav:"created_at"`
	UpdatedAt      time.Time `dynamodbav:"updated_at"`
	ExpiresAt      int64     `dynamodbav:"expires_at"` // TTL epoch seconds
	Note           string    `dynamodbav:"note,omitempty"`
}

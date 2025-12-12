package worker

// WorkerMessage is the payload sent from API -> SQS -> Worker.
type WorkerMessage struct {
	OrderID        string `json:"order_id"`
	IdempotencyKey string `json:"idempotency_key"`
	CorrelationID  string `json:"correlation_id,omitempty"`
}

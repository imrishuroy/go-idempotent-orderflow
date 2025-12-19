package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"github.com/imrishuroy/go-idempotent-orderflow/internal/aws"
	"github.com/imrishuroy/go-idempotent-orderflow/internal/idempotency"
	"github.com/imrishuroy/go-idempotent-orderflow/internal/orders"
)

// Processor handles SQS messages and performs order lifecycle transitions.
type Processor struct {
	dynamo         aws.DynamoDBAPI
	idempotencyTbl string
	ordersTbl      string
	idempStore     *idempotency.Store
	orderStore     *orders.Store
}

// NewProcessor creates a new worker processor with AWS clients injected.
func NewProcessor(clients *aws.AWSClients, idempTable, ordersTable string) *Processor {
	return &Processor{
		dynamo:         clients.DynamoDB,
		idempotencyTbl: idempTable,
		ordersTbl:      ordersTable,
		idempStore:     idempotency.NewStore(clients.DynamoDB, idempTable, 48*time.Hour),
		orderStore:     orders.NewStore(clients.DynamoDB, ordersTable),
	}
}

// Handle receives an SQS batch event and processes each message.
func (p *Processor) Handle(ctx context.Context, ev events.SQSEvent) error {
	for _, rec := range ev.Records {
		if err := p.processMessage(ctx, rec); err != nil {
			// Return error: Lambda will retry. If failed too many times, message goes to DLQ.
			log.Printf("worker error: %v", err)
			return err
		}
	}
	return nil
}

func (p *Processor) processMessage(ctx context.Context, rec events.SQSMessage) error {
	var msg WorkerMessage
	if err := json.Unmarshal([]byte(rec.Body), &msg); err != nil {
		return fmt.Errorf("invalid message body: %w", err)
	}

	log.Printf("[worker] received order=%s idempotency_key=%s corr=%s",
		msg.OrderID, msg.IdempotencyKey, msg.CorrelationID)

	// Step 1: Read the current order
	order, err := p.orderStore.Get(ctx, msg.OrderID)
	if err != nil {
		return fmt.Errorf("failed to fetch order: %w", err)
	}
	if order == nil {
		// Should never happen — DLQ if it does
		return fmt.Errorf("order not found: %s", msg.OrderID)
	}

	// Step 2: Move PENDING -> PROCESSING (idempotent)
	err = p.orderStore.UpdateStatus(ctx, msg.OrderID, orders.StatusPending, orders.StatusProcessing)
	if err == orders.ErrStatusMismatch {
		// Already processed or competing worker:
		// If already COMPLETED -> treat as success.
		// If already FAILED -> fail permanently.
		// If already PROCESSING -> another worker took it — return nil to swallow duplicated messages.
		o2, _ := p.orderStore.Get(ctx, msg.OrderID)
		switch o2.Status {
		case orders.StatusCompleted:
			log.Printf("[worker] already completed order=%s", msg.OrderID)
			return nil
		case orders.StatusFailed:
			return fmt.Errorf("order=%s is already FAILED", msg.OrderID)
		case orders.StatusProcessing:
			log.Printf("[worker] duplicate processing event for order=%s", msg.OrderID)
			return nil
		default:
			return fmt.Errorf("unexpected status for order=%s: %s", msg.OrderID, o2.Status)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to update status to PROCESSING: %w", err)
	}

	// Step 3: Do actual work (simulate for now)
	log.Printf("[worker] processing business logic for order=%s", msg.OrderID)
	time.Sleep(200 * time.Millisecond) // simulate processing work

	// Step 4: Complete order: PROCESSING -> COMPLETED
	err = p.orderStore.UpdateStatus(ctx, msg.OrderID, orders.StatusProcessing, orders.StatusCompleted)
	if err != nil {
		return fmt.Errorf("failed to update status to COMPLETED: %w", err)
	}

	// Step 5: Mark idempotency DONE (API created the record)
	response := fmt.Sprintf(`{"order_id":"%s","status":"COMPLETED"}`, msg.OrderID)
	if err := p.idempStore.MarkDone(ctx, msg.IdempotencyKey, response, 200); err != nil {
		return fmt.Errorf("failed to update idempotency: %w", err)
	}

	log.Printf("[worker] completed order=%s", msg.OrderID)
	return nil
}

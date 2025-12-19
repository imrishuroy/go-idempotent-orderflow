package worker

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)



func handleSQSEvent(ctx context.Context, event events.SQSEvent) error {
	log.Printf("received %d SQS messages", len(event.Records))
	for _, r := range event.Records {
		var msg WorkerMessage
		if err := json.Unmarshal([]byte(r.Body), &msg); err != nil {
			log.Printf("failed to unmarshal message body: %v, body: %s", err, r.Body)
			// return error so SQS/Lambda runtime will handle retries
			return err
		}
		// TODO: replace with actual idempotency & order processing logic
		log.Printf("processing order_id=%s idempotency_key=%s", msg.OrderID, msg.IdempotencyKey)
	}
	return nil
}

func main() {
	// If RUN_LOCAL=true, we can optionally simulate a single SQS event for local testing.
	if os.Getenv("RUN_LOCAL") == "true" {
		// Local testing helper: simulate an event using environment variables
		testBody := os.Getenv("LOCAL_SQS_BODY")
		if testBody == "" {
			testBody = `{"order_id":"local-order-1","idempotency_key":"local-key-1"}`
		}
		event := events.SQSEvent{
			Records: []events.SQSMessage{
				{
					Body: testBody,
				},
			},
		}
		if err := handleSQSEvent(context.Background(), event); err != nil {
			log.Fatalf("local handler error: %v", err)
		}
		return
	}

	lambda.Start(handleSQSEvent)
}

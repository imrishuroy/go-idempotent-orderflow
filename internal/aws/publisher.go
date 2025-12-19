package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// Publisher wraps an SQS client and a queue URL.
type Publisher struct {
	SQS      SQSAPI
	QueueURL string
}

// NewPublisher returns a Publisher bound to a queue URL.
func NewPublisher(sqsClient SQSAPI, queueURL string) *Publisher {
	return &Publisher{
		SQS:      sqsClient,
		QueueURL: queueURL,
	}
}

// SendOrderMessage sends an order message to SQS. messageBody should be a JSON string.
// attributes map[string]string -> sent as MessageAttributes.
func (p *Publisher) SendOrderMessage(ctx context.Context, messageBody string, attributes map[string]string) error {
	input := &sqs.SendMessageInput{
		QueueUrl:    &p.QueueURL,
		MessageBody: &messageBody,
	}
	if len(attributes) > 0 {
		msgAttrs := map[string]sqstypes.MessageAttributeValue{}
		for k, v := range attributes {
			// using string type for all attrs
			msgAttrs[k] = sqstypes.MessageAttributeValue{
				DataType:    awsString("String"),
				StringValue: &v,
			}
		}
		input.MessageAttributes = msgAttrs
	}

	_, err := p.SQS.SendMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
}

// awsString helper
func awsString(s string) *string { return &s }

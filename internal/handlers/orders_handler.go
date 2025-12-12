package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/imrishuroy/go-idempotent-orderflow/internal/aws"
	"github.com/imrishuroy/go-idempotent-orderflow/internal/idempotency"
	"github.com/imrishuroy/go-idempotent-orderflow/internal/orders"
	"github.com/imrishuroy/go-idempotent-orderflow/internal/validation"
)

// HandlerConfig groups dependencies for the orders handler.
type HandlerConfig struct {
	DynamoDBClient   aws.DynamoDBAPI
	SQSClient        aws.SQSAPI
	IdempotencyTable string
	OrdersTable      string
	QueueURL         string
	TTLWindow        time.Duration
}

// RegisterOrdersRoutes registers routes for order API.
func RegisterOrdersRoutes(r *gin.Engine, cfg HandlerConfig) {
	v := validation.New()
	idempStore := idempotency.NewStore(cfg.DynamoDBClient, cfg.IdempotencyTable, cfg.TTLWindow)
	ordersStore := orders.NewStore(cfg.DynamoDBClient, cfg.OrdersTable)
	publisher := aws.NewPublisher(cfg.SQSClient, cfg.QueueURL)

	r.POST("/orders", func(c *gin.Context) {
		ctx := c.Request.Context()

		// Bind + validate request
		var req validation.CreateOrderRequest
		if err := validation.BindAndValidate(c, &req, v); err != nil {
			// BindAndValidate already wrote a 400
			return
		}

		// Require idempotency key header
		idempKey := c.GetHeader("Idempotency-Key")
		if idempKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing_idempotency_key"})
			return
		}

		// Generate order id
		orderID := uuid.NewString()

		// Build idempotency item (map) - lightweight
		now := time.Now().UTC()
		idempItem := map[string]interface{}{
			"idempotency_key": idempKey,
			"status":          idempotency.StatusInProgress,
			"created_at":      now.Format(time.RFC3339),
			"updated_at":      now.Format(time.RFC3339),
			"order_id":        orderID,
		}

		// Build order object
		order := orders.Order{
			OrderID:    orderID,
			CustomerID: req.CustomerID,
			Status:     orders.StatusPending,
			Amount:     req.Amount,
			Metadata:   req.Metadata,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		// NOTE: convert items to generic representation
		items := make([]map[string]interface{}, 0, len(req.Items))
		for _, it := range req.Items {
			items = append(items, map[string]interface{}{
				"sku":      it.SKU,
				"quantity": it.Quantity,
				"price":    it.Price,
			})
		}
		order.Items = items

		// Attempt the transact write to create idempotency + order atomically
		err := ordersStore.CreateWithIdempotencyTransaction(ctx, cfg.DynamoDBClient, cfg.IdempotencyTable, idempItem, order, cfg.TTLWindow)
		if err != nil {
			// If transaction failed because idempotency exists, fetch idempotency record and return stored response or 202
			// Detect TransactionCanceledException by string or wrapper (our Create method wraps with message)
			// Best-effort: call idempStore.Get and decide
			rec, getErr := idempStore.Get(ctx, idempKey)
			if getErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "idempotency_check_failed", "detail": getErr.Error()})
				return
			}
			if rec == nil {
				// Unexpected: transaction failed but no record found
				c.JSON(http.StatusInternalServerError, gin.H{"error": "transaction_failed_no_idempotency_record", "detail": err.Error()})
				return
			}
			switch rec.Status {
			case idempotency.StatusDone:
				// return stored response if present
				if rec.ResponseBody != "" {
					var body interface{}
					if derr := json.Unmarshal([]byte(rec.ResponseBody), &body); derr == nil {
						c.Data(rec.ResponseStatus, "application/json", []byte(rec.ResponseBody))
						return
					}
					// if not JSON, just return as string
					c.JSON(rec.ResponseStatus, gin.H{"response": rec.ResponseBody})
					return
				}
				// if no response body stored, return 200 with order_id
				c.JSON(http.StatusOK, gin.H{"order_id": rec.OrderID})
				return
			case idempotency.StatusInProgress:
				c.JSON(http.StatusAccepted, gin.H{"message": "request already in progress", "order_id": rec.OrderID})
				return
			case idempotency.StatusFailed:
				// let client retry
				c.JSON(http.StatusInternalServerError, gin.H{"error": "previous_attempt_failed", "order_id": rec.OrderID})
				return
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "unknown_idempotency_status"})
				return
			}
		}

		// Successfully created atomic records; now send SQS message. If SQS send fails we mark idempotency FAILED.
		// Build message body
		msgPayload := map[string]string{
			"order_id":        orderID,
			"idempotency_key": idempKey,
		}
		payloadBytes, _ := json.Marshal(msgPayload)

		attrs := map[string]string{
			"idempotency_key": idempKey,
			"order_id":        orderID,
			"correlation_id":  c.GetHeader("X-Request-Id"),
		}

		if err := publisher.SendOrderMessage(ctx, string(payloadBytes), attrs); err != nil {
			// mark idempotency failed so client can retry; attempt to set note
			_ = idempStore.MarkFailed(ctx, idempKey, fmt.Sprintf("sqs_send_failed: %v", err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "enqueue_failed", "detail": err.Error()})
			return
		}

		// Success
		// Optionally, we can store a minimal response in idempotency to return for duplicates
		responseBody, _ := json.Marshal(gin.H{"order_id": orderID, "status": "PENDING"})
		_ = idempStore.MarkDone(ctx, idempKey, string(responseBody), http.StatusCreated)

		c.Header("Location", fmt.Sprintf("/orders/%s", orderID))
		c.JSON(http.StatusCreated, gin.H{"order_id": orderID, "status": "PENDING"})
	})
}

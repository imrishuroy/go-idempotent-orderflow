package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
	"github.com/imrishuroy/go-idempotent-orderflow/internal/aws"
	"github.com/imrishuroy/go-idempotent-orderflow/internal/handlers"
)

func setupRouter(cfg handlers.HandlerConfig) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	handlers.RegisterOrdersRoutes(r, cfg)

	return r
}

func main() {
	clients, err := aws.NewAWSClients(context.Background())

	if err != nil {
		log.Fatalf("failed to init aws clients: %v", err)
	}

	cfg := handlers.HandlerConfig{
		DynamoDBClient:   clients.DynamoDB,
		SQSClient:        clients.SQS,
		IdempotencyTable: os.Getenv("IDEMPOTENCY_TABLE"),
		OrdersTable:      os.Getenv("ORDERS_TABLE"),
		QueueURL:         os.Getenv("ORDERS_QUEUE_URL"),
		TTLWindow:        48 * time.Hour,
	}

	r := setupRouter(cfg)

	// if environment variable RUN_LOCAL is set to "true", run local HTTP server for development.
	if os.Getenv("RUN_LOCAL") == "true" {
		addr := ":8080"
		log.Printf("running local server on %s", addr)
		if err := r.Run(addr); err != nil {
			log.Fatalf("failed to run local server: %v", err)
		}
		return
	}

	// lambda adapter
	adapter := ginadapter.New(r)

	lambda.Start(func(ctx context.Context, req events.APIGatewayProxyRequest) (interface{}, error) {
		// the adapter handles proxying; use adapter.ProxyWithContext for proper context propagation
		return adapter.ProxyWithContext(ctx, req)
	})
}

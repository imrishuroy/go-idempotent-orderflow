package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/orders", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"message": "orders endpoint not implemented yet - proceed to Step 2 to implement",
		})
	})

	return r
}

func main() {
	r := setupRouter()

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

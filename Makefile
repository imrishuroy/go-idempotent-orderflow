.PHONY: build-api build-worker run-local-api run-local-worker test lint

BINARY_NAME_API=api
BINARY_NAME_WORKER=worker

build-api:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/$(BINARY_NAME_API) ./cmd/api

build-worker:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/$(BINARY_NAME_WORKER) ./cmd/worker

run-local-api:
	RUN_LOCAL=true go run ./cmd/api

run-local-worker:
	RUN_LOCAL=true go run ./cmd/worker

test:
	go test ./... -v

lint:
	# requires golangci-lint installed
	golangci-lint run

# Order Processor (Learning Project)

This repository is a learning project to implement an idempotent order creation & processing system using Go, AWS Lambda, DynamoDB, SQS and Terraform.

This first step contains the project skeleton and CI. Subsequent steps will add validation, idempotency logic, DynamoDB access, SQS publishing, worker processing, Terraform modules, and tests.

## Local dev

Run API locally:
```bash
# runs Gin HTTP server on :8080
make run-local-api
# then
curl -X GET http://localhost:8080/health

.
├─ cmd/
│  ├─ api/
│  │  └─ main.go
│  └─ worker/
│     └─ main.go
├─ internal/
│  └─ placeholders (README note)
├─ .golangci.yml
├─ Makefile
├─ go.mod
├─ go.sum (not included here; generated)
├─ .github/
│  └─ workflows/
│     └─ ci.yml
├─ README.md
└─ tests/
   └─ unit/
      └─ placeholder_test.go

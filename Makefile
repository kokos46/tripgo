-include .env
export

.PHONY: migrate migrate-up migrate-down run lint test test-integration generate

migrate: migrate-up

migrate-up:
	goose -dir ./migrations postgres "$$DATABASE_URL" up

migrate-down:
	goose -dir ./migrations postgres "$$DATABASE_URL" down

test:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

test-integration:
	go test -race -tags=integration ./...

run:
	go run ./cmd/trip-service $(ARGS)

lint:
	golangci-lint run ./...

generate:
	go tool oapi-codegen \
         -generate types,chi-server \
         -package api \
         -o internal/generated/api.gen.go \
         contracts/openapi/trip-service.openapi.yaml
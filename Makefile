include .env
export

.PHONY: migrate-up migrate-down run lint test generate

migrate-up:
	goose -dir ./migrations postgres "$$DATABASE_URL" up

migrate-down:
	goose -dir ./migrations postgres "$$DATABASE_URL" down

test:
	go test -race

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
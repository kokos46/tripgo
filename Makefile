export DATABASE_URL ?= postgres://app:app@localhost:5432/cinema?sslmode=disable

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
	go tool oapi-codegen -generate types,std-http -package openapi -o ./api/trip_service.gen.go ./contracts/openapi/trip-service.openapi.yaml
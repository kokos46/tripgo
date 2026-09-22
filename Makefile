export DATABASE_URL ?= postgres://app:app@localhost:5432/cinema?sslmode=disable

.PHONY: migrate-up migrate-down run lint test

migrate-up:
	goose -dir ./migrations postgres "$$DATABASE_URL" up

migrate-down:
	goose -dir ./migrations postgres "$$DATABASE_URL" down

test:
	go test -race

run:
	go run ./cmd/app $(ARGS)

lint:
	golangci-lint run ./...
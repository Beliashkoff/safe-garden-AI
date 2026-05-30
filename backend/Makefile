.PHONY: dev worker-dev cleanup-run test test-integration lint build worker-build cleanup-build migrate-up migrate-down sqlc-gen clean

dev:
	docker compose up -d
	air

worker-dev:
	go run ./cmd/llmworker

cleanup-run:
	go run ./cmd/cleanup

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

lint:
	golangci-lint run ./...

build:
	go build -ldflags="-s -w" -o bin/api ./cmd/api

worker-build:
	go build -ldflags="-s -w" -o bin/llmworker ./cmd/llmworker

cleanup-build:
	go build -ldflags="-s -w" -o bin/cleanup ./cmd/cleanup

migrate-up:
	@goose -dir migrations postgres "$$POSTGRES_DSN" up

migrate-down:
	@goose -dir migrations postgres "$$POSTGRES_DSN" down

sqlc-gen:
	sqlc generate

clean:
	rm -rf bin/ tmp/

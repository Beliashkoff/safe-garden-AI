.PHONY: dev worker-dev cleanup-run test test-integration lint build worker-build cleanup-build migrate-up migrate-down sqlc-gen clean loadtest

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

# Load test /v1/messages at ~50 RPS (ROADMAP §6.2 DoD). Run against the local
# dev stack with LLM_CLIENT_KIND=mock so it exercises the handler/DB/ratelimit/
# SSE path without spending Anthropic tokens. TOKEN is a valid access token for
# a dev user (see infra/loadtest/messages.js header for how to mint one).
loadtest:
	k6 run -e BASE_URL=$${BASE_URL:-http://localhost:8080} -e TOKEN=$$TOKEN ../infra/loadtest/messages.js

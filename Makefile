.PHONY: help tidy fmt vet build test run-api run-cli up down logs clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

tidy: ## Resolve and tidy module dependencies
	go mod tidy

fmt: ## Format code
	go fmt ./...

vet: ## Static checks
	go vet ./...

build: ## Build api and cli binaries into ./bin
	go build -o bin/omni-api ./cmd/api
	go build -o bin/omni ./cmd/cli

test: ## Run unit tests
	go test ./...

test-smoke: ## Run smoke tests against the local stack (requires: make up)
	go test -tags smoke -v -timeout 120s ./test/smoke/...

run-api: ## Run the HTTP API
	go run ./cmd/api

run-cli: ## Run the CLI (pass args after --, e.g. make run-cli -- wallet create)
	go run ./cmd/cli $(filter-out $@,$(MAKECMDGOALS))

up: ## Start local chain nodes + Kafka
	docker compose up -d

down: ## Stop local stack
	docker compose down

logs: ## Tail local stack logs
	docker compose logs -f

clean: ## Remove build artifacts and local data
	rm -rf bin/ data/

%:
	@:

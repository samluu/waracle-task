GOLANGCI_LINT_VERSION := v2.15.0
# Build the linter with this module's toolchain: it refuses to analyse code
# targeting a newer Go than it was compiled with.
GOLANGCI_LINT := GOTOOLCHAIN=$(shell go env GOVERSION) go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: run build test lint fmt generate check-generate up down

run: ## Run the API locally (needs `make up` db and a .env, see .env.example)
	set -a && . ./.env && set +a && go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test -race ./...

lint:
	$(GOLANGCI_LINT) run

fmt:
	$(GOLANGCI_LINT) fmt

generate: ## Regenerate server code from api/openapi.yaml
	go generate ./...

check-generate: generate ## Fail if generated code is out of date with the spec
	git diff --exit-code -- '*.gen.go'

up:
	docker compose up --build -d

down:
	docker compose down

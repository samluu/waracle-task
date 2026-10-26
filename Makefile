GOLANGCI_LINT_VERSION := v2.15.0
# Build the linter with this module's toolchain: it refuses to analyse code
# targeting a newer Go than it was compiled with. Lazily expanded (=) so
# targets that don't lint, like `make up`, don't need Go installed.
GOLANGCI_LINT = GOTOOLCHAIN=$(shell go env GOVERSION) go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: run build test lint fmt generate check-generate up down

run: ## Run the API on the host (needs a .env and the db: `docker compose up -d db`)
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

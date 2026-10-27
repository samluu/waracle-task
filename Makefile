GOLANGCI_LINT_VERSION := v2.15.0
# Build the linter with this module's toolchain: it refuses to analyse code
# targeting a newer Go than it was compiled with. Lazily expanded (=) so
# targets that don't lint, like `make up`, don't need Go installed.
GOLANGCI_LINT = GOTOOLCHAIN=$(shell go env GOVERSION) go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: run build test test-integration lint fmt generate check-generate migrate migrate-status seed reset up down

run: ## Run the API on the host (needs a .env, the db and its schema: `docker compose up -d db && make migrate`)
	set -a && . ./.env && set +a && go run ./cmd/api

build:
	go build -o bin/ ./cmd/api

test:
	go test -race ./...

test-integration: ## Run all tests, including database ones against a separate hotel_test db (needs a .env)
	./scripts/test-integration.sh

lint:
	$(GOLANGCI_LINT) run

fmt:
	$(GOLANGCI_LINT) fmt

generate: ## Regenerate server code from api/openapi.yaml and test mocks from their interfaces
	go generate ./...

# Compares the generated files before and after regenerating, not against git, so it also works with uncommitted spec changes.
check-generate: ## Fail if generated code is out of date with the spec or the mocked interfaces
	@before=$$(find . -name '*.gen.go' -exec shasum {} + | sort); \
	go generate ./...; \
	after=$$(find . -name '*.gen.go' -exec shasum {} + | sort); \
	if [ "$$before" != "$$after" ]; then echo "generated code was out of date (now regenerated): run make generate and commit"; exit 1; fi

migrate: ## Apply pending migrations to the compose db (`make up` also does this before starting the API)
	docker compose run --rm --build migrate up

migrate-status: ## Show which migrations have been applied
	docker compose run --rm --build migrate status

seed: ## Insert the test hotels and rooms through the running API (fails if already seeded; `make reset` first)
	set -a && . ./.env && set +a && curl -sS --fail-with-body -X POST "localhost:$${PORT}/admin/seed" && echo seeded

reset: ## Remove all hotels, rooms and bookings through the running API
	set -a && . ./.env && set +a && curl -sS --fail-with-body -X POST "localhost:$${PORT}/admin/reset" && echo reset

up:
	docker compose up --build -d

down:
	docker compose down

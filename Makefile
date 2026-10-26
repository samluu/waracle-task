GOLANGCI_LINT_VERSION := v2.15.0
# Build the linter with this module's toolchain: it refuses to analyse code
# targeting a newer Go than it was compiled with. Lazily expanded (=) so
# targets that don't lint, like `make up`, don't need Go installed.
GOLANGCI_LINT = GOTOOLCHAIN=$(shell go env GOVERSION) go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: run build test test-integration lint fmt generate check-generate migrate migrate-status seed reset up down

run: ## Run the API on the host (needs a .env, the db and its schema: `docker compose up -d db && make migrate`)
	set -a && . ./.env && set +a && go run ./cmd/api

build:
	go build -o bin/ ./cmd/...

test:
	go test -race ./...

test-integration: migrate ## Run tests against the compose db too (needs a .env)
	set -a && . ./.env && set +a && TEST_DATABASE_URL="$$DATABASE_URL" go test -race -count=1 ./...

lint:
	$(GOLANGCI_LINT) run

fmt:
	$(GOLANGCI_LINT) fmt

generate: ## Regenerate server code from api/openapi.yaml
	go generate ./...

check-generate: generate ## Fail if generated code is out of date with the spec
	git diff --exit-code -- '*.gen.go'

migrate: ## Apply pending migrations to the compose db (`make up` also does this before starting the API)
	docker compose run --rm --build migrate up

migrate-status: ## Show which migrations have been applied
	docker compose run --rm --build migrate status

seed: ## Insert the test hotels, rooms and bookings (fails if already seeded; `make reset` first)
	docker compose run --rm --build --entrypoint /devdata api seed

reset: ## Remove all hotels, rooms and bookings
	docker compose run --rm --build --entrypoint /devdata api reset

up:
	docker compose up --build -d

down:
	docker compose down

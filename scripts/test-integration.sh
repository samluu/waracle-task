#!/bin/sh
# Runs all tests, including the database ones, against a separate hotel_test database in the
# compose Postgres. The tests reset and seed it freely, so the dev database is never touched.
set -eu

set -a
. ./.env
set +a

test_db=hotel_test

docker compose up -d --wait db

# Create the test database on first run.
docker compose exec -T db sh -c \
  "psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -Atc \"SELECT 1 FROM pg_database WHERE datname = '$test_db'\" | grep -q 1 \
   || createdb -U \"\$POSTGRES_USER\" $test_db"

docker compose run --rm --build \
  -e GOOSE_DBSTRING="postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@db:5432/$test_db?sslmode=disable" \
  migrate up

TEST_DATABASE_URL="postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@localhost:5432/$test_db?sslmode=disable" \
  go test -race -count=1 ./...

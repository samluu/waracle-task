# syntax=docker/dockerfile:1

FROM golang:1.27.3-alpine AS build
WORKDIR /src

# goose CLI for migrations, without the drivers for databases we don't use.
RUN CGO_ENABLED=0 go install -trimpath -ldflags="-s -w" \
    -tags='no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb' \
    github.com/pressly/goose/v3/cmd/goose@v3.28.0

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Migration job: `docker compose run migrate <goose command>`, e.g. up, status, down.
FROM gcr.io/distroless/static-debian13:nonroot AS migrate
COPY --from=build /go/bin/goose /goose
COPY migrations /migrations
ENV GOOSE_DRIVER=postgres GOOSE_MIGRATION_DIR=/migrations
USER nonroot:nonroot
ENTRYPOINT ["/goose"]
CMD ["up"]

FROM gcr.io/distroless/static-debian13:nonroot AS api
COPY --from=build /out/api /
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]

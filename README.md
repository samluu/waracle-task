# waracle-task

## Use of AI
- generate a skeleton with AI to setup docker-compose, postgres, the default go web framework, CI/CD etc

## Assumptions

## Requirements

- Docker with Compose v2
- make
- Go 1.27.3+ (optional, only for running or testing outside Docker; an older Go toolchain downloads 1.27.3 automatically)

## Setup

Start server
```
cp .env.example .env
make up
```
This builds the API image and starts it alongside Postgres. The API is at http://localhost:8080:
```
curl localhost:8080/health   # {"status":"ok"}
```
Stop it with `make down`.

Run tests
```
make test
```
Run the linter with `make lint`.

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

## API

The API is defined in [api/openapi.yaml](api/openapi.yaml). With the server running:

- http://localhost:8080/docs: Swagger UI, to browse the endpoints and call them with "Try it out"
- http://localhost:8080/openapi.yaml: the spec the running server was built from, e.g. to import into Postman

The spec is the source of truth: the routes and request/response types in `internal/httpapi/api.gen.go` are generated from it with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen). After changing the spec, run:
```
make generate
```
The build fails if a handler for an operation in the spec is missing, and CI fails if the generated code is out of date with the spec.

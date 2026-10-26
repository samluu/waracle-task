# waracle-task

## Use of AI

- generate a skeleton with AI to setup docker-compose, postgres, the default go web framework, CI/CD etc
- set up the OpenAPI spec, code generation from it, and Swagger UI
- review the repo and explain trade-offs in the generated code

## Assumptions

- the 3 hotel room types implies the capacity, so single is 1, double is 2 and deluxe is 4
- hotels have exactly 6 rooms each
- refunds/cancellations are out of scope
- adding extra people to rooms that aren't full is out of scope
- a booking is only for one hotel and for one room therefore a booking also has a maximum of 4 people. If someone wants to book additional rooms, another booking needs to be made
- check in date is inclusive and check out date is exclusive, they'll be the same if the stay is 1 night
- a booking records the lead guest's name and the number of guests, we don't need to store all the guests
- when searching for a room, the filters are an optional hotel name, number of guests and a date range. The search is applied to all matching hotels and rooms
- when searching for a hotel by name, it has to be an exact match
- hotel names are unique
- seeding gives some fixed data that can be reset. Not generating random data as that's more useful for load testing whilst the fixed data is better for manual tests

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
This builds the image, starts Postgres, applies database migrations, then starts the API at http://localhost:8080:
```
curl localhost:8080/health   # {"status":"ok"}
```
Stop it with `make down`.

Migrations live in [migrations/](migrations) and are run with the [goose](https://github.com/pressly/goose) CLI in a container: `make migrate` applies pending ones to a running database, `make migrate-status` lists what has been applied.

Test data
```
make seed    # 2 hotels ("The Grand Hotel", "Seaside Inn"), 6 rooms each, 5 bookings
make reset   # remove all hotels, rooms and bookings
```
Seeding fails if the data is already there, so run `make reset` first to start again. The bookings have fixed references (`SEED-0001` to `SEED-0005`) and are all in January 2027; see [internal/store/seed.go](internal/store/seed.go).

Run tests
```
make test
```
`make test-integration` also runs the database tests against the compose Postgres (it runs `make migrate` first). Run the linter with `make lint`.

## API

The API is defined in [api/openapi.yaml](api/openapi.yaml). With the server running:

- http://localhost:8080/docs: Swagger UI, to browse the endpoints and call them with "Try it out"
- http://localhost:8080/openapi.yaml: the spec the running server was built from, e.g. to import into Postman

The spec is the source of truth: the routes and request/response types in `internal/httpapi/api.gen.go` are generated from it with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen). After changing the spec, run:
```
make generate
```
The build fails if a handler for an operation in the spec is missing, and CI fails if the generated code is out of date with the spec.

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
- when booking check in has to be between today and a year from now. I believe hotels usually have a limit to how far in advanced you can book.
- when booking, a stay can be at most 30 nights. There are probably local laws restricting how long you can stay in a hotel in one go

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
Seeding fails if the data is already there, so run `make reset` first to start again. The bookings have fixed references, UUIDs `5eed0000-0000-4000-8000-000000000001` to `...005`, and are all in January 2027; see [internal/store/seed.go](internal/store/seed.go).

Run tests
```
make test
```
`make test-integration` also runs the database tests, against a separate `hotel_test` database in the compose Postgres that it creates and migrates, so your dev data is untouched. Run the linter with `make lint`.

## API

| Method | Path | Does |
|---|---|---|
| GET | `/hotels?name=` | Find a hotel by its exact name |
| GET | `/rooms?check_in=&check_out=&guests=[&hotel_name=][&limit=][&cursor=]` | Rooms free for the whole stay that can hold the guests, in every hotel or just the one named. Paged: up to `limit` rooms (default 20, max 100) per page; pass the response's `next_cursor` as `cursor` for the next page |
| POST | `/bookings` | Book a room (`room_id` comes from `GET /rooms`) |
| GET | `/bookings/{reference}` | Booking details |

For example, after `make seed`:
```
curl 'localhost:8080/rooms?check_in=2027-01-11&check_out=2027-01-13&guests=1&hotel_name=The%20Grand%20Hotel'
curl -X POST localhost:8080/bookings -d '{"room_id":2,"guest_name":"Jane Doe","guests":1,"check_in":"2027-01-11","check_out":"2027-01-13"}'
curl localhost:8080/bookings/5eed0000-0000-4000-8000-000000000001
```
A booking that clashes with another is 409, too many guests for the room is 422, and invalid input is 400; errors are `{"error": "..."}`.

The API is defined in [api/openapi.yaml](api/openapi.yaml). With the server running:

- http://localhost:8080/docs: Swagger UI, to browse the endpoints and call them with "Try it out"
- http://localhost:8080/openapi.yaml: the spec the running server was built from, e.g. to import into Postman

The spec is the source of truth: the routes and request/response types in `internal/httpapi/handlers/api.gen.go` are generated from it with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen). After changing the spec, run:
```
make generate
```
The build fails if a handler for an operation in the spec is missing, and CI fails if the generated code is out of date with the spec.

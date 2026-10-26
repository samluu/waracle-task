# waracle-task

## Use of AI

I used Claude Code throughout to do the following:

- generate a skeleton to setup docker-compose, postgres, the default go web framework, CI/CD etc
- generate most of the code, I then followed up on what it outputted and asked for changes where I either found it too convoluted or the feature wasn't needed
- I asked AI about some of the code it generated so I could understand everything

## Assumptions

- the 3 hotel room types implies the capacity, so single is 1, double is 2 and deluxe is 4
- hotels have exactly 6 rooms each, this is only done via seeding. There isn't anything to enforce this elsewhere
- refunds/cancellations are out of scope
- adding extra people to rooms that aren't full is out of scope
- a booking is only for one hotel and for one room therefore a booking also has a maximum of 4 people. If someone wants to book additional rooms, another booking needs to be made
- check in date is inclusive and check out date is exclusive, so a 1 night stay checks out the day after it checks in
- a booking records the lead guest's name and the number of guests, we don't need to store all the guests
- when searching for a room, the date range and number of guests are required and the hotel name is optional. Without a hotel name the search covers every hotel
- searching by hotel name is fuzzy and uses Postgres trigram word similarity
- hotel names are unique
- seeding gives some fixed data that can be reset. Not generating random data as that's more useful for load testing whilst the fixed data is better for manual tests
- seeding is just hotels and rooms, no bookings
- when booking check in has to be between today and a year from now. I believe hotels usually have a limit to how far in advanced you can book.
- when booking, a stay can be at most 30 nights. There are probably local laws restricting how long you can stay in a hotel in one go
- the 2 second delay mentioned in the spec is assumed to simulate a email send delay
- not adding email address for the simulated log email
- if the email buffer is full, an error is logged. There isn't any saga

## Requirements

- Docker with Compose v2
- make
- Go 1.27.3+ (optional, only for running or testing outside Docker; an older Go toolchain downloads 1.27.3 automatically)

## Setup

```
cp .env.example .env
make up      # start Postgres and the API at http://localhost:8080, applying migrations
make seed    # add test data: 2 hotels, 6 rooms each, no bookings
make reset   # remove all data, e.g. to seed again
make down    # stop
```

## API

With the server running, browse and try the endpoints at http://localhost:8080/docs.

The API is defined in [api/openapi.yaml](api/openapi.yaml), and the server code is generated from it with `make generate`.

## Tests

```
make test              # unit tests
make test-integration  # also database tests, in a separate database so your data is untouched
make lint
```

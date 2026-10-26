-- +goose Up
-- btree_gist lets the bookings exclusion constraint mix = on room_id with && on date ranges.
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Reference data, not test data: reset leaves this table alone.
CREATE TABLE room_types (
  type     text PRIMARY KEY,
  capacity int  NOT NULL CHECK (capacity > 0)
);
INSERT INTO room_types (type, capacity) VALUES ('single', 1), ('double', 2), ('deluxe', 4);

CREATE TABLE hotels (
  id   bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  -- Unique, and its index serves the exact-name hotel search (WHERE name = $1).
  name text NOT NULL UNIQUE
);

CREATE TABLE rooms (
  id       bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  hotel_id bigint NOT NULL REFERENCES hotels (id) ON DELETE CASCADE,
  number   text   NOT NULL,
  type     text   NOT NULL REFERENCES room_types (type),
  UNIQUE (hotel_id, number)
);

-- reference is the public booking number and the key. It's random rather than sequential so that,
-- with no auth on the API, bookings can't be enumerated by guessing numbers.
CREATE TABLE bookings (
  reference  text        PRIMARY KEY,
  room_id    bigint      NOT NULL REFERENCES rooms (id) ON DELETE CASCADE,
  guest_name text        NOT NULL,
  guests     int         NOT NULL CHECK (guests > 0),
  check_in   date        NOT NULL,
  check_out  date        NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (check_out > check_in),
  -- A room can't be booked for overlapping nights. Ranges are half-open [check_in, check_out),
  -- so checking out on a day and someone else checking in that day doesn't clash.
  CONSTRAINT bookings_no_overlap
    EXCLUDE USING gist (room_id WITH =, daterange(check_in, check_out, '[)') WITH &&)
);

-- +goose Down
DROP TABLE bookings, rooms, hotels, room_types;

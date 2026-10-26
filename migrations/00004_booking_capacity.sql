-- +goose Up
-- A booking can't have more guests than its room's capacity. Capacity is on room_types, and a CHECK can't
-- look at another table, so a trigger enforces it, raising check_violation with the constraint name
-- bookings_capacity so callers can tell it apart. A room that doesn't exist gives a NULL capacity, which
-- passes here and is left to the room_id foreign key. Changing a room's type or a type's capacity isn't
-- checked against existing bookings.
-- +goose StatementBegin
CREATE FUNCTION bookings_check_capacity() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.guests > (SELECT rt.capacity FROM rooms r JOIN room_types rt ON rt.type = r.type WHERE r.id = NEW.room_id) THEN
    RAISE EXCEPTION 'more guests than the room''s capacity'
      USING ERRCODE = 'check_violation', CONSTRAINT = 'bookings_capacity';
  END IF;
  RETURN NEW;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER bookings_capacity BEFORE INSERT OR UPDATE OF room_id, guests ON bookings
  FOR EACH ROW EXECUTE FUNCTION bookings_check_capacity();

-- +goose Down
DROP TRIGGER bookings_capacity ON bookings;
DROP FUNCTION bookings_check_capacity();

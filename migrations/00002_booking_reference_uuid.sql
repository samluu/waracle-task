-- +goose Up
-- Booking references become database-generated UUIDs: 122 random bits, so with no auth on the API they
-- can't be guessed, and no collision handling is needed. Existing references that aren't UUIDs (e.g. old
-- seed data) are given new ones rather than failing the migration.
ALTER TABLE bookings
  ALTER COLUMN reference TYPE uuid USING (
    CASE WHEN reference ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
         THEN reference::uuid
         ELSE gen_random_uuid()
    END),
  ALTER COLUMN reference SET DEFAULT gen_random_uuid();

-- +goose Down
ALTER TABLE bookings
  ALTER COLUMN reference DROP DEFAULT,
  ALTER COLUMN reference TYPE text USING reference::text;

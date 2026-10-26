-- +goose Up
-- Hotel names are searched fuzzily with pg_trgm word similarity (search <% name), which matches whole or
-- partial words and small misspellings, ignoring case. A trigram index serves it; hotels_name_key still
-- serves sorting and the room search's paging.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX hotels_name_trgm_idx ON hotels USING gin (name gin_trgm_ops);

-- +goose Down
DROP INDEX hotels_name_trgm_idx;
DROP EXTENSION IF EXISTS pg_trgm;

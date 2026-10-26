package store

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrRoomNotFound    = errors.New("room not found")
	ErrRoomUnavailable = errors.New("room is already booked for some of those nights")
	ErrOverCapacity    = errors.New("more guests than the room's capacity")
)

// Postgres SQLSTATE codes the schema's constraints raise.
const (
	codeForeignKeyViolation = "23503"
	codeUniqueViolation     = "23505"
	codeCheckViolation      = "23514"
	codeExclusionViolation  = "23P01"
)

// pgConstraintError reports whether err is a Postgres error with the given SQLSTATE code,
// and if so the name of the constraint that raised it.
func pgConstraintError(err error, code string) (constraint string, ok bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == code {
		return pgErr.ConstraintName, true
	}
	return "", false
}

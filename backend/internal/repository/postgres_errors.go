package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func isUniqueViolation(err error) bool {
	var pgxError *pgconn.PgError
	if errors.As(err, &pgxError) && pgxError.Code == "23505" {
		return true
	}
	return false
}

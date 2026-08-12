package repository

import (
	"errors"

	"github.com/jackc/pgconn"
	"github.com/lib/pq"
)

func isUniqueViolation(err error) bool {
	var pgxError *pgconn.PgError
	if errors.As(err, &pgxError) && pgxError.Code == "23505" {
		return true
	}
	var pqError *pq.Error
	return errors.As(err, &pqError) && pqError.Code == "23505"
}

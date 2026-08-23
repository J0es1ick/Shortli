package repository

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "pgx", err: &pgconn.PgError{Code: "23505"}, want: true},
		{name: "other postgres error", err: &pgconn.PgError{Code: "23503"}, want: false},
		{name: "generic", err: errors.New("duplicate-ish text"), want: false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := isUniqueViolation(testCase.err); got != testCase.want {
				t.Fatalf("isUniqueViolation() = %v, want %v", got, testCase.want)
			}
		})
	}
}

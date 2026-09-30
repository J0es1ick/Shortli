package testutil

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	"github.com/jmoiron/sqlx"
)

type SQLStub struct {
	Query func(context.Context, string, []driver.NamedValue) (driver.Rows, error)
	Exec  func(context.Context, string, []driver.NamedValue) (driver.Result, error)
}

func (s *SQLStub) DB(t testing.TB) *sqlx.DB {
	t.Helper()
	db := sqlx.NewDb(sql.OpenDB(s), "stub")
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func (s *SQLStub) Connect(context.Context) (driver.Conn, error) { return s, nil }
func (s *SQLStub) Driver() driver.Driver                        { return s }
func (s *SQLStub) Open(string) (driver.Conn, error)             { return s, nil }
func (s *SQLStub) Close() error                                 { return nil }
func (s *SQLStub) Prepare(string) (driver.Stmt, error)          { return nil, errors.New("unexpected prepare") }
func (s *SQLStub) Begin() (driver.Tx, error)                    { return nil, errors.New("unexpected transaction") }
func (s *SQLStub) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if s.Query == nil {
		return nil, errors.New("unexpected query")
	}
	return s.Query(ctx, query, args)
}
func (s *SQLStub) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if s.Exec == nil {
		return nil, errors.New("unexpected exec")
	}
	return s.Exec(ctx, query, args)
}

type Rows struct {
	Names  []string
	Values [][]driver.Value
}

func (r *Rows) Columns() []string { return r.Names }
func (r *Rows) Close() error      { return nil }
func (r *Rows) Next(dest []driver.Value) error {
	if len(r.Values) == 0 {
		return io.EOF
	}
	copy(dest, r.Values[0])
	r.Values = r.Values[1:]
	return nil
}

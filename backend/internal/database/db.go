package database

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/J0es1ick/shortli/internal/config"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
)

type Database struct {
	DB *sqlx.DB
}

func DBInit(cfg *config.Config) (*Database, error) {
	databaseURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.Database.User, cfg.Database.Password),
		Host:   cfg.Database.Host + ":" + cfg.Database.Port,
		Path:   cfg.Database.Name,
	}
	query := databaseURL.Query()
	query.Set("sslmode", cfg.Database.SSLMode)
	query.Set("connect_timeout", fmt.Sprintf("%d", cfg.Database.ConnectTimeout))
	query.Set("statement_timeout", fmt.Sprintf("%d", cfg.Database.StatementTimeout))
	query.Set("lock_timeout", fmt.Sprintf("%d", cfg.Database.LockTimeout))
	databaseURL.RawQuery = query.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Database.ConnectTimeout)*time.Second)
	defer cancel()
	conn, err := sqlx.ConnectContext(ctx, "pgx", databaseURL.String())
	if err != nil {
		return nil, fmt.Errorf("can't connect to pg instance, %v", err)
	}
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(10)
	conn.SetConnMaxLifetime(30 * time.Minute)
	conn.SetConnMaxIdleTime(5 * time.Minute)

	return &Database{DB: conn}, nil
}

func (d *Database) Close() error {
	return d.DB.Close()
}

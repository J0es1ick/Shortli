package database

import (
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
	databaseURL.RawQuery = query.Encode()

	conn, err := sqlx.Connect("pgx", databaseURL.String())
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

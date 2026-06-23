package models

import "time"

type User struct {
	ID           int       `db:"user_id" json:"user_id"`
	Email        string    `db:"email" json:"email"`
	// Username     string    `db:"username" json:"username"`
	PasswordHash string    `db:"password_hash" json:"-"`
	IsAdmin      bool      `db:"is_admin" json:"is_admin"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}
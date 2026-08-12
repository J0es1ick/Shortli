package models

import "time"

type APIKey struct {
	ID         int64      `db:"key_id" json:"key_id"`
	UserID     int        `db:"user_id" json:"user_id,omitempty"`
	Name       string     `db:"name" json:"name"`
	Prefix     string     `db:"key_prefix" json:"prefix"`
	Hash       string     `db:"key_hash" json:"-"`
	LastUsedAt *time.Time `db:"last_used_at" json:"last_used_at"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	RevokedAt  *time.Time `db:"revoked_at" json:"revoked_at,omitempty"`
}

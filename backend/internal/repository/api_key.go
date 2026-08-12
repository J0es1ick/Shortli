package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/J0es1ick/shortli/internal/models"
	"github.com/jmoiron/sqlx"
)

type APIKeyRepository struct{ db *sqlx.DB }

func NewAPIKeyRepository(db *sqlx.DB) *APIKeyRepository { return &APIKeyRepository{db: db} }

func (r *APIKeyRepository) CountActive(ctx context.Context, userID int) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM api_key WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return count, err
}

func (r *APIKeyRepository) Create(ctx context.Context, key *models.APIKey) error {
	return r.db.QueryRowxContext(ctx, `
		INSERT INTO api_key (user_id, name, key_prefix, key_hash, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING key_id, created_at
	`, key.UserID, key.Name, key.Prefix, key.Hash).Scan(&key.ID, &key.CreatedAt)
}

func (r *APIKeyRepository) List(ctx context.Context, userID int) ([]models.APIKey, error) {
	keys := []models.APIKey{}
	err := r.db.SelectContext(ctx, &keys, `
		SELECT key_id, user_id, name, key_prefix, '' AS key_hash, last_used_at, created_at, revoked_at
		FROM api_key WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	return keys, err
}

func (r *APIKeyRepository) Revoke(ctx context.Context, id int64, userID int) error {
	result, err := r.db.ExecContext(ctx, `UPDATE api_key SET revoked_at = NOW() WHERE key_id = $1 AND user_id = $2 AND revoked_at IS NULL`, id, userID)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *APIKeyRepository) Authenticate(ctx context.Context, raw string) (*models.APIKey, error) {
	hash := sha256.Sum256([]byte(raw))
	encoded := hex.EncodeToString(hash[:])
	key := &models.APIKey{}
	err := r.db.GetContext(ctx, key, `
		SELECT key_id, user_id, name, key_prefix, key_hash, last_used_at, created_at, revoked_at
		FROM api_key WHERE key_hash = $1 AND revoked_at IS NULL
	`, encoded)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("authenticate api key: %w", err)
	}
	return key, nil
}

func (r *APIKeyRepository) Touch(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE api_key SET last_used_at = $1
		WHERE key_id = $2 AND (last_used_at IS NULL OR last_used_at < $3)
	`, time.Now().UTC(), id, time.Now().UTC().Add(-5*time.Minute))
	return err
}

package repository

import (
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

func (r *APIKeyRepository) CountActive(userID int) (int, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM api_key WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return count, err
}

func (r *APIKeyRepository) Create(key *models.APIKey) error {
	return r.db.QueryRowx(`
		INSERT INTO api_key (user_id, name, key_prefix, key_hash, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING key_id, created_at
	`, key.UserID, key.Name, key.Prefix, key.Hash).Scan(&key.ID, &key.CreatedAt)
}

func (r *APIKeyRepository) List(userID int) ([]models.APIKey, error) {
	keys := []models.APIKey{}
	err := r.db.Select(&keys, `
		SELECT key_id, user_id, name, key_prefix, '' AS key_hash, last_used_at, created_at, revoked_at
		FROM api_key WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	return keys, err
}

func (r *APIKeyRepository) Revoke(id int64, userID int) error {
	result, err := r.db.Exec(`UPDATE api_key SET revoked_at = NOW() WHERE key_id = $1 AND user_id = $2 AND revoked_at IS NULL`, id, userID)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *APIKeyRepository) Authenticate(raw string) (*models.APIKey, error) {
	hash := sha256.Sum256([]byte(raw))
	encoded := hex.EncodeToString(hash[:])
	key := &models.APIKey{}
	err := r.db.Get(key, `
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

func (r *APIKeyRepository) Touch(id int64) {
	_, _ = r.db.Exec(`UPDATE api_key SET last_used_at = $1 WHERE key_id = $2`, time.Now().UTC(), id)
}

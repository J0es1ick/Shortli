package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"time"

	"github.com/J0es1ick/shortli/internal/models"
	"github.com/jmoiron/sqlx"
)

type SessionRepository struct {
	db *sqlx.DB
}

func NewSessionRepository(db *sqlx.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) CreateSession(ctx context.Context, session *models.Session) error {
	query := `
		INSERT INTO session_info (session_id, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (session_id) DO UPDATE SET expires_at = EXCLUDED.expires_at
	`
	_, err := r.db.ExecContext(ctx,
		query,
		hashSessionID(session.ID),
		session.UserID,
		session.ExpiresAt,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("create session error: %v", err)
	}

	return nil
}

func (r *SessionRepository) GetSessionByID(ctx context.Context, sessionID string) (*models.Session, error) {
	query := `
		SELECT session_id, user_id, expires_at, created_at
		FROM session_info
		WHERE session_id = $1
	`
	session := &models.Session{}
	err := r.db.QueryRowContext(ctx, query, hashSessionID(sessionID)).Scan(
		&session.ID,
		&session.UserID,
		&session.ExpiresAt,
		&session.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("get session by id error: %v", err)
	}

	return session, nil
}

func (r *SessionRepository) DeleteSession(ctx context.Context, sessionID string) error {
	query := `DELETE FROM session_info WHERE session_id = $1`
	_, err := r.db.ExecContext(ctx, query, hashSessionID(sessionID))
	if err != nil {
		return fmt.Errorf("delete session error: %v", err)
	}
	return nil
}

func hashSessionID(value string) string {
	valueHash := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", valueHash[:])
}

func (r *SessionRepository) DeleteExpiredSessions(ctx context.Context) error {
	query := `DELETE FROM session_info WHERE expires_at < $1`
	_, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("delete expired sessions error: %v", err)
	}
	return nil
}

package repository

import (
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

func (r *SessionRepository) CreateSession(session *models.Session) error {
	query := `
		INSERT INTO session_info (session_id, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.Exec(
		query,
		session.ID,
		session.UserID,
		session.ExpiresAt,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("create session error: %v", err)
	}

	return nil
}

func (r *SessionRepository) GetSessionByID(sessionID string) (*models.Session, error) {
	query := `
		SELECT session_id, user_id, expires_at, created_at
		FROM session_info
		WHERE session_id = $1
	`
	session := &models.Session{}
	err := r.db.QueryRow(query, sessionID).Scan(
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

func (r *SessionRepository) DeleteSession(sessionID string) error {
	query := `DELETE FROM session_info WHERE session_id = $1`
	_, err := r.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("delete session error: %v", err)
	}
	return nil
}

func (r *SessionRepository) DeleteExpiredSessions() error {
	query := `DELETE FROM session_info WHERE expires_at < $1`
	_, err := r.db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("delete expired sessions error: %v", err)
	}
	return nil
}
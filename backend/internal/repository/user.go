package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/J0es1ick/shortli/internal/models"
	"github.com/jmoiron/sqlx"
)

var ErrLastOwner = errors.New("last owner cannot be removed")
var ErrAdminAlreadyExists = errors.New("an administrator already exists")

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) SaveUser(ctx context.Context, user *models.User) error {
	user.NormalizeAccess()
	query := `
		INSERT INTO user_info (email, password_hash, is_admin, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING user_id, created_at
	`
	err := r.db.QueryRowContext(ctx,
		query,
		user.Email,
		user.PasswordHash,
		user.IsAdmin,
		user.Role,
		time.Now(),
	).Scan(&user.ID, &user.CreatedAt)

	if err != nil {
		return fmt.Errorf("save user error: %v", err)
	}

	return nil
}

func (r *UserRepository) BootstrapAdmin(ctx context.Context, user *models.User) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin admin bootstrap: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, int64(739104226)); err != nil {
		return fmt.Errorf("lock admin bootstrap: %w", err)
	}
	var exists bool
	if err := tx.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM user_info WHERE role = 'owner')`); err != nil {
		return fmt.Errorf("check administrator: %w", err)
	}
	if exists {
		return ErrAdminAlreadyExists
	}
	if err := tx.QueryRowxContext(ctx, `
		INSERT INTO user_info (email, password_hash, is_admin, role, created_at)
		VALUES ($1, $2, TRUE, 'owner', NOW())
		ON CONFLICT (email) DO UPDATE
		SET password_hash = EXCLUDED.password_hash, is_admin = TRUE, role = 'owner'
		RETURNING user_id, created_at
	`, user.Email, user.PasswordHash).Scan(&user.ID, &user.CreatedAt); err != nil {
		return fmt.Errorf("create bootstrap administrator: %w", err)
	}
	user.Role = models.RoleOwner
	user.IsAdmin = true
	if _, err := tx.ExecContext(ctx, `DELETE FROM session_info WHERE user_id = $1`, user.ID); err != nil {
		return fmt.Errorf("revoke bootstrap account sessions: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE api_key SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, user.ID); err != nil {
		return fmt.Errorf("revoke bootstrap account api keys: %w", err)
	}
	return tx.Commit()
}

func (r *UserRepository) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT user_id, email, password_hash, is_admin, role, created_at
		FROM user_info
		WHERE email = $1
	`
	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.IsAdmin,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("find user by email error: %v", err)
	}

	user.NormalizeAccess()
	return user, nil
}

func (r *UserRepository) FindUserByID(ctx context.Context, id int) (*models.User, error) {
	query := `
		SELECT user_id, email, password_hash, is_admin, role, created_at
		FROM user_info
		WHERE user_id = $1
	`
	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.IsAdmin,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("find user by id error: %v", err)
	}

	user.NormalizeAccess()
	return user, nil
}

func (r *UserRepository) FindTotalUsers(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_info").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users error: %v", err)
	}
	return count, nil
}

func (r *UserRepository) FindRoleCounts(ctx context.Context) (map[models.UserRole]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT role, COUNT(*) FROM user_info GROUP BY role`)
	if err != nil {
		return nil, fmt.Errorf("count user roles: %v", err)
	}
	defer rows.Close()
	counts := map[models.UserRole]int{
		models.RoleOwner: 0, models.RoleAdmin: 0,
		models.RoleSupport: 0, models.RoleUser: 0,
	}
	for rows.Next() {
		var role models.UserRole
		var count int
		if err := rows.Scan(&role, &count); err != nil {
			return nil, fmt.Errorf("scan user role count: %v", err)
		}
		counts[role] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user role counts: %v", err)
	}
	return counts, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, user *models.User) error {
	user.NormalizeAccess()
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin user update: %v", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, int64(739104226)); err != nil {
		return fmt.Errorf("lock user update: %v", err)
	}
	var previousRole models.UserRole
	if err := tx.GetContext(ctx, &previousRole, `SELECT role FROM user_info WHERE user_id = $1 FOR UPDATE`, user.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("load user for update: %v", err)
	}
	if previousRole == models.RoleOwner && user.Role != models.RoleOwner {
		var ownerCount int
		if err := tx.GetContext(ctx, &ownerCount, `SELECT COUNT(*) FROM user_info WHERE role = 'owner'`); err != nil {
			return fmt.Errorf("count owners: %v", err)
		}
		if ownerCount <= 1 {
			return ErrLastOwner
		}
	}
	query := `
        UPDATE user_info 
		SET email = $1, role = $2, is_admin = $3
		WHERE user_id = $4
    `
	result, err := tx.ExecContext(ctx, query, user.Email, user.Role, user.IsAdmin, user.ID)
	if err != nil {
		return fmt.Errorf("update user error: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return tx.Commit()
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID int, newPasswordHash string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin password update: %v", err)
	}
	defer tx.Rollback()
	query := `UPDATE user_info SET password_hash = $1 WHERE user_id = $2`
	result, err := tx.ExecContext(ctx, query, newPasswordHash, userID)
	if err != nil {
		return fmt.Errorf("update password error: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM session_info WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("revoke sessions: %v", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE api_key SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
		return fmt.Errorf("revoke api keys: %v", err)
	}
	return tx.Commit()
}

func (r *UserRepository) DeleteUser(ctx context.Context, userID int) ([]string, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin user deletion: %v", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, int64(739104226)); err != nil {
		return nil, fmt.Errorf("lock user deletion: %v", err)
	}
	var role models.UserRole
	if err := tx.GetContext(ctx, &role, `SELECT role FROM user_info WHERE user_id = $1 FOR UPDATE`, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("load user for deletion: %v", err)
	}
	if role == models.RoleOwner {
		var ownerCount int
		if err := tx.GetContext(ctx, &ownerCount, `SELECT COUNT(*) FROM user_info WHERE role = 'owner'`); err != nil {
			return nil, fmt.Errorf("count owners: %v", err)
		}
		if ownerCount <= 1 {
			return nil, ErrLastOwner
		}
	}
	shortCodes := []string{}
	if err := tx.SelectContext(ctx, &shortCodes, `DELETE FROM url_info WHERE user_id = $1 RETURNING short_code`, userID); err != nil {
		return nil, fmt.Errorf("delete user links: %v", err)
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM user_info WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("delete user error: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("user not found")
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return shortCodes, nil
}

func (r *UserRepository) GetAllUsers(ctx context.Context, limit, offset int) ([]models.User, error) {
	query := `
		SELECT user_id, email, is_admin, role, created_at
        FROM user_info
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    `

	users := []models.User{}
	err := r.db.SelectContext(ctx, &users, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get all users error: %v", err)
	}

	return users, nil
}

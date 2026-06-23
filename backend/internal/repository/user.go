package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/J0es1ick/shortli/internal/models"
	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) SaveUser(user *models.User) error {
	query := `
		INSERT INTO user_info (email, password_hash, is_admin, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING user_id
	`
	err := r.db.QueryRow(
		query,
		user.Email,
		user.PasswordHash,
		user.IsAdmin,
		time.Now(),
	).Scan(&user.ID)

	if err != nil {
		return fmt.Errorf("save user error: %v", err)
	}

	return nil
}

func (r *UserRepository) FindUserByEmail(email string) (*models.User, error) {
	query := `
		SELECT user_id, email, password_hash, is_admin, created_at
		FROM user_info
		WHERE email = $1
	`
	user := &models.User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.IsAdmin,
		&user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("find user by email error: %v", err)
	}

	return user, nil
}

func (r *UserRepository) FindUserByID(id int) (*models.User, error) {
	query := `
		SELECT user_id, email, password_hash, is_admin, created_at
		FROM user_info
		WHERE user_id = $1
	`
	user := &models.User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.IsAdmin,
		&user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("find user by id error: %v", err)
	}

	return user, nil
}

func (r *UserRepository) FindTotalUsers() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM user_info").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users error: %v", err)
	}
	return count, nil
}

func (r *UserRepository) UpdateUser(user *models.User) error {
    query := `
        UPDATE user_info 
        SET email = $1, is_admin = $2
        WHERE user_id = $3
    `
    result, err := r.db.Exec(query, user.Email, user.IsAdmin, user.ID)
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
    
    return nil
}

func (r *UserRepository) UpdatePassword(userID int, newPasswordHash string) error {
    query := `UPDATE user_info SET password_hash = $1 WHERE user_id = $2`
    result, err := r.db.Exec(query, newPasswordHash, userID)
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
    
    return nil
}

func (r *UserRepository) DeleteUser(userID int) error {
    query := `DELETE FROM user_info WHERE user_id = $1`
    result, err := r.db.Exec(query, userID)
    if err != nil {
        return fmt.Errorf("delete user error: %v", err)
    }
    
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %v", err)
    }
    
    if rowsAffected == 0 {
        return fmt.Errorf("user not found")
    }
    
    return nil
}

func (r *UserRepository) GetAllUsers(limit, offset int) ([]models.User, error) {
    query := `
        SELECT user_id, email, is_admin, created_at
        FROM user_info
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    `
    
    users := []models.User{}
    err := r.db.Select(&users, query, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("get all users error: %v", err)
    }
    
    return users, nil
}
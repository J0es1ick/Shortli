package models

import "time"

type UserRole string

const (
	RoleOwner   UserRole = "owner"
	RoleAdmin   UserRole = "admin"
	RoleSupport UserRole = "support"
	RoleUser    UserRole = "user"
)

func (role UserRole) IsValid() bool {
	switch role {
	case RoleOwner, RoleAdmin, RoleSupport, RoleUser:
		return true
	default:
		return false
	}
}

type User struct {
	ID           int       `db:"user_id" json:"user_id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	IsAdmin      bool      `db:"is_admin" json:"is_admin"`
	Role         UserRole  `db:"role" json:"role"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

func (user *User) NormalizeAccess() {
	if !user.Role.IsValid() {
		if user.IsAdmin {
			user.Role = RoleOwner
		} else {
			user.Role = RoleUser
		}
	}
	user.IsAdmin = user.Role != RoleUser
}

func (user *User) HasStaffAccess() bool {
	user.NormalizeAccess()
	return user.Role != RoleUser
}

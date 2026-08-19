package userHandlers

import (
	"time"

	"github.com/J0es1ick/shortli/internal/models"
)

type UserResponse struct {
	ID        int             `json:"user_id"`
	Email     string          `json:"email"`
	IsAdmin   bool            `json:"is_admin"`
	Role      models.UserRole `json:"role"`
	CreatedAt time.Time       `json:"created_at"`
}

func NewUserResponse(user *models.User) UserResponse {
	user.NormalizeAccess()
	return UserResponse{
		ID: user.ID, Email: user.Email, IsAdmin: user.IsAdmin,
		Role: user.Role, CreatedAt: user.CreatedAt,
	}
}

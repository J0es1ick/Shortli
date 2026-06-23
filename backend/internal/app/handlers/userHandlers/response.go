package userHandlers

import "time"

type UserResponse struct {
	ID    int    `json:"user_id"`
	Email string `json:"email"`
	// Username  string    `json:"username"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
}
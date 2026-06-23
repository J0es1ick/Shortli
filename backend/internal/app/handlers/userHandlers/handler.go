package userHandlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	response "github.com/J0es1ick/shortli/internal/app/httputils"
	"github.com/J0es1ick/shortli/internal/app/middleware"
	"github.com/J0es1ick/shortli/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	userRepo *repository.UserRepository
}

func NewUserHandler(userRepo *repository.UserRepository) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
	}
}

type UpdateProfileRequest struct {
	Email string `json:"email"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type UpdateUserRequest struct {
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	response.JSON(w, http.StatusOK, UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		IsAdmin:   user.IsAdmin,
		CreatedAt: user.CreatedAt,
	})
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Email == "" {
		response.Error(w, http.StatusBadRequest, "Email is required")
		return
	}

	existingUser, err := h.userRepo.FindUserByEmail(req.Email)
	if err == nil && existingUser.ID != user.ID {
		response.Error(w, http.StatusConflict, "Email already taken")
		return
	}

	user.Email = req.Email
	if err := h.userRepo.UpdateUser(user); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	response.JSON(w, http.StatusOK, UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		IsAdmin:   user.IsAdmin,
		CreatedAt: user.CreatedAt,
	})
}

func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if len(req.NewPassword) < 6 {
		response.Error(w, http.StatusBadRequest, "New password must be at least 6 characters")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		response.Error(w, http.StatusUnauthorized, "Old password is incorrect")
		return
	}

	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to hash new password")
		return
	}

	if err := h.userRepo.UpdatePassword(user.ID, string(newHashedPassword)); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Password updated successfully",
	})
}

func (h *UserHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	if err := h.userRepo.DeleteUser(user.ID); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete account")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Account deleted successfully",
	})
}

func (h *UserHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 || err != nil {
		page = 1
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 || err != nil {
		limit = 10
	}

	offset := (page - 1) * limit

	users, err := h.userRepo.GetAllUsers(limit, offset)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get users")
		return
	}

	total, err := h.userRepo.FindTotalUsers()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get total count")
		return
	}

	userResponses := make([]UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			IsAdmin:   user.IsAdmin,
			CreatedAt: user.CreatedAt,
		}
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"data": userResponses,
		"meta": map[string]interface{}{
			"total":      total,
			"page":       page,
			"limit":      limit,
			"totalPages": (total + limit - 1) / limit,
		},
	})
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	userIDStr := r.PathValue("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	user, err := h.userRepo.FindUserByID(userID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "User not found")
		return
	}

	if req.Email != user.Email {
		existingUser, err := h.userRepo.FindUserByEmail(req.Email)
		if err == nil && existingUser.ID != user.ID {
			response.Error(w, http.StatusConflict, "Email already taken")
			return
		}
	}

	user.Email = req.Email
	user.IsAdmin = req.IsAdmin

	if err := h.userRepo.UpdateUser(user); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	response.JSON(w, http.StatusOK, UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		IsAdmin:   user.IsAdmin,
		CreatedAt: user.CreatedAt,
	})
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	userIDStr := r.PathValue("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	currentUser := middleware.GetUserFromContext(r)
	if currentUser.ID == userID {
		response.Error(w, http.StatusBadRequest, "Cannot delete your own account")
		return
	}

	if err := h.userRepo.DeleteUser(userID); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "User deleted successfully",
	})
}
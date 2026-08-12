package userHandlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	response "github.com/J0es1ick/shortli/internal/app/httputils"
	"github.com/J0es1ick/shortli/internal/app/middleware"
	"github.com/J0es1ick/shortli/internal/repository"
	"github.com/J0es1ick/shortli/pkg/validator"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	userRepo      *repository.UserRepository
	invalidate    func(string)
	secureCookies bool
}

func NewUserHandler(userRepo *repository.UserRepository, invalidate func(string), secureCookies bool) *UserHandler {
	return &UserHandler{
		userRepo: userRepo, invalidate: invalidate, secureCookies: secureCookies,
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

	email, err := validator.ValidateEmail(req.Email)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Email = email

	existingUser, err := h.userRepo.FindUserByEmail(r.Context(), req.Email)
	if err == nil && existingUser.ID != user.ID {
		response.Error(w, http.StatusConflict, "Email already taken")
		return
	}

	user.Email = req.Email
	if err := h.userRepo.UpdateUser(r.Context(), user); err != nil {
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

	if err := validator.ValidatePassword(req.NewPassword); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
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

	if err := h.userRepo.UpdatePassword(r.Context(), user.ID, string(newHashedPassword)); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update password")
		return
	}
	h.clearSessionCookie(w)

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

	shortCodes, err := h.userRepo.DeleteUser(r.Context(), user.ID)
	if err != nil {
		if errors.Is(err, repository.ErrLastAdmin) {
			response.Error(w, http.StatusConflict, "The last administrator cannot be deleted")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Failed to delete account")
		return
	}
	for _, shortCode := range shortCodes {
		h.invalidate(shortCode)
	}
	h.clearSessionCookie(w)

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Account deleted successfully",
	})
}

func (h *UserHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: middleware.SessionCookieName, Value: "", Path: "/",
		Expires: time.Now().Add(-time.Hour), MaxAge: -1, HttpOnly: true,
		Secure: h.secureCookies, SameSite: http.SameSiteStrictMode,
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

	users, err := h.userRepo.GetAllUsers(r.Context(), limit, offset)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get users")
		return
	}

	total, err := h.userRepo.FindTotalUsers(r.Context())
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
	email, err := validator.ValidateEmail(req.Email)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Email = email

	user, err := h.userRepo.FindUserByID(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "User not found")
		return
	}

	if req.Email != user.Email {
		existingUser, err := h.userRepo.FindUserByEmail(r.Context(), req.Email)
		if err == nil && existingUser.ID != user.ID {
			response.Error(w, http.StatusConflict, "Email already taken")
			return
		}
	}

	user.Email = req.Email
	user.IsAdmin = req.IsAdmin

	if err := h.userRepo.UpdateUser(r.Context(), user); err != nil {
		if errors.Is(err, repository.ErrLastAdmin) {
			response.Error(w, http.StatusConflict, "The last administrator cannot be demoted")
			return
		}
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

	shortCodes, err := h.userRepo.DeleteUser(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrLastAdmin) {
			response.Error(w, http.StatusConflict, "The last administrator cannot be deleted")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}
	for _, shortCode := range shortCodes {
		h.invalidate(shortCode)
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "User deleted successfully",
	})
}

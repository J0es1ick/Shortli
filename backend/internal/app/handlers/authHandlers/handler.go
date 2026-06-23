package authHandlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/J0es1ick/shortli/internal/app/handlers/userHandlers"
	response "github.com/J0es1ick/shortli/internal/app/httputils"
	"github.com/J0es1ick/shortli/internal/app/middleware"
	"github.com/J0es1ick/shortli/internal/models"
	"github.com/J0es1ick/shortli/internal/repository"
	"github.com/J0es1ick/shortli/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userRepo    *repository.UserRepository
	sessionRepo *repository.SessionRepository
}

func NewAuthHandler(userRepo *repository.UserRepository, sessionRepo *repository.SessionRepository) *AuthHandler {
	return &AuthHandler{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	if len(req.Password) < 6 {
		response.Error(w, http.StatusBadRequest, "Password must be at least 6 characters")
		return
	}

	_, err := h.userRepo.FindUserByEmail(req.Email)
	if err == nil {
		response.Error(w, http.StatusConflict, "User with this email already exists")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	user := &models.User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		IsAdmin:      false,
	}

	if err := h.userRepo.SaveUser(user); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	response.JSON(w, http.StatusCreated, userHandlers.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		IsAdmin:   user.IsAdmin,
		CreatedAt: user.CreatedAt,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	user, err := h.userRepo.FindUserByEmail(req.Email)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		response.Error(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	sessionID, err := utils.GenerateRandomString(32)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	session := &models.Session{
		ID:        sessionID,
		UserID:    user.ID,
		ExpiresAt: expiresAt,
	}

	if err := h.sessionRepo.CreateSession(session); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Expires:  expiresAt,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})

	response.JSON(w, http.StatusOK, userHandlers.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		IsAdmin:   user.IsAdmin,
		CreatedAt: user.CreatedAt,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	sessionCookie, err := r.Cookie("session_id")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Not authenticated")
		return
	}

	if err := h.sessionRepo.DeleteSession(sessionCookie.Value); err != nil {
		_ = h.sessionRepo.DeleteSession(sessionCookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "session_id",
		Value:   "",
		Expires: time.Now().Add(-1 * time.Hour),
		Path:    "/",
	})

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Logged out successfully",
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	response.JSON(w, http.StatusOK, userHandlers.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		IsAdmin:   user.IsAdmin,
		CreatedAt: user.CreatedAt,
	})
}
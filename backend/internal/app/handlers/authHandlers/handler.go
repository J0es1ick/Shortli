package authHandlers

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/J0es1ick/shortli/internal/app/handlers/userHandlers"
	response "github.com/J0es1ick/shortli/internal/app/httputils"
	"github.com/J0es1ick/shortli/internal/app/middleware"
	"github.com/J0es1ick/shortli/internal/models"
	"github.com/J0es1ick/shortli/internal/repository"
	"github.com/J0es1ick/shortli/internal/utils"
	"github.com/J0es1ick/shortli/pkg/validator"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userRepo       *repository.UserRepository
	sessionRepo    *repository.SessionRepository
	secureCookies  bool
	bootstrapToken string
}

func NewAuthHandler(userRepo *repository.UserRepository, sessionRepo *repository.SessionRepository, secureCookies bool, bootstrapToken string) *AuthHandler {
	return &AuthHandler{
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		secureCookies:  secureCookies,
		bootstrapToken: bootstrapToken,
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

type BootstrapRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

func (h *AuthHandler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	if h.bootstrapToken == "" {
		response.Error(w, http.StatusNotFound, "Not found")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var req BootstrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if subtle.ConstantTimeCompare([]byte(req.Token), []byte(h.bootstrapToken)) != 1 {
		response.Error(w, http.StatusForbidden, "Invalid bootstrap token")
		return
	}
	email, err := validator.ValidateEmail(req.Email)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validator.ValidatePassword(req.Password); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}
	user := &models.User{Email: email, PasswordHash: string(passwordHash), IsAdmin: true}
	if err := h.userRepo.BootstrapAdmin(r.Context(), user); err != nil {
		if errors.Is(err, repository.ErrAdminAlreadyExists) {
			response.Error(w, http.StatusConflict, "An administrator already exists")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Failed to create administrator")
		return
	}
	response.JSON(w, http.StatusCreated, userHandlers.UserResponse{
		ID: user.ID, Email: user.Email, IsAdmin: true, CreatedAt: user.CreatedAt,
	})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	email, err := validator.ValidateEmail(req.Email)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validator.ValidatePassword(req.Password); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Email = email

	_, err = h.userRepo.FindUserByEmail(r.Context(), req.Email)
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

	if err := h.userRepo.SaveUser(r.Context(), user); err != nil {
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

	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	email, err := validator.ValidateEmail(req.Email)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}
	user, err := h.userRepo.FindUserByEmail(r.Context(), email)
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

	if err := h.sessionRepo.CreateSession(r.Context(), session); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Expires:  expiresAt,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
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

	_ = h.sessionRepo.DeleteSession(r.Context(), sessionCookie.Value)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteStrictMode,
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

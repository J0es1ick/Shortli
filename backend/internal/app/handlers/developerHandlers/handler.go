package developerHandlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	response "github.com/J0es1ick/shortli/internal/app/httputils"
	"github.com/J0es1ick/shortli/internal/app/middleware"
	"github.com/J0es1ick/shortli/internal/models"
	"github.com/J0es1ick/shortli/internal/repository"
	"github.com/J0es1ick/shortli/internal/utils"
)

type Handler struct{ repo *repository.APIKeyRepository }

func NewHandler(repo *repository.APIKeyRepository) *Handler { return &Handler{repo: repo} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	keys, err := h.repo.List(r.Context(), user.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to load API keys")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"data": keys, "limit": 10})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) < 2 || len(req.Name) > 60 {
		response.Error(w, http.StatusBadRequest, "API key name must be between 2 and 60 characters")
		return
	}
	count, err := h.repo.CountActive(r.Context(), user.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to check API key limit")
		return
	}
	if count >= 10 {
		response.Error(w, http.StatusConflict, "Active API key limit reached")
		return
	}
	raw, prefix, hash, err := utils.GenerateAPIKey()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to generate API key")
		return
	}
	key := &models.APIKey{UserID: user.ID, Name: req.Name, Prefix: prefix, Hash: hash}
	if err := h.repo.Create(r.Context(), key); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to save API key")
		return
	}
	response.JSON(w, http.StatusCreated, map[string]interface{}{"key": raw, "api_key": key})
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid API key ID")
		return
	}
	if err := h.repo.Revoke(r.Context(), id, user.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.Error(w, http.StatusNotFound, "API key not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Failed to revoke API key")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

package abuseHandlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	response "github.com/J0es1ick/shortli/internal/app/httputils"
	"github.com/J0es1ick/shortli/internal/app/middleware"
	"github.com/J0es1ick/shortli/internal/models"
	"github.com/J0es1ick/shortli/internal/repository"
	"github.com/J0es1ick/shortli/pkg/validator"
)

type Handler struct {
	abuseRepo  *repository.AbuseRepository
	urlRepo    *repository.UrlRepository
	salt       string
	trustProxy bool
	invalidate func(string)
}

type CreateRequest struct {
	ShortLink      string `json:"short_link"`
	ReporterEmail  string `json:"reporter_email,omitempty"`
	Reason         string `json:"reason"`
	Details        string `json:"details"`
	CompanyWebsite string `json:"company_website,omitempty"`
}

type ResolveRequest struct {
	Status         string `json:"status"`
	ResolutionNote string `json:"resolution_note"`
	PauseLink      bool   `json:"pause_link"`
	BlockDomain    bool   `json:"block_domain"`
}

var allowedReasons = map[string]bool{
	"phishing": true, "malware": true, "spam": true,
	"impersonation": true, "illegal": true, "other": true,
}

var allowedStatuses = map[string]bool{
	"reviewed": true, "dismissed": true, "blocked": true,
}

func NewHandler(
	abuseRepo *repository.AbuseRepository,
	urlRepo *repository.UrlRepository,
	salt string,
	trustProxy bool,
	invalidate func(string),
) *Handler {
	return &Handler{
		abuseRepo: abuseRepo, urlRepo: urlRepo, salt: salt,
		trustProxy: trustProxy, invalidate: invalidate,
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 20<<10)
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.CompanyWebsite != "" {
		response.JSON(w, http.StatusAccepted, map[string]string{"status": "received"})
		return
	}

	shortCode, err := extractShortCode(req.ShortLink)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	reason := strings.ToLower(strings.TrimSpace(req.Reason))
	if !allowedReasons[reason] {
		response.Error(w, http.StatusBadRequest, "Select a valid report reason")
		return
	}
	details := strings.TrimSpace(req.Details)
	if len(details) < 10 || len(details) > 2000 {
		response.Error(w, http.StatusBadRequest, "Details must be between 10 and 2000 characters")
		return
	}

	var reporterEmail *string
	if strings.TrimSpace(req.ReporterEmail) != "" {
		email, err := validator.ValidateEmail(req.ReporterEmail)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		reporterEmail = &email
	}

	link, err := h.urlRepo.FindUrlByCode(shortCode)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Short link not found")
		return
	}
	ipHash := h.hashIP(middleware.GetClientIP(r, h.trustProxy))
	urlID := int64(link.ID)
	report := &models.AbuseReport{
		URLID: &urlID, ShortCode: shortCode,
		ReporterEmail: reporterEmail, ReporterIPHash: ipHash,
		Reason: reason, Details: details,
	}
	if err := h.abuseRepo.Create(report); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			response.Error(w, http.StatusConflict, "A pending report for this link was already received")
			return
		}
		response.Error(w, http.StatusInternalServerError, "Failed to submit report")
		return
	}
	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"report_id": report.ID, "status": report.Status,
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page := parsePositive(r.URL.Query().Get("page"), 1, 1, 100000)
	limit := parsePositive(r.URL.Query().Get("limit"), 20, 1, 100)
	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	if status == "" {
		status = "pending"
	}
	if status != "all" && status != "pending" && !allowedStatuses[status] {
		response.Error(w, http.StatusBadRequest, "Invalid report status")
		return
	}
	reports, total, err := h.abuseRepo.List(status, limit, (page-1)*limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to load abuse reports")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"data": reports,
		"meta": map[string]int{"page": page, "limit": limit, "total": total},
	})
}

func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		response.Error(w, http.StatusBadRequest, "Invalid report ID")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var req ResolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	req.ResolutionNote = strings.TrimSpace(req.ResolutionNote)
	if !allowedStatuses[req.Status] {
		response.Error(w, http.StatusBadRequest, "Invalid resolution status")
		return
	}
	if len(req.ResolutionNote) > 1000 {
		response.Error(w, http.StatusBadRequest, "Resolution note is too long")
		return
	}

	report, err := h.abuseRepo.FindByID(id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Abuse report not found")
		return
	}
	user := middleware.GetUserFromContext(r)
	shouldPause := req.PauseLink || req.BlockDomain || req.Status == "blocked"
	if shouldPause && report.URLID != nil {
		link, findErr := h.urlRepo.FindUrlByCode(report.ShortCode)
		if findErr == nil {
			if err := h.urlRepo.UpdateUrlSettings(report.ShortCode, false, link.ExpiresAt); err != nil {
				response.Error(w, http.StatusInternalServerError, "Failed to pause reported link")
				return
			}
			h.invalidate(report.ShortCode)
		}
	}
	if req.BlockDomain && report.OriginalURL != "" {
		parsed, parseErr := url.Parse(report.OriginalURL)
		if parseErr != nil || parsed.Hostname() == "" {
			response.Error(w, http.StatusInternalServerError, "Failed to identify destination domain")
			return
		}
		reason := req.ResolutionNote
		if reason == "" {
			reason = fmt.Sprintf("Blocked from abuse report #%d", id)
		}
		if err := h.abuseRepo.BlockDomain(parsed.Hostname(), reason, user.ID); err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to block destination domain")
			return
		}
	}
	if err := h.abuseRepo.Resolve(id, req.Status, req.ResolutionNote, user.ID); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to resolve abuse report")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

func (h *Handler) BlockedDomains(w http.ResponseWriter, _ *http.Request) {
	items, err := h.abuseRepo.ListBlockedDomains()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to load blocked domains")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

func (h *Handler) UnblockDomain(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		response.Error(w, http.StatusBadRequest, "Invalid domain ID")
		return
	}
	if err := h.abuseRepo.UnblockDomain(id); err != nil {
		response.Error(w, http.StatusNotFound, "Blocked domain not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) hashIP(value string) string {
	mac := hmac.New(sha256.New, []byte(h.salt))
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func extractShortCode(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("Enter a short link or code")
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Host != "" {
		value = strings.Trim(parsed.Path, "/")
	}
	if strings.Contains(value, "/") {
		return "", fmt.Errorf("Enter a valid Shortli link")
	}
	code := strings.ToLower(strings.TrimSpace(value))
	if len(code) < 3 || len(code) > 100 {
		return "", fmt.Errorf("Enter a valid short code")
	}
	return code, nil
}

func parsePositive(value string, fallback, min, max int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < min || parsed > max {
		return fallback
	}
	return parsed
}

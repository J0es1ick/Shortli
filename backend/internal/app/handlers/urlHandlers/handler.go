package urlHandlers

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	response "github.com/J0es1ick/shortli/internal/app/httputils"
	"github.com/J0es1ick/shortli/internal/app/middleware"
	"github.com/J0es1ick/shortli/internal/app/tasks"
	"github.com/J0es1ick/shortli/internal/config"
	"github.com/J0es1ick/shortli/internal/models"
	"github.com/J0es1ick/shortli/internal/repository"
	"github.com/J0es1ick/shortli/pkg/shortener"
	"github.com/J0es1ick/shortli/pkg/validator"
	"github.com/skip2/go-qrcode"
)

type Handler struct {
	cfg             *config.Config
	urlRepository   *repository.UrlRepository
	abuseRepository *repository.AbuseRepository
	redirectCache   *redirectCache
	clickRecorder   *tasks.ClickRecorder
	clientIP        *middleware.ClientIPResolver
}

func NewHandler(
	cfg *config.Config,
	urlRepository *repository.UrlRepository,
	abuseRepository *repository.AbuseRepository,
	clickRecorder *tasks.ClickRecorder,
	clientIP *middleware.ClientIPResolver,
) *Handler {
	return &Handler{
		cfg:             cfg,
		urlRepository:   urlRepository,
		abuseRepository: abuseRepository,
		redirectCache:   newRedirectCache(5*time.Minute, 10_000),
		clickRecorder:   clickRecorder,
		clientIP:        clientIP,
	}
}

func (h *Handler) shortURL(r *http.Request, shortCode string) string {
	if publicBaseURL := strings.TrimRight(h.cfg.PublicBaseURL, "/"); publicBaseURL != "" {
		return fmt.Sprintf("%s/%s", publicBaseURL, shortCode)
	}

	protocol := ""
	if h.clientIP != nil && h.clientIP.Trusts(r) {
		protocol = r.Header.Get("X-Forwarded-Proto")
	}
	if protocol == "" {
		protocol = "http"
		if r.TLS != nil {
			protocol = "https"
		}
	}

	return fmt.Sprintf("%s://%s/%s", protocol, r.Host, shortCode)
}

func (h *Handler) writeShortenResponse(w http.ResponseWriter, r *http.Request, status int, url *models.URL) {
	shortURL := h.shortURL(r, url.ShortCode)
	qrCodeBase64 := ""
	includeQR := !strings.HasPrefix(r.URL.Path, "/api/v1/") || r.URL.Query().Get("include_qr") == "true"
	if includeQR {
		qrCode, err := qrcode.Encode(shortURL, qrcode.High, 512)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to generate QR code")
			return
		}
		qrCodeBase64 = fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(qrCode))
	}

	response.JSON(w, status, UrlResponse{
		OriginalURL:  url.OriginalURL,
		ShortCode:    url.ShortCode,
		ShortURL:     shortURL,
		QRCodeBase64: qrCodeBase64,
		ExpiresAt:    url.ExpiresAt,
		IsActive:     url.IsActive,
	})
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		h.Redirect(w, r)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "URL Shortener API",
		"version": "1.0",
	})
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	databaseStatus := "operational"
	status := "operational"
	statusCode := http.StatusOK
	if err := h.urlRepository.Health(ctx); err != nil {
		databaseStatus = "degraded"
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	response.JSON(w, statusCode, map[string]interface{}{
		"status":     status,
		"version":    "1.1",
		"checked_at": time.Now().UTC(),
		"services": map[string]string{
			"api":      "operational",
			"database": databaseStatus,
		},
		"click_queue": h.clickRecorder.Stats(),
	})
}

func (h *Handler) Liveness(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"status": "operational", "checked_at": time.Now().UTC(),
	})
}

func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	var req UrlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.CompanyWebsite != "" {
		response.JSON(w, http.StatusAccepted, map[string]string{"status": "created"})
		return
	}

	if req.OriginalURL == "" {
		response.Error(w, http.StatusBadRequest, "Required original_url")
		return
	}

	normalizedURL, err := validator.ValidateURL(req.OriginalURL)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	req.OriginalURL = normalizedURL
	parsedDestination, err := url.Parse(req.OriginalURL)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid destination URL")
		return
	}
	blocked, err := h.abuseRepository.IsDomainBlocked(r.Context(), parsedDestination.Hostname())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to validate destination domain")
		return
	}
	if blocked {
		response.Error(w, http.StatusBadRequest, "This destination domain is blocked")
		return
	}
	if req.ExpiresAt != nil {
		now := time.Now()
		if !req.ExpiresAt.After(now.Add(5 * time.Minute)) {
			response.Error(w, http.StatusBadRequest, "Expiration must be at least 5 minutes in the future")
			return
		}
		if req.ExpiresAt.After(now.AddDate(1, 0, 0)) {
			response.Error(w, http.StatusBadRequest, "Expiration cannot be more than one year away")
			return
		}
	}
	user := middleware.GetUserFromContext(r)
	var userID *int
	if user != nil {
		userID = &user.ID
	}
	existingURL, lookupErr := h.urlRepository.FindUrlByOriginalForOwner(r.Context(), req.OriginalURL, userID)
	if lookupErr == nil {
		h.redirectCache.Set(existingURL)
		h.writeShortenResponse(w, r, http.StatusOK, existingURL)
		return
	}
	if !errors.Is(lookupErr, sql.ErrNoRows) {
		response.Error(w, http.StatusInternalServerError, "Failed to check existing URL")
		return
	}

	customAlias, err := validator.ValidateShortCode(req.CustomAlias)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	var shortCode string
	if customAlias != "" {
		existingURL, lookupErr := h.urlRepository.FindUrlByCode(r.Context(), customAlias)
		if lookupErr == nil {
			_ = existingURL
			response.Error(w, http.StatusConflict, "Custom alias is already in use")
			return
		}
		if !strings.Contains(lookupErr.Error(), "url not found") {
			response.Error(w, http.StatusInternalServerError, "Failed to check custom alias")
			return
		}
		shortCode = customAlias
	} else {
		for attempt := 0; attempt <= 5; attempt++ {
			shortCode = shortener.GenerateShortCode(req.OriginalURL, attempt)
			if validator.IsReservedShortCode(shortCode) {
				shortCode = ""
				continue
			}
			existingURLByCode, codeErr := h.urlRepository.FindUrlByCode(r.Context(), shortCode)
			if codeErr != nil && strings.Contains(codeErr.Error(), "url not found") {
				break
			}
			if codeErr != nil {
				response.Error(w, http.StatusInternalServerError, "Failed to generate unique short code")
				return
			}
			_ = existingURLByCode
			shortCode = ""
		}
		if shortCode == "" {
			response.Error(w, http.StatusInternalServerError, "Failed to generate unique short code")
			return
		}
	}

	url := &models.URL{
		OriginalURL: req.OriginalURL,
		ShortCode:   shortCode,
		UserID:      userID,
		ClickCount:  0,
		CreatedAt:   time.Now(),
		ExpiresAt:   req.ExpiresAt,
		IsActive:    true,
	}

	savedURL, created, err := h.urlRepository.FindOrSaveUrl(r.Context(), url)
	if err != nil {
		log.Printf("shorten storage failure: %v", err)
		if strings.Contains(err.Error(), "already exists") {
			if customAlias != "" {
				response.Error(w, http.StatusConflict, "Custom alias is already in use")
			} else {
				response.Error(w, http.StatusConflict, "URL already exists")
			}
			return
		}
		response.Error(w, http.StatusInternalServerError, "Failed to save URL")
		return
	}
	h.redirectCache.Set(savedURL)

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	h.writeShortenResponse(w, r, status, savedURL)
}

func (h *Handler) UserHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 || err != nil {
		page = 1
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 50 || err != nil {
		limit = 10
	}

	offset := (page - 1) * limit

	urls, err := h.urlRepository.FindUrlsByUserID(r.Context(), user.ID, limit, offset)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get user history")
		return
	}

	total, err := h.urlRepository.GetTotalUrlsByUserID(r.Context(), user.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get total count")
		return
	}

	historyResults := make([]HistoryUrlResponse, len(urls))
	for i, url := range urls {
		shortURL := h.shortURL(r, url.ShortCode)
		qrCode, _ := qrcode.Encode(shortURL, qrcode.High, 320)

		var qrCodeBase64 string
		if qrCode != nil {
			qrCodeBase64 = base64.StdEncoding.EncodeToString(qrCode)
		}

		historyResults[i] = HistoryUrlResponse{
			URLID:        url.ID,
			OriginalURL:  url.OriginalURL,
			ShortCode:    url.ShortCode,
			ShortURL:     shortURL,
			QRCodeBase64: fmt.Sprintf("data:image/png;base64,%s", qrCodeBase64),
			ClickCount:   url.ClickCount,
			CreatedAt:    url.CreatedAt,
			ExpiresAt:    url.ExpiresAt,
			IsActive:     url.IsActive,
		}
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"data": historyResults,
		"meta": map[string]interface{}{
			"total":      total,
			"page":       page,
			"limit":      limit,
			"totalPages": int(math.Ceil(float64(total) / float64(limit))),
		},
	})
}

func (h *Handler) SearchUrls(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		response.Error(w, http.StatusBadRequest, "Query parameter 'q' is required")
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

	urls, err := h.urlRepository.SearchUrls(r.Context(), query, limit, offset)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Search failed")
		return
	}

	total, err := h.urlRepository.GetTotalSearchUrls(r.Context(), query)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get total count")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"data": urls,
		"meta": map[string]interface{}{
			"total":      total,
			"page":       page,
			"limit":      limit,
			"totalPages": int(math.Ceil(float64(total) / float64(limit))),
			"query":      query,
		},
	})
}

func (h *Handler) AdminStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	totalURLs, err := h.urlRepository.GetTotalUrls(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get URLs count")
		return
	}

	totalClicks, err := h.urlRepository.GetTotalClicks(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get clicks count")
		return
	}

	// totalUsers, err := h.userRepo.GetTotalUsers()
	// if err != nil {
	//     response.Error(w, http.StatusInternalServerError, "Failed to get users count")
	//     return
	// }

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"stats": map[string]interface{}{
			"total_urls":   totalURLs,
			"total_clicks": totalClicks,
			// "total_users":  totalUsers,
		},
	})
}

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	shortCode := strings.TrimPrefix(r.URL.Path, "/")
	url, cacheHit := h.redirectCache.Get(shortCode)
	if !cacheHit {
		var err error
		url, err = h.urlRepository.FindUrlByCode(r.Context(), shortCode)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				response.Error(w, http.StatusNotFound, "URL not found")
			} else {
				response.Error(w, http.StatusInternalServerError, "Database error")
			}
			return
		}
		h.redirectCache.Set(url)
	}
	if cacheHit {
		w.Header().Set("X-Shortli-Cache", "HIT")
	} else {
		w.Header().Set("X-Shortli-Cache", "MISS")
	}

	if !url.IsActive {
		response.Error(w, http.StatusGone, "This link is paused")
		return
	}
	if url.ExpiresAt != nil && !url.ExpiresAt.After(time.Now()) {
		response.Error(w, http.StatusGone, "This link has expired")
		return
	}

	if err := h.clickRecorder.Submit(h.clickEventFromRequest(r, url.ID)); err != nil {
		fmt.Printf("queue click event: %v\n", err)
	}
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, url.OriginalURL, http.StatusFound)
}

func (h *Handler) InvalidateRedirect(shortCode string) {
	h.redirectCache.Delete(shortCode)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	shortCode := r.PathValue("shortCode")
	url, user, ok := h.ownedURL(w, r, shortCode)
	if !ok {
		return
	}
	_ = user

	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var req UpdateUrlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.IsActive == nil && req.ExpiresAt == nil && !req.ClearExpiration {
		response.Error(w, http.StatusBadRequest, "No link settings supplied")
		return
	}

	isActive := url.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	expiresAt := url.ExpiresAt
	if req.ClearExpiration {
		expiresAt = nil
	}
	if req.ExpiresAt != nil {
		if !req.ExpiresAt.After(time.Now().Add(5 * time.Minute)) {
			response.Error(w, http.StatusBadRequest, "Expiration must be at least 5 minutes in the future")
			return
		}
		if req.ExpiresAt.After(time.Now().AddDate(1, 0, 0)) {
			response.Error(w, http.StatusBadRequest, "Expiration cannot be more than one year away")
			return
		}
		expiresAt = req.ExpiresAt
	}

	if err := h.urlRepository.UpdateUrlSettings(r.Context(), shortCode, isActive, expiresAt); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update link")
		return
	}
	url.IsActive, url.ExpiresAt = isActive, expiresAt
	h.redirectCache.Set(url)
	response.JSON(w, http.StatusOK, url)
}

func (h *Handler) Analytics(w http.ResponseWriter, r *http.Request) {
	shortCode := r.PathValue("shortCode")
	url, _, ok := h.ownedURL(w, r, shortCode)
	if !ok {
		return
	}

	days, err := strconv.Atoi(r.URL.Query().Get("days"))
	if err != nil || days < 1 {
		days = 30
	}
	if days > 365 {
		days = 365
	}
	analytics, err := h.urlRepository.GetAnalytics(r.Context(), url.ID, time.Now().AddDate(0, 0, -days))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to load analytics")
		return
	}
	response.JSON(w, http.StatusOK, AnalyticsResponse{
		ShortCode: shortCode, PeriodDays: days, LifetimeClicks: url.ClickCount, AnalyticsSummary: analytics,
	})
}

func (h *Handler) ownedURL(w http.ResponseWriter, r *http.Request, shortCode string) (*models.URL, *models.User, bool) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Authentication required")
		return nil, nil, false
	}
	url, err := h.urlRepository.FindUrlByCode(r.Context(), shortCode)
	if err != nil {
		response.Error(w, http.StatusNotFound, "URL not found")
		return nil, nil, false
	}
	if !user.IsAdmin && (url.UserID == nil || *url.UserID != user.ID) {
		response.Error(w, http.StatusForbidden, "You cannot manage this URL")
		return nil, nil, false
	}
	return url, user, true
}

func (h *Handler) UrlStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	shortCode := r.PathValue("shortCode")
	url, _, ok := h.ownedURL(w, r, shortCode)
	if !ok {
		return
	}

	response.JSON(w, http.StatusOK, UrlStatsResponse{
		URL:         *url,
		TotalClicks: url.ClickCount,
	})
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
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

	urls, err := h.urlRepository.FindAllUrl(r.Context(), limit, offset)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.Error(w, http.StatusNotFound, "URL not found")
		} else {
			response.Error(w, http.StatusInternalServerError, "Database error")
		}
		return
	}

	total, err := h.urlRepository.GetTotalUrls(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get total count")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"data": urls,
		"meta": map[string]interface{}{
			"total":      total,
			"page":       page,
			"limit":      limit,
			"totalPages": int(math.Ceil(float64(total) / float64(limit))),
		},
	})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	shortCode := r.PathValue("shortCode")
	if shortCode == "" {
		response.Error(w, http.StatusBadRequest, "Short code is required")
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/api/admin/") {
		user := middleware.GetUserFromContext(r)
		url, err := h.urlRepository.FindUrlByCode(r.Context(), shortCode)
		if err != nil {
			response.Error(w, http.StatusNotFound, "URL not found")
			return
		}
		if user == nil || url.UserID == nil || *url.UserID != user.ID {
			response.Error(w, http.StatusForbidden, "You cannot delete this URL")
			return
		}
	}

	if err := h.urlRepository.DeleteUrlByCode(r.Context(), shortCode); err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.Error(w, http.StatusNotFound, "URL not found")
		} else {
			response.Error(w, http.StatusInternalServerError, "Failed to delete URL")
		}
		return
	}
	h.redirectCache.Delete(shortCode)

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "URL deleted successfully",
		"code":    shortCode,
	})
}

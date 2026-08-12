package urlHandlers

import (
	"time"

	"github.com/J0es1ick/shortli/internal/models"
)

type UrlRequest struct {
	OriginalURL    string     `json:"original_url"`
	CustomAlias    string     `json:"custom_alias,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CompanyWebsite string     `json:"company_website,omitempty"`
}

type UrlResponse struct {
	OriginalURL  string     `json:"original_url"`
	ShortCode    string     `json:"short_code"`
	ShortURL     string     `json:"short_url"`
	QRCodeBase64 string     `json:"qr_code_base64,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at"`
	IsActive     bool       `json:"is_active"`
}

type UrlStatsResponse struct {
	models.URL
	TotalClicks int `json:"total_clicks"`
}

type HistoryUrlResponse struct {
	URLID        int        `json:"url_id"`
	OriginalURL  string     `json:"original_url"`
	ShortCode    string     `json:"short_code"`
	ShortURL     string     `json:"short_url"`
	QRCodeBase64 string     `json:"qr_code_base64,omitempty"`
	ClickCount   int        `json:"click_count"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    *time.Time `json:"expires_at"`
	IsActive     bool       `json:"is_active"`
}

type UpdateUrlRequest struct {
	IsActive        *bool      `json:"is_active,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	ClearExpiration bool       `json:"clear_expiration,omitempty"`
}

type AnalyticsResponse struct {
	ShortCode      string `json:"short_code"`
	PeriodDays     int    `json:"period_days"`
	LifetimeClicks int    `json:"lifetime_clicks"`
	models.AnalyticsSummary
}

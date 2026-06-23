package urlHandlers

import "github.com/J0es1ick/shortli/internal/models"

type UrlRequest struct {
	OriginalURL string `json:"original_url"`
}

type UrlResponse struct {
	OriginalURL  string `json:"original_url"`
	ShortCode    string `json:"short_code"`
	ShortURL     string `json:"short_url"`
	QRCodeBase64 string `json:"qr_code_base64,omitempty"`
}

type UrlStatsResponse struct {
	models.URL
	TotalClicks int `json:"total_clicks"`
}
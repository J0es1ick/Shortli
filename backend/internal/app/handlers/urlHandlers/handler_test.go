package urlHandlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image/png"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/J0es1ick/shortli/internal/config"
	"github.com/J0es1ick/shortli/internal/models"
)

func TestShortURLUsesConfiguredPublicBase(t *testing.T) {
	handler := &Handler{cfg: &config.Config{PublicBaseURL: "https://go.example.com/"}}
	request := httptest.NewRequest("GET", "http://internal:8088/", nil)

	if got := handler.shortURL(request, "demo"); got != "https://go.example.com/demo" {
		t.Fatalf("shortURL() = %q", got)
	}
}

func TestShortURLFallsBackToForwardedRequestHost(t *testing.T) {
	handler := &Handler{cfg: &config.Config{TrustProxyHeaders: true}}
	request := httptest.NewRequest("GET", "http://internal:8088/", nil)
	request.Host = "sho.rt"
	request.Header.Set("X-Forwarded-Proto", "https")

	if got := handler.shortURL(request, "demo"); got != "https://sho.rt/demo" {
		t.Fatalf("shortURL() = %q", got)
	}
}

func TestShortenResponseCreatesHighResolutionQR(t *testing.T) {
	handler := &Handler{cfg: &config.Config{PublicBaseURL: "https://go.example.com"}}
	request := httptest.NewRequest("POST", "http://internal:8088/api/shorten", nil)
	recorder := httptest.NewRecorder()

	handler.writeShortenResponse(recorder, request, 201, &models.URL{
		OriginalURL: "https://example.com/article",
		ShortCode:   "demo",
	})

	if recorder.Code != 201 {
		t.Fatalf("status = %d", recorder.Code)
	}

	var payload UrlResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.ShortURL != "https://go.example.com/demo" {
		t.Fatalf("short_url = %q", payload.ShortURL)
	}

	encodedPNG := strings.TrimPrefix(payload.QRCodeBase64, "data:image/png;base64,")
	qrBytes, err := base64.StdEncoding.DecodeString(encodedPNG)
	if err != nil {
		t.Fatalf("decode QR: %v", err)
	}
	qrConfig, err := png.DecodeConfig(bytes.NewReader(qrBytes))
	if err != nil {
		t.Fatalf("decode QR image: %v", err)
	}
	if qrConfig.Width != 512 || qrConfig.Height != 512 {
		t.Fatalf("QR dimensions = %dx%d", qrConfig.Width, qrConfig.Height)
	}
}

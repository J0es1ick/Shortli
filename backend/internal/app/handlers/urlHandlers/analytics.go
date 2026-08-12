package urlHandlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/J0es1ick/shortli/internal/models"
)

var countryCodePattern = regexp.MustCompile(`^[A-Z]{2}$`)

func (h *Handler) clickEventFromRequest(r *http.Request, urlID int) *models.ClickEvent {
	device, browser, operatingSystem := parseUserAgent(r.UserAgent())
	referrer := "Direct"
	if value := r.Referer(); value != "" {
		if parsed, err := url.Parse(value); err == nil && parsed.Hostname() != "" {
			referrer = strings.ToLower(parsed.Hostname())
		}
	}
	country := strings.ToUpper(strings.TrimSpace(r.Header.Get("CF-IPCountry")))
	if country == "" {
		country = strings.ToUpper(strings.TrimSpace(r.Header.Get("X-Country-Code")))
	}
	if !countryCodePattern.MatchString(country) {
		country = "Unknown"
	}

	clientIP := h.clientIP.Resolve(r)
	mac := hmac.New(sha256.New, []byte(h.cfg.AnalyticsSalt))
	_, _ = mac.Write([]byte(clientIP))

	return &models.ClickEvent{
		URLID: urlID, ClickedAt: time.Now().UTC(), DeviceType: device,
		Browser: browser, OS: operatingSystem, ReferrerHost: referrer,
		CountryCode: country, IPHash: hex.EncodeToString(mac.Sum(nil)),
	}
}

func parseUserAgent(value string) (string, string, string) {
	ua := strings.ToLower(value)
	device := "Desktop"
	switch {
	case strings.Contains(ua, "bot") || strings.Contains(ua, "crawler") || strings.Contains(ua, "spider"):
		device = "Bot"
	case strings.Contains(ua, "ipad") || strings.Contains(ua, "tablet"):
		device = "Tablet"
	case strings.Contains(ua, "mobile") || strings.Contains(ua, "iphone") || strings.Contains(ua, "android"):
		device = "Mobile"
	}

	browser := "Other"
	switch {
	case strings.Contains(ua, "edg/"):
		browser = "Edge"
	case strings.Contains(ua, "opr/") || strings.Contains(ua, "opera"):
		browser = "Opera"
	case strings.Contains(ua, "firefox/"):
		browser = "Firefox"
	case strings.Contains(ua, "chrome/") || strings.Contains(ua, "crios/"):
		browser = "Chrome"
	case strings.Contains(ua, "safari/"):
		browser = "Safari"
	}

	operatingSystem := "Other"
	switch {
	case strings.Contains(ua, "windows"):
		operatingSystem = "Windows"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		operatingSystem = "iOS"
	case strings.Contains(ua, "android"):
		operatingSystem = "Android"
	case strings.Contains(ua, "mac os") || strings.Contains(ua, "macintosh"):
		operatingSystem = "macOS"
	case strings.Contains(ua, "linux"):
		operatingSystem = "Linux"
	}
	return device, browser, operatingSystem
}

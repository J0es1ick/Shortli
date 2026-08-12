package validator

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

var shortCodePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)
var passwordLetterPattern = regexp.MustCompile(`[A-Za-zА-Яа-яЁё]`)
var passwordNumberPattern = regexp.MustCompile(`[0-9]`)

func ValidateShortCode(value string) (string, error) {
	shortCode := strings.ToLower(strings.TrimSpace(value))
	if shortCode == "" {
		return "", nil
	}
	if len(shortCode) < 3 || len(shortCode) > 32 {
		return "", fmt.Errorf("custom alias must be between 3 and 32 characters")
	}
	if !shortCodePattern.MatchString(shortCode) {
		return "", fmt.Errorf("custom alias may contain only latin letters, numbers, hyphens, and underscores")
	}
	reserved := []string{
		"api", "admin", "register", "login", "logout", "stats", "health",
		"privacy", "terms", "status", "developers", "report", "assets",
	}
	if slices.Contains(reserved, shortCode) {
		return "", fmt.Errorf("custom alias is reserved")
	}

	return shortCode, nil
}

func IsReservedShortCode(value string) bool {
	_, err := ValidateShortCode(value)
	return err != nil
}

func ValidateURL(inputURL string) (string, error) {
	if inputURL == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}

	parsed, err := url.Parse(inputURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL format")
	}

	if parsed.Scheme != "" {
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			blockedSchemes := []string{"ftp", "file", "javascript", "data"}
			if slices.Contains(blockedSchemes, parsed.Scheme) {
				return "", fmt.Errorf("URL scheme not allowed")
			}
			return "", fmt.Errorf("URL scheme must be http or https")
		}
	} else {
		if !strings.Contains(inputURL, ".") || strings.Contains(inputURL, " ") {
			return "", fmt.Errorf("invalid URL format")
		}
		inputURL = "https://" + inputURL
		parsed, err = url.Parse(inputURL)
		if err != nil {
			return "", fmt.Errorf("invalid URL format")
		}
	}

	if parsed.Host == "" {
		return "", fmt.Errorf("URL must contain a host")
	}
	if len(inputURL) > 2048 {
		return "", fmt.Errorf("URL is too long")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("URLs containing credentials are not allowed")
	}
	hostname := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if hostname == "" {
		return "", fmt.Errorf("URL must contain a valid host")
	}
	if hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") ||
		strings.HasSuffix(hostname, ".local") || strings.HasSuffix(hostname, ".internal") {
		return "", fmt.Errorf("local and internal destinations are not allowed")
	}
	if ip := net.ParseIP(hostname); ip != nil && isUnsafeIP(ip) {
		return "", fmt.Errorf("private and reserved IP destinations are not allowed")
	}

	parsed.Host = strings.ToLower(parsed.Host)
	return parsed.String(), nil
}

func isUnsafeIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

func ValidateEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	if len(email) > 254 {
		return "", fmt.Errorf("email address is too long")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || !strings.Contains(email, "@") {
		return "", fmt.Errorf("enter a valid email address")
	}
	return email, nil
}

func ValidatePassword(value string) error {
	if len(value) < 10 {
		return fmt.Errorf("password must be at least 10 characters")
	}
	if len([]byte(value)) > 72 {
		return fmt.Errorf("password must be no longer than 72 bytes")
	}
	if !passwordLetterPattern.MatchString(value) || !passwordNumberPattern.MatchString(value) {
		return fmt.Errorf("password must contain at least one letter and one number")
	}
	return nil
}

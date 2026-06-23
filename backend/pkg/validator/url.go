package validator

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
)

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

    parsed.Host = strings.ToLower(parsed.Host)
    return parsed.String(), nil
}
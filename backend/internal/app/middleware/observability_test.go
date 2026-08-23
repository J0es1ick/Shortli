package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestMetricsClientIDDoesNotExposeAddress(t *testing.T) {
	registry := NewMetricsRegistry(NewClientIPResolver(""), "0123456789abcdef0123456789abcdef")
	request := httptest.NewRequest("GET", "http://example.com", nil)
	request.RemoteAddr = "203.0.113.10:443"
	first := registry.clientID(request)
	second := registry.clientID(request)
	if first != second || first == "203.0.113.10" || len(first) != 32 {
		t.Fatalf("clientID() returned %q and %q", first, second)
	}
}

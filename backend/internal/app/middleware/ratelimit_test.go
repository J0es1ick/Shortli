package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestClientIPResolverIgnoresUntrustedHeaders(t *testing.T) {
	resolver := NewClientIPResolver("172.16.0.0/12")
	request := httptest.NewRequest("GET", "http://example.com", nil)
	request.RemoteAddr = "203.0.113.10:443"
	request.Header.Set("X-Forwarded-For", "198.51.100.7")
	if actual := resolver.Resolve(request); actual != "203.0.113.10" {
		t.Fatalf("Resolve() = %q, want direct peer", actual)
	}
}

func TestClientIPResolverWalksTrustedChainFromRight(t *testing.T) {
	resolver := NewClientIPResolver("172.16.0.0/12,10.0.0.0/8")
	request := httptest.NewRequest("GET", "http://example.com", nil)
	request.RemoteAddr = "172.20.0.5:43178"
	request.Header.Set("X-Forwarded-For", "198.51.100.7, 10.2.0.8")
	if actual := resolver.Resolve(request); actual != "198.51.100.7" {
		t.Fatalf("Resolve() = %q, want original client", actual)
	}
}

func TestClientIPResolverDoesNotTrustSpoofedLeftmostValue(t *testing.T) {
	resolver := NewClientIPResolver("172.16.0.0/12")
	request := httptest.NewRequest("GET", "http://example.com", nil)
	request.RemoteAddr = "172.20.0.5:43178"
	request.Header.Set("X-Forwarded-For", "192.0.2.1, 203.0.113.9")
	if actual := resolver.Resolve(request); actual != "203.0.113.9" {
		t.Fatalf("Resolve() = %q, want nearest untrusted proxy", actual)
	}
}

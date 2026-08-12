package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	raw, prefix, hash, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey returned error: %v", err)
	}
	if !strings.HasPrefix(raw, "sk_live_") || len(raw) < 40 {
		t.Fatalf("unexpected raw key format: %q", raw)
	}
	if prefix != raw[:16] {
		t.Fatalf("prefix = %q", prefix)
	}
	digest := sha256.Sum256([]byte(raw))
	if hash != hex.EncodeToString(digest[:]) {
		t.Fatal("stored hash does not match raw key")
	}
}

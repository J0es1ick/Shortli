package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

func GenerateAPIKey() (raw, prefix, hash string, err error) {
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return "", "", "", err
	}
	raw = "sk_live_" + base64.RawURLEncoding.EncodeToString(secret)
	prefix = raw[:16]
	digest := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(digest[:])
	return raw, prefix, hash, nil
}

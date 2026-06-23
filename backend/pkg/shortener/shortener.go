package shortener

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateShortCode(originalURL string, attempt int) string {
	if attempt == 0 {
		return generateDeterministicCode(originalURL)
	}

	return generateRandomCode()
}

func generateDeterministicCode(originalURL string) string {
	hash := uint32(5381)
	for i := 0; i < len(originalURL); i++ {
		hash = ((hash << 5) + hash) + uint32(originalURL[i])
	}
	
	return uint32ToHex(hash)
}

func generateRandomCode() string {
	bytes := make([]byte, 4) 
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func uint32ToHex(num uint32) string {
	const hexChars = "0123456789abcdef"
	result := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		result[i] = hexChars[num&0xF]
		num >>= 4
	}
	return string(result)
}
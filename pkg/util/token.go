package util

import (
	"crypto/rand"
	"encoding/base64"
)

func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)

	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Encode to base64 URL-safe (no `+`, `/`, `=`)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes), nil
}

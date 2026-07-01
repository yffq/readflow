package middleware

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func GenerateAPIKey() (prefix string, rawKey string, keyHash string) {
	b := make([]byte, 32)
	rand.Read(b)
	rawKey = "rf_" + hex.EncodeToString(b)
	prefix = rawKey[:11]
	hash := sha256.Sum256([]byte(rawKey))
	keyHash = hex.EncodeToString(hash[:])
	return prefix, rawKey, keyHash
}

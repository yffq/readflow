package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

func GenerateAPIKey() (prefix string, rawKey string, keyHash string) {
	b := make([]byte, 32)
	rand.Read(b)
	rawKey = "rf_" + hex.EncodeToString(b)
	prefix = rawKey[:11]
	hash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	keyHash = string(hash)
	return prefix, rawKey, keyHash
}

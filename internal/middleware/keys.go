package middleware

import (
	"crypto/rand"
	"crypto/sha256"
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

func GenerateWebhookToken() (rawToken string, tokenHash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	rawToken = "wh_" + hex.EncodeToString(b)
	hash := sha256.Sum256([]byte(rawToken))
	return rawToken, hex.EncodeToString(hash[:]), nil
}

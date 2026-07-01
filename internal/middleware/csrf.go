package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
)

type CSRFManager interface {
	LoadAndSave(next http.Handler) http.Handler
}

func GetCSRFToken(r *http.Request, getter func(r *http.Request, key string) string) string {
	token := getter(r, "csrf_token")
	if token == "" {
		token = generateCSRFToken()
	}
	return token
}

func CSRFToken(r *http.Request, getter func(r *http.Request, key string) string, setter func(r *http.Request, key, val string)) string {
	token := getter(r, "csrf_token")
	if token == "" {
		token = generateCSRFToken()
		setter(r, "csrf_token", token)
	}
	return token
}

func ValidateCSRF(r *http.Request, getter func(r *http.Request, key string) string) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return true
	}
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		return true
	}

	stored := getter(r, "csrf_token")
	sent := r.Header.Get("X-CSRF-Token")
	if sent == "" {
		sent = r.FormValue("csrf_token")
	}

	return sent != "" && sent == stored
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

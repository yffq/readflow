package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/readflow/readflow/internal/store"
)

type contextKey string

const ContextUserID contextKey = "user_id"

func AuthRequired(sm *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := sm.GetString(r.Context(), "user_id")
			if userID == "" {
				acceptsJSON := strings.Contains(r.Header.Get("Accept"), "application/json")
				isFetch := strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest")
				if acceptsJSON || isFetch {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error":"login required"}`))
					return
				}
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			ctx := context.WithValue(r.Context(), ContextUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(r *http.Request) string {
	if v := r.Context().Value(ContextUserID); v != nil {
		return v.(string)
	}
	return ""
}

func APIKeyAuth(s *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := APIKeyFromRequest(r)
			if key == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}

			hash := sha256.Sum256([]byte(key))
			keyHash := hex.EncodeToString(hash[:])

			if err := s.ValidateAPIKey(keyHash); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"invalid api key"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func APIKeyFromRequest(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		auth = strings.TrimSpace(r.URL.Query().Get("api_key"))
	}
	if auth == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	if strings.HasPrefix(auth, "rf_") {
		return auth
	}
	return ""
}

func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "0")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		if os.Getenv("READFLOW_ENV") == "production" {
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		}

		isStatic := strings.HasPrefix(r.URL.Path, "/static/")
		if isStatic {
			w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'self' 'unsafe-inline'")
		} else {
			w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src * data:; media-src *")
		}

		next.ServeHTTP(w, r)
	})
}

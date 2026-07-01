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
			auth := r.Header.Get("Authorization")
			if auth == "" {
				auth = r.URL.Query().Get("api_key")
			}

			if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}

			key := strings.TrimPrefix(auth, "Bearer ")
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

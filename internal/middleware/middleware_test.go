package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	prefix, rawKey, keyHash := GenerateAPIKey()

	if len(rawKey) == 0 {
		t.Fatal("rawKey is empty")
	}
	if !hasPrefix(rawKey, "rf_") {
		t.Fatal("rawKey should start with rf_")
	}
	if len(prefix) != 11 {
		t.Fatalf("prefix should be 11 chars, got %d", len(prefix))
	}
	if len(keyHash) != 64 {
		t.Fatalf("keyHash should be 64 hex chars, got %d", len(keyHash))
	}

	prefix2, _, keyHash2 := GenerateAPIKey()
	if prefix2 == prefix {
		t.Log("prefix collision (unlikely but possible)")
	}
	if keyHash2 == keyHash {
		t.Fatal("hash collision — two keys should not have the same hash")
	}
}

func TestSecureHeaders(t *testing.T) {
	handler := SecureHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	headers := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Referrer-Policy",
		"Permissions-Policy",
		"Content-Security-Policy",
	}
	for _, h := range headers {
		if rec.Header().Get(h) == "" {
			t.Errorf("expected %s header to be set", h)
		}
	}
}

func TestAPIKeyAuth(t *testing.T) {
	// This requires a store with a valid key, so just test the middleware chain logic
	t.Run("no auth header", func(t *testing.T) {
		called := false
		handler := APIKeyAuth(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))

		req := httptest.NewRequest("GET", "/api/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if called {
			t.Fatal("handler should not be called without auth")
		}
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("malformed auth header", func(t *testing.T) {
		called := false
		handler := APIKeyAuth(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if called {
			t.Fatal("handler should not be called with bad auth header")
		}
	})
}

func TestRateLimit(t *testing.T) {
	// Allow 2 requests per second, burst 3
	handler := RateLimit(2, 3)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("request %d should be allowed, got %d", i, rec.Code)
		}
	}

	// 4th request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
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

func TestAuthRequiredJSONRequest(t *testing.T) {
	called := false
	sm := scs.New()
	handler := sm.LoadAndSave(AuthRequired(sm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})))

	req := httptest.NewRequest("POST", "/save-link", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Fatal("handler should not be called without login")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected JSON content type, got %q", rec.Header().Get("Content-Type"))
	}
	if rec.Body.String() != `{"error":"login required"}` {
		t.Fatalf("unexpected body: %s", rec.Body.String())
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

func TestAPIKeyFromRequest(t *testing.T) {
	tests := []struct {
		name string
		auth string
		url  string
		want string
	}{
		{name: "bearer header", auth: "Bearer rf_test", url: "/api/test", want: "rf_test"},
		{name: "raw query", url: "/api/test?api_key=rf_test", want: "rf_test"},
		{name: "bearer query", url: "/api/test?api_key=Bearer%20rf_test", want: "rf_test"},
		{name: "invalid header", auth: "Token rf_test", url: "/api/test", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			if got := APIKeyFromRequest(req); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
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

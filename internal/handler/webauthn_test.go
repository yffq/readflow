package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func webauthnRequest(t *testing.T, h *Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", path, nil)
	req.Host = "localhost:8080"
	rec := httptest.NewRecorder()
	h.SM.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch path {
		case "/webauthn/register/begin":
			h.WebAuthnRegisterBegin(w, r)
		case "/webauthn/register/finish":
			h.WebAuthnRegisterFinish(w, r)
		case "/webauthn/login/begin":
			h.WebAuthnLoginBegin(w, r)
		case "/webauthn/login/finish":
			h.WebAuthnLoginFinish(w, r)
		}
	})).ServeHTTP(rec, req)
	return rec
}

func TestWebAuthnRegisterBegin(t *testing.T) {
	h := setupTestHandler(t)
	rec := webauthnRequest(t, h, "/webauthn/register/begin")

	if rec.Code != http.StatusOK {
		t.Fatalf("register begin: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		PublicKey struct {
			Challenge string `json:"challenge"`
		} `json:"publicKey"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode register begin: %v", err)
	}
	if body.PublicKey.Challenge == "" {
		t.Fatal("expected a challenge in register begin response")
	}
}

func TestWebAuthnLoginBegin(t *testing.T) {
	h := setupTestHandler(t)
	// A login assertion requires at least one stored credential.
	if err := h.Store.CreateWebAuthnCredential("default", "cred-1", []byte(`{"id":"Y3JlZC0x"}`)); err != nil {
		t.Fatalf("seed credential: %v", err)
	}

	rec := webauthnRequest(t, h, "/webauthn/login/begin")

	if rec.Code != http.StatusOK {
		t.Fatalf("login begin: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		PublicKey struct {
			Challenge string `json:"challenge"`
		} `json:"publicKey"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode login begin: %v", err)
	}
	if body.PublicKey.Challenge == "" {
		t.Fatal("expected a challenge in login begin response")
	}
}

func TestWebAuthnRegisterFinishMissingSession(t *testing.T) {
	h := setupTestHandler(t)
	rec := webauthnRequest(t, h, "/webauthn/register/finish")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("register finish: expected 400 without session, got %d", rec.Code)
	}
}

func TestWebAuthnLoginFinishMissingSession(t *testing.T) {
	h := setupTestHandler(t)
	rec := webauthnRequest(t, h, "/webauthn/login/finish")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("login finish: expected 400 without session, got %d", rec.Code)
	}
}

func TestWebAuthnCredentialsEmpty(t *testing.T) {
	h := setupTestHandler(t)
	req := httptest.NewRequest("GET", "/webauthn/credentials", nil)
	rec := httptest.NewRecorder()
	h.SM.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.WebAuthnCredentials(w, r)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("credentials: expected 200, got %d", rec.Code)
	}
	var body struct {
		Credentials []string `json:"credentials"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode credentials: %v", err)
	}
	if body.Credentials == nil {
		t.Fatal("expected credentials array, got nil")
	}
}

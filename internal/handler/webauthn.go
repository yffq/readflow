package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-webauthn/webauthn/webauthn"
)

const (
	webauthnRegisterSessionKey = "webauthn_register_session"
	webauthnLoginSessionKey    = "webauthn_login_session"
	webauthnUserID             = "default"
)

// webauthnUser is the single Readflow user for WebAuthn purposes.
type webauthnUser struct {
	id          []byte
	credentials []webauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte                         { return u.id }
func (u *webauthnUser) WebAuthnName() string                       { return "readflow" }
func (u *webauthnUser) WebAuthnDisplayName() string                { return "Readflow" }
func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// webauthnInstance builds a WebAuthn instance bound to the request origin.
func (h *Handler) webauthnInstance(r *http.Request) (*webauthn.WebAuthn, error) {
	rpID := r.Host
	if i := strings.IndexByte(rpID, ':'); i != -1 {
		rpID = rpID[:i]
	}

	scheme := "https"
	if os.Getenv("READFLOW_ENV") != "production" {
		scheme = "http"
	}

	return webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: "Readflow",
		RPOrigins:     []string{scheme + "://" + r.Host},
	})
}

// loadWebAuthnUser loads the single user and its stored credentials.
func (h *Handler) loadWebAuthnUser(ctx context.Context) (*webauthnUser, error) {
	rows, err := h.Store.ListWebAuthnCredentials(webauthnUserID)
	if err != nil {
		return nil, err
	}
	creds := make([]webauthn.Credential, 0, len(rows))
	for _, row := range rows {
		var c webauthn.Credential
		if err := json.Unmarshal(row.Data, &c); err != nil {
			log.Printf("skip invalid webauthn credential %s: %v", row.ID, err)
			continue
		}
		creds = append(creds, c)
	}
	return &webauthnUser{id: []byte(webauthnUserID), credentials: creds}, nil
}

func (h *Handler) storeWebAuthnSession(ctx context.Context, key string, sd *webauthn.SessionData) error {
	b, err := json.Marshal(sd)
	if err != nil {
		return err
	}
	h.SM.Put(ctx, key, string(b))
	_, _, err = h.SM.Commit(ctx)
	return err
}

func (h *Handler) loadWebAuthnSession(ctx context.Context, key string) (*webauthn.SessionData, error) {
	raw := h.SM.GetString(ctx, key)
	if raw == "" {
		return nil, fmt.Errorf("webauthn session not found")
	}
	var sd webauthn.SessionData
	if err := json.Unmarshal([]byte(raw), &sd); err != nil {
		return nil, err
	}
	return &sd, nil
}

// WebAuthnRegisterBegin starts a passkey registration. Requires an authenticated
// session (the user is already logged in via password).
func (h *Handler) WebAuthnRegisterBegin(w http.ResponseWriter, r *http.Request) {
	user, err := h.loadWebAuthnUser(r.Context())
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to load credentials")
		return
	}
	wa, err := h.webauthnInstance(r)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to configure webauthn")
		return
	}
	options, sessionData, err := wa.BeginRegistration(user)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to begin registration")
		return
	}
	if err := h.storeWebAuthnSession(r.Context(), webauthnRegisterSessionKey, sessionData); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to store registration session")
		return
	}
	writeJSON(w, http.StatusOK, options)
}

// WebAuthnRegisterFinish completes a passkey registration and stores the credential.
func (h *Handler) WebAuthnRegisterFinish(w http.ResponseWriter, r *http.Request) {
	user, err := h.loadWebAuthnUser(r.Context())
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to load credentials")
		return
	}
	sessionData, err := h.loadWebAuthnSession(r.Context(), webauthnRegisterSessionKey)
	if err != nil {
		h.jsonError(w, http.StatusBadRequest, "registration session missing or expired")
		return
	}
	wa, err := h.webauthnInstance(r)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to configure webauthn")
		return
	}
	credential, err := wa.FinishRegistration(user, *sessionData, r)
	if err != nil {
		h.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	data, err := json.Marshal(credential)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to serialize credential")
		return
	}
	credentialID := base64.RawURLEncoding.EncodeToString(credential.ID)
	if err := h.Store.CreateWebAuthnCredential(webauthnUserID, credentialID, data); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to store credential")
		return
	}
	h.SM.Remove(r.Context(), webauthnRegisterSessionKey)
	h.SM.Commit(r.Context())
	writeJSON(w, http.StatusOK, map[string]string{"status": "registered", "id": credentialID})
}

// WebAuthnLoginBegin starts a passkey login.
func (h *Handler) WebAuthnLoginBegin(w http.ResponseWriter, r *http.Request) {
	user, err := h.loadWebAuthnUser(r.Context())
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to load credentials")
		return
	}
	wa, err := h.webauthnInstance(r)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to configure webauthn")
		return
	}
	options, sessionData, err := wa.BeginLogin(user)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to begin login")
		return
	}
	if err := h.storeWebAuthnSession(r.Context(), webauthnLoginSessionKey, sessionData); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to store login session")
		return
	}
	writeJSON(w, http.StatusOK, options)
}

// WebAuthnLoginFinish verifies a passkey assertion and establishes a session.
func (h *Handler) WebAuthnLoginFinish(w http.ResponseWriter, r *http.Request) {
	user, err := h.loadWebAuthnUser(r.Context())
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to load credentials")
		return
	}
	sessionData, err := h.loadWebAuthnSession(r.Context(), webauthnLoginSessionKey)
	if err != nil {
		h.jsonError(w, http.StatusBadRequest, "login session missing or expired")
		return
	}
	wa, err := h.webauthnInstance(r)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to configure webauthn")
		return
	}
	if _, err := wa.FinishLogin(user, *sessionData, r); err != nil {
		h.jsonError(w, http.StatusUnauthorized, err.Error())
		return
	}

	h.SM.Remove(r.Context(), webauthnLoginSessionKey)
	h.SM.Put(r.Context(), "user_id", webauthnUserID)
	if _, _, err := h.SM.Commit(r.Context()); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to establish session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// WebAuthnCredentials lists stored passkeys (id + created time) for the settings page.
func (h *Handler) WebAuthnCredentials(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Store.ListWebAuthnCredentials(webauthnUserID)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to list credentials")
		return
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"credentials": ids})
}

// WebAuthnCredentialDelete removes a stored passkey.
func (h *Handler) WebAuthnCredentialDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.jsonError(w, http.StatusBadRequest, "credential id is required")
		return
	}
	if err := h.Store.DeleteWebAuthnCredential(webauthnUserID, id); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to delete credential")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

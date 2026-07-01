package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/readflow/readflow/internal/model"
)

func TestServerIntegration(t *testing.T) {
	projectRoot := findProjectRoot()
	migrationFile := filepath.Join(projectRoot, "migrations", "001_init.sql")
	if _, err := os.Stat(migrationFile); err != nil {
		t.Skipf("migration file not found: %v (skipping integration test)", err)
	}

	srv, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ts := httptest.NewServer(srv.http.Handler)
	defer ts.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	t.Run("Setup", func(t *testing.T) {
		resp := doRequest(t, client, ts.URL+"/setup", "GET", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("setup page: expected 200, got %d", resp.StatusCode)
		}

		form := strings.NewReader("password=testpass123")
		resp = doRequest(t, client, ts.URL+"/setup", "POST", map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		}, form)
		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("setup: expected 303, got %d", resp.StatusCode)
		}
		cookies := resp.Cookies()
		if len(cookies) == 0 {
			t.Fatal("expected session cookie after setup")
		}
	})

	t.Run("Login", func(t *testing.T) {
		resp := doRequest(t, client, ts.URL+"/login", "GET", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("login page: expected 200, got %d", resp.StatusCode)
		}

		form := strings.NewReader("password=wrong")
		resp = doRequest(t, client, ts.URL+"/login", "POST", map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		}, form)
		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("bad login: expected 303, got %d", resp.StatusCode)
		}
		loc := resp.Header.Get("Location")
		if !strings.Contains(loc, "error") {
			t.Fatalf("expected error in redirect, got %s", loc)
		}
	})

	t.Run("LoginSuccess_RedirectToIndex", func(t *testing.T) {
		form := strings.NewReader("password=testpass123")
		resp := doRequest(t, client, ts.URL+"/login", "POST", map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		}, form)
		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("login: expected 303, got %d", resp.StatusCode)
		}
		cookies := resp.Cookies()
		if len(cookies) == 0 {
			t.Fatal("expected session cookie after login")
		}
	})

	t.Run("SaveAPI_NoAuth", func(t *testing.T) {
		body := model.SaveRequest{
			HTML:  "<p>Hello</p>",
			Title: "Test",
		}
		b, _ := json.Marshal(body)
		resp := doRequest(t, client, ts.URL+"/api/v1/save", "POST", map[string]string{
			"Content-Type": "application/json",
		}, bytes.NewReader(b))
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 without auth, got %d", resp.StatusCode)
		}
	})

	t.Run("ExportAPI_NoAuth", func(t *testing.T) {
		resp := doRequest(t, client, ts.URL+"/api/v1/export", "GET", nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 without auth, got %d", resp.StatusCode)
		}
	})
}

func doRequest(t *testing.T, client *http.Client, url, method string, headers map[string]string, body io.Reader) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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

	newClient := func() *http.Client {
		return &http.Client{
			Jar: newCookieJar(),
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}

	t.Run("Setup", func(t *testing.T) {
		client := newClient()
		resp := doRequest(t, client, ts.URL+"/setup", "GET", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("setup page: expected 200, got %d", resp.StatusCode)
		}
		csrf := extractCSRF(resp)

		form := url.Values{"password": {"testpass123"}, "csrf_token": {csrf}}
		resp = doRequest(t, client, ts.URL+"/setup", "POST", map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		}, strings.NewReader(form.Encode()))
		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("setup: expected 303, got %d", resp.StatusCode)
		}
		cookies := resp.Cookies()
		if len(cookies) == 0 {
			t.Fatal("expected session cookie after setup")
		}
	})

	t.Run("Login", func(t *testing.T) {
		client := newClient()
		resp := doRequest(t, client, ts.URL+"/login", "GET", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("login page: expected 200, got %d", resp.StatusCode)
		}
		csrf := extractCSRF(resp)

		form := url.Values{"password": {"wrong"}, "csrf_token": {csrf}}
		resp = doRequest(t, client, ts.URL+"/login", "POST", map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		}, strings.NewReader(form.Encode()))
		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("bad login: expected 303, got %d", resp.StatusCode)
		}
		loc := resp.Header.Get("Location")
		if !strings.Contains(loc, "error") {
			t.Fatalf("expected error in redirect, got %s", loc)
		}
	})

	t.Run("LoginSuccess_RedirectToIndex", func(t *testing.T) {
		client := newClient()
		resp := doRequest(t, client, ts.URL+"/login", "GET", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("login page: expected 200, got %d", resp.StatusCode)
		}
		csrf := extractCSRF(resp)

		form := url.Values{"password": {"testpass123"}, "csrf_token": {csrf}}
		resp = doRequest(t, client, ts.URL+"/login", "POST", map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		}, strings.NewReader(form.Encode()))
		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("login: expected 303, got %d", resp.StatusCode)
		}
		cookies := resp.Cookies()
		if len(cookies) == 0 {
			t.Fatal("expected session cookie after login")
		}
	})

	t.Run("SaveAPI_NoAuth", func(t *testing.T) {
		client := newClient()
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
		client := newClient()
		resp := doRequest(t, client, ts.URL+"/api/v1/export", "GET", nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 without auth, got %d", resp.StatusCode)
		}
	})

	t.Run("InoreaderWebhook_Auth", func(t *testing.T) {
		client := newClient()
		body := strings.NewReader("{\"items\":[{\"title\":\"From Inoreader\",\"canonical\":[{\"href\":\"https://example.com/webhook-route\"}],\"summary\":{\"content\":\"<p>Webhook body</p>\"}}]}")
		resp := doRequest(t, client, ts.URL+"/api/v1/webhooks/inoreader", "POST", map[string]string{
			"Content-Type": "application/json",
		}, body)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 without API key, got %d", resp.StatusCode)
		}

		rawKey := "rf_inoreader_test_key"
		hash := sha256.Sum256([]byte(rawKey))
		userExists, err := srv.store.UserExists()
		if err != nil {
			t.Fatalf("check test user: %v", err)
		}
		if !userExists {
			if _, err := srv.store.CreateUser("inoreader-test-password"); err != nil {
				t.Fatalf("create webhook test user: %v", err)
			}
		}
		if err := srv.store.CreateAPIKey("default", rawKey[:11], hex.EncodeToString(hash[:]), "Inoreader"); err != nil {
			t.Fatalf("create webhook API key: %v", err)
		}
		body = strings.NewReader("{\"items\":[{\"title\":\"From Inoreader\",\"canonical\":[{\"href\":\"https://example.com/webhook-route\"}],\"summary\":{\"content\":\"<p>Webhook body</p>\"}}]}")
		resp = doRequest(t, client, ts.URL+"/api/v1/webhooks/inoreader/"+url.PathEscape(rawKey), "POST", map[string]string{
			"Content-Type": "application/json",
		}, body)
		if resp.StatusCode != http.StatusOK {
			responseBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read webhook response: %v", err)
			}
			t.Fatalf("expected 200 with API key, got %d: %s", resp.StatusCode, responseBody)
		}
		var webhookResponse struct {
			Created    int `json:"created"`
			Duplicates int `json:"duplicates"`
			Failed     int `json:"failed"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&webhookResponse); err != nil {
			t.Fatalf("decode webhook response: %v", err)
		}
		resp.Body.Close()
		if webhookResponse.Created != 1 || webhookResponse.Duplicates != 0 || webhookResponse.Failed != 0 {
			t.Fatalf("unexpected webhook response: %+v", webhookResponse)
		}

		resp = doRequest(t, client, ts.URL+"/api/v1/export?api_key="+url.QueryEscape(rawKey), "GET", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("export after webhook: expected 200, got %d", resp.StatusCode)
		}
		var exported model.ExportResponse
		if err := json.NewDecoder(resp.Body).Decode(&exported); err != nil {
			t.Fatalf("decode exported articles: %v", err)
		}
		resp.Body.Close()
		if len(exported.Results) != 1 {
			t.Fatalf("expected one exported article, got %d", len(exported.Results))
		}
		article := exported.Results[0]
		if article.Title != "From Inoreader" || article.URL != "https://example.com/webhook-route" ||
			article.Source != "inoreader" || !strings.Contains(article.ContentMarkdown, "Webhook body") {
			t.Fatalf("unexpected exported article: %+v", article)
		}

		body = strings.NewReader("{\"items\":[{\"title\":\"From Inoreader\",\"canonical\":[{\"href\":\"https://example.com/webhook-route\"}],\"summary\":{\"content\":\"<p>Webhook body</p>\"}}]}")
		resp = doRequest(t, client, ts.URL+"/api/v1/webhooks/inoreader/"+url.PathEscape(rawKey), "POST", map[string]string{
			"Content-Type": "application/json",
		}, body)
		if err := json.NewDecoder(resp.Body).Decode(&webhookResponse); err != nil {
			t.Fatalf("decode duplicate webhook response: %v", err)
		}
		resp.Body.Close()
		if webhookResponse.Created != 0 || webhookResponse.Duplicates != 1 || webhookResponse.Failed != 0 {
			t.Fatalf("unexpected duplicate webhook response: %+v", webhookResponse)
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

func extractCSRF(resp *http.Response) string {
	body, _ := io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(bytes.NewReader(body))
	html := string(body)
	idx := strings.Index(html, `name="csrf_token" value="`)
	if idx == -1 {
		return ""
	}
	start := idx + len(`name="csrf_token" value="`)
	end := strings.Index(html[start:], `"`)
	if end == -1 {
		return ""
	}
	return html[start : start+end]
}

func newCookieJar() http.CookieJar {
	return &testCookieJar{jar: make(map[string][]*http.Cookie)}
}

type testCookieJar struct {
	jar map[string][]*http.Cookie
}

func (j *testCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.jar[u.Host] = cookies
}

func (j *testCookieJar) Cookies(u *url.URL) []*http.Cookie {
	return j.jar[u.Host]
}

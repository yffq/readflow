package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/sqlite3store"
	"github.com/readflow/readflow/internal/model"
	"github.com/readflow/readflow/internal/store"
)

func setupTestHandler(t *testing.T) *Handler {
	t.Helper()

	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	migrationSQL := `
	CREATE TABLE IF NOT EXISTS sessions (token TEXT PRIMARY KEY, data BLOB NOT NULL, expiry DATETIME NOT NULL);
	CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY, password_hash TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT (datetime('now')));
	CREATE TABLE IF NOT EXISTS api_keys (id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, key_prefix TEXT NOT NULL, key_hash TEXT NOT NULL UNIQUE, name TEXT NOT NULL DEFAULT '', last_used DATETIME, created_at DATETIME NOT NULL DEFAULT (datetime('now')));
	CREATE TABLE IF NOT EXISTS articles (id TEXT PRIMARY KEY, title TEXT NOT NULL DEFAULT '', url TEXT DEFAULT '', content_html TEXT NOT NULL DEFAULT '', content_md TEXT NOT NULL DEFAULT '', author TEXT NOT NULL DEFAULT '', site_name TEXT NOT NULL DEFAULT '', word_count INTEGER NOT NULL DEFAULT 0, source TEXT NOT NULL DEFAULT 'url', extraction_failed INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL DEFAULT 'unread', created_at DATETIME NOT NULL DEFAULT (datetime('now')), updated_at DATETIME NOT NULL DEFAULT (datetime('now')));
	`
	if err := s.Migrate(migrationSQL); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	sm := scs.New()
	sm.Store = sqlite3store.New(s.DB())
	sm.Lifetime = 7 * 24 * time.Hour

	tmpl := loadTestTemplates(t)
	h := New(s, tmpl, sm)
	h.SetupDone = false
	return h
}

func loadTestTemplates(t *testing.T) *template.Template {
	t.Helper()

	cwd, _ := os.Getwd()
	projectRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatalf("cannot find project root")
		}
		projectRoot = parent
	}

	templateDir := filepath.Join(projectRoot, "template")

	funcMap := template.FuncMap{
		"lower":    strings.ToLower,
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		"formatTime": func(s string) string {
			if len(s) >= 10 {
				return s[:10]
			}
			return s
		},
		"truncate": func(s string, n int) string {
			if len(s) > n {
				return s[:n] + "..."
			}
			return s
		},
		"add":      func(a, b int) int { return a + b },
		"subtract": func(a, b int) int { return a - b },
		"multiply": func(a, b int) int { return a * b },
	}

	tmpl := template.New("").Funcs(funcMap)
	return template.Must(tmpl.ParseGlob(filepath.Join(templateDir, "*.html")))
}

func withSession(h *Handler, handler func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.SM.LoadAndSave(http.HandlerFunc(handler)).ServeHTTP(w, r)
	}
}

func TestSetupFlow(t *testing.T) {
	h := setupTestHandler(t)

	// GET /setup
	req := httptest.NewRequest("GET", "/setup", nil)
	rec := httptest.NewRecorder()
	withSession(h, h.SetupPage).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("setup page: expected 200, got %d", rec.Code)
	}

	// POST /setup with short password
	form := url.Values{"password": {"ab"}}
	req = httptest.NewRequest("POST", "/setup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	withSession(h, h.Setup).ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("setup with short pass: expected 303, got %d", rec.Code)
	}

	// POST /setup with valid password
	form = url.Values{"password": {"testpass123"}}
	req = httptest.NewRequest("POST", "/setup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	withSession(h, h.Setup).ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("setup: expected 303, got %d", rec.Code)
	}

	if !h.SetupDone {
		t.Fatal("expected setup to be done")
	}
}

func TestLoginFlow(t *testing.T) {
	h := setupTestHandler(t)
	h.Store.CreateUser("testpass123")
	h.SetupDone = true

	// GET /login
	req := httptest.NewRequest("GET", "/login", nil)
	rec := httptest.NewRecorder()
	withSession(h, h.LoginPage).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login page: expected 200, got %d", rec.Code)
	}

	// POST invalid password
	form := url.Values{"password": {"wrong"}}
	req = httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	withSession(h, h.Login).ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("bad login: expected 303, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "error") {
		t.Fatalf("expected error redirect, got %s", loc)
	}

	// POST valid password
	form = url.Values{"password": {"testpass123"}}
	req = httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	withSession(h, h.Login).ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("login: expected 303, got %d", rec.Code)
	}
}

func TestSaveAPI_HTML(t *testing.T) {
	h := setupTestHandler(t)

	body := model.SaveRequest{
		HTML:  "<p>Hello <b>World</b></p><script>alert('xss')</script>",
		Title: "Test Article",
	}

	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/save", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APISave(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("save API: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["status"] != "created" {
		t.Fatalf("expected status 'created', got %v", resp["status"])
	}
}

func TestSaveAPI_URL(t *testing.T) {
	h := setupTestHandler(t)

	body := model.SaveRequest{
		URL: "https://example.com",
	}

	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/save", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APISave(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("save API (url): expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["status"] != "created" && resp["status"] != "duplicate" {
		t.Fatalf("expected status 'created', got %v", resp["status"])
	}
}

func TestSaveAPI_InvalidRequest(t *testing.T) {
	h := setupTestHandler(t)

	// Empty request
	body := model.SaveRequest{}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/save", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APISave(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty request: expected 400, got %d", rec.Code)
	}

	// Malformed JSON
	req = httptest.NewRequest("POST", "/api/v1/save", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.APISave(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad json: expected 400, got %d", rec.Code)
	}
}

func TestExportAPI(t *testing.T) {
	h := setupTestHandler(t)

	// Save a test article first
	a := testArticle("export-test-1", "Export Test", "export", time.Now())
	h.Store.CreateArticle(a)

	// Test export
	req := httptest.NewRequest("GET", "/api/v1/export?limit=10", nil)
	rec := httptest.NewRecorder()
	h.APIExport(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("export API: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp model.ExportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse export response: %v", err)
	}
	if resp.Count != 1 {
		t.Fatalf("expected count 1, got %d", resp.Count)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	result := resp.Results[0]
	if result.Title != "Export Test" {
		t.Fatalf("expected 'Export Test', got '%s'", result.Title)
	}
}

func TestExportAPI_UpdatedAfter(t *testing.T) {
	h := setupTestHandler(t)

	// Save articles at different times
	old := testArticle("old-1", "Old Article", "url", time.Now().Add(-2*time.Hour))
	h.Store.CreateArticle(old)

	new := testArticle("new-1", "New Article", "url", time.Now())
	h.Store.CreateArticle(new)

	// Query with updated_after (should get only new article)
	since := time.Now().Add(-1 * time.Hour)
	params := url.Values{}
	params.Set("updated_after", since.Format(time.RFC3339))
	params.Set("limit", "10")

	req := httptest.NewRequest("GET", "/api/v1/export?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	h.APIExport(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("export API: expected 200, got %d", rec.Code)
	}

	var resp model.ExportResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Count != 1 {
		t.Fatalf("expected count 1 with updated_after, got %d", resp.Count)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
}

func TestExportAPI_Pagination(t *testing.T) {
	h := setupTestHandler(t)

	for i := 0; i < 5; i++ {
		a := testArticle(fmt.Sprintf("page-%d", i), fmt.Sprintf("Article %d", i), "url", time.Now())
		h.Store.CreateArticle(a)
	}

	// Page 1: limit 2
	params := url.Values{"limit": {"2"}, "offset": {"0"}}
	req := httptest.NewRequest("GET", "/api/v1/export?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	h.APIExport(rec, req)

	var resp model.ExportResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results on page 1, got %d", len(resp.Results))
	}
	if resp.Count != 5 {
		t.Fatalf("expected total count 5, got %d", resp.Count)
	}
	if resp.Next == "" {
		t.Fatal("expected next pagination link")
	}

	// Page 2: limit 2, offset 2
	params = url.Values{"limit": {"2"}, "offset": {"2"}}
	req = httptest.NewRequest("GET", "/api/v1/export?"+params.Encode(), nil)
	rec = httptest.NewRecorder()
	h.APIExport(rec, req)

	json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results on page 2, got %d", len(resp.Results))
	}
}

func TestReadPage(t *testing.T) {
	h := setupTestHandler(t)

	a := testArticle("read-1", "Readable Article", "url", time.Now())
	a.ContentHTML = "<p>Article content with a <a href='https://example.com/link'>link</a></p>"
	h.Store.CreateArticle(a)

	// Read page by ID
	req := httptest.NewRequest("GET", "/read/read-1", nil)
	req.SetPathValue("id", "read-1")
	rec := httptest.NewRecorder()
	h.ReadPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("read page: expected 200, got %d", rec.Code)
	}

	// Verify HTML contains the article content
	body := rec.Body.String()
	if !strings.Contains(body, "Readable Article") {
		t.Fatal("response should contain article title")
	}
	if !strings.Contains(body, "link") {
		t.Fatal("response should contain link in content")
	}

	// Not found
	req = httptest.NewRequest("GET", "/read/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rec = httptest.NewRecorder()
	h.ReadPage(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("non-existent article: expected 404, got %d", rec.Code)
	}
}

func TestReadPage_ExtractionFailed(t *testing.T) {
	h := setupTestHandler(t)

	a := testArticle("ef-1", "Failed Article", "url", time.Now())
	a.ExtractionFailed = true
	h.Store.CreateArticle(a)

	req := httptest.NewRequest("GET", "/read/ef-1", nil)
	req.SetPathValue("id", "ef-1")
	rec := httptest.NewRecorder()
	h.ReadPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("read page (extraction failed): expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Open Original") && !strings.Contains(body, "open-original") && !strings.Contains(body, "Open in New Tab") {
		t.Fatal("extraction-failed page should have fallback UI")
	}
}

func TestArchiveDelete(t *testing.T) {
	h := setupTestHandler(t)

	a := testArticle("arch-1", "To Archive", "url", time.Now())
	a.Status = "unread"
	h.Store.CreateArticle(a)

	// Archive
	req := httptest.NewRequest("POST", "/archive/arch-1", nil)
	req.SetPathValue("id", "arch-1")
	rec := httptest.NewRecorder()
	h.ArchiveArticle(rec, req)
	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusOK {
		t.Fatalf("archive: expected 303 or 200, got %d", rec.Code)
	}

	got, _ := h.Store.GetArticle("arch-1")
	if got.Status != "archived" {
		t.Fatal("article should be archived")
	}

	// Delete
	req = httptest.NewRequest("POST", "/delete/arch-1", nil)
	req.SetPathValue("id", "arch-1")
	rec = httptest.NewRecorder()
	h.DeleteArticle(rec, req)
	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusOK {
		t.Fatalf("delete: expected 303 or 200, got %d", rec.Code)
	}

	got, _ = h.Store.GetArticle("arch-1")
	if got != nil {
		t.Fatal("article should be deleted")
	}
}

func testArticle(id, title, source string, t time.Time) *model.Article {
	return &model.Article{
		ID:          id,
		Title:       title,
		URL:         "https://example.com/" + id,
		ContentHTML: "<p>Test content for " + title + "</p>",
		ContentMD:   "Test content for " + title,
		Author:      "Test Author",
		SiteName:    "Example",
		WordCount:   5,
		Source:      source,
		Status:      "unread",
		CreatedAt:   t,
		UpdatedAt:   t,
	}
}

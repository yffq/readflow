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

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
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

func withCSRFSession(h *Handler, token string, handler func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.SM.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.SM.Put(r.Context(), "csrf_token", token)
			handler(w, r)
		})).ServeHTTP(w, r)
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

func TestSaveLink(t *testing.T) {
	h := setupTestHandler(t)

	body := map[string]string{
		"url":        "https://example.com/from-link",
		"csrf_token": "test-csrf",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/save-link", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.SM.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.SM.Put(r.Context(), "csrf_token", "test-csrf")
		h.SaveLink(w, r)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("save link: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse save link response: %v", err)
	}
	if resp["status"] != "created" && resp["status"] != "duplicate" {
		t.Fatalf("expected created or duplicate, got %v", resp["status"])
	}
}

func TestSaveLink_InvalidCSRF(t *testing.T) {
	h := setupTestHandler(t)

	body := map[string]string{
		"url":        "https://example.com/from-link",
		"csrf_token": "wrong-csrf",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/save-link", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.SM.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.SM.Put(r.Context(), "csrf_token", "test-csrf")
		h.SaveLink(w, r)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("invalid csrf: expected 403, got %d: %s", rec.Code, rec.Body.String())
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

func TestSaveAPI_InvalidURL(t *testing.T) {
	h := setupTestHandler(t)

	for _, rawURL := range []string{
		"ftp://example.com/file",
		"http://localhost:8080",
		"http://127.0.0.1:8080",
		"http://192.168.1.1",
		"http://169.254.169.254/latest/meta-data",
	} {
		body := model.SaveRequest{URL: rawURL}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/v1/save", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.APISave(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("save invalid URL %q: expected 400, got %d: %s", rawURL, rec.Code, rec.Body.String())
		}
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

func TestExportAPI_WithoutContent(t *testing.T) {
	h := setupTestHandler(t)

	a := testArticle("export-summary-1", "Export Summary", "export", time.Now())
	a.ContentHTML = "<p>Large article body</p>"
	a.ContentMD = "Large article body"
	h.Store.CreateArticle(a)

	req := httptest.NewRequest("GET", "/api/v1/export?limit=10&content=false", nil)
	rec := httptest.NewRecorder()
	h.APIExport(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("export summary API: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp model.ExportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse export summary response: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	result := resp.Results[0]
	if result.Title != "Export Summary" {
		t.Fatalf("expected title metadata, got %q", result.Title)
	}
	if result.ContentHTML != "" || result.ContentMarkdown != "" {
		t.Fatal("content=false should omit article body fields")
	}
}

func TestExportAPI_WithoutCount(t *testing.T) {
	h := setupTestHandler(t)

	for i := 0; i < 3; i++ {
		a := testArticle(fmt.Sprintf("summary-page-%d", i), fmt.Sprintf("Summary Page %d", i), "export", time.Now().Add(time.Duration(i)*time.Second))
		a.ContentHTML = "<p>Large article body</p>"
		h.Store.CreateArticle(a)
	}

	req := httptest.NewRequest("GET", "/api/v1/export?limit=2&content=false&count=false", nil)
	rec := httptest.NewRecorder()
	h.APIExport(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("export no-count API: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp model.ExportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse export no-count response: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}
	if !resp.HasMore {
		t.Fatal("expected has_more=true")
	}
	if resp.Next == "" {
		t.Fatal("expected next link")
	}
	if resp.Count != 2 {
		t.Fatalf("expected visible count 2, got %d", resp.Count)
	}
	if resp.Results[0].ContentHTML != "" || resp.Results[0].ContentMarkdown != "" {
		t.Fatal("count=false summary response should omit article body fields")
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

func TestReadArticleAPI(t *testing.T) {
	h := setupTestHandler(t)

	a := testArticle("api-read-1", "API Readable Article", "url", time.Now())
	a.ContentHTML = "<p>Article content for mini program</p>"
	h.Store.CreateArticle(a)

	req := httptest.NewRequest("GET", "/api/v1/article/api-read-1", nil)
	req.SetPathValue("id", "api-read-1")
	rec := httptest.NewRecorder()
	h.APIReadArticle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("read article API: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp model.ArticleExport
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse read article response: %v", err)
	}
	if resp.ID != "api-read-1" {
		t.Fatalf("expected article id api-read-1, got %s", resp.ID)
	}
	if !strings.Contains(resp.ContentHTML, "mini program") {
		t.Fatal("response should include content_html")
	}

	req = httptest.NewRequest("GET", "/api/v1/article/missing", nil)
	req.SetPathValue("id", "missing")
	rec = httptest.NewRecorder()
	h.APIReadArticle(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing article: expected 404, got %d", rec.Code)
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
	if !strings.Contains(body, `/static/app.js?v=`) {
		t.Fatal("read page should load versioned app script")
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

func TestReadMobilePage(t *testing.T) {
	h := setupTestHandler(t)

	a := testArticle("mobile-read-1", "Mobile Article", "url", time.Now())
	a.URL = "https://example.com/base/article"
	a.ContentHTML = "<p>Article content with a <a href='/link'>link</a></p>"
	h.Store.CreateArticle(a)

	req := httptest.NewRequest("GET", "/api/v1/read/mobile-read-1?api_key=Bearer%20rf_test", nil)
	req.SetPathValue("id", "mobile-read-1")
	rec := httptest.NewRecorder()
	h.ReadMobilePage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("mobile read page: expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `meta name="api-key" content="rf_test"`) {
		t.Fatal("mobile read page should expose raw api key to external script")
	}
	if !strings.Contains(body, `/static/mobile_read.js?v=`) {
		t.Fatal("mobile read page should load versioned mobile script")
	}
	if !strings.Contains(body, `data-base-url="https://example.com/base/article"`) {
		t.Fatal("mobile read page should include base URL for link resolution")
	}
	if !strings.Contains(body, `meta name="article-id" content="mobile-read-1"`) {
		t.Fatal("mobile read page should include article-id meta tag")
	}
	if !strings.Contains(body, `delete-btn`) {
		t.Fatal("mobile read page should include delete buttons")
	}
}

func TestExportAPI_NegativePagination(t *testing.T) {
	h := setupTestHandler(t)
	h.Store.CreateArticle(testArticle("neg-1", "Negative Pagination", "url", time.Now()))

	req := httptest.NewRequest("GET", "/api/v1/export?limit=-1&offset=-5&content=false&count=false", nil)
	rec := httptest.NewRecorder()
	h.APIExport(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("negative pagination: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDelete(t *testing.T) {
	h := setupTestHandler(t)

	a := testArticle("arch-1", "To Delete", "url", time.Now())
	a.Status = "unread"
	h.Store.CreateArticle(a)

	form := url.Values{"csrf_token": {"test-csrf"}}
	req := httptest.NewRequest("POST", "/delete/arch-1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "arch-1")
	rec := httptest.NewRecorder()
	withCSRFSession(h, "test-csrf", h.DeleteArticle).ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusOK {
		t.Fatalf("delete: expected 303 or 200, got %d", rec.Code)
	}

	got, _ := h.Store.GetArticle("arch-1")
	if got != nil {
		t.Fatal("article should be deleted")
	}
}

func TestDelete_InvalidCSRF(t *testing.T) {
	h := setupTestHandler(t)

	a := testArticle("arch-csrf-1", "To Delete", "url", time.Now())
	a.Status = "unread"
	h.Store.CreateArticle(a)

	req := httptest.NewRequest("POST", "/delete/arch-csrf-1", nil)
	req.SetPathValue("id", "arch-csrf-1")
	rec := httptest.NewRecorder()
	withCSRFSession(h, "test-csrf", h.DeleteArticle).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("delete without csrf: expected 403, got %d", rec.Code)
	}
	got, _ := h.Store.GetArticle("arch-csrf-1")
	if got == nil {
		t.Fatal("article should not be deleted without csrf")
	}
}

func TestIndex_SortAndLimit(t *testing.T) {
	h := setupTestHandler(t)

	now := time.Now()
	h.Store.CreateArticle(testArticle("idx-1", "Alpha", "url", now.Add(-3*time.Hour)))
	h.Store.CreateArticle(testArticle("idx-2", "Beta", "url", now.Add(-2*time.Hour)))
	h.Store.CreateArticle(testArticle("idx-3", "Gamma", "url", now.Add(-1*time.Hour)))

	// Default: newest first (DESC)
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	withSession(h, h.Index).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("index: expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Inbox") {
		t.Fatal("index should render Inbox heading")
	}
	if !strings.Contains(body, "Gamma") || !strings.Contains(body, "Alpha") {
		t.Fatal("index desc: first should be Gamma (newest)")
	}

	// Sort ascending (oldest first)
	req = httptest.NewRequest("GET", "/?sort=asc", nil)
	rec = httptest.NewRecorder()
	withSession(h, h.Index).ServeHTTP(rec, req)
	body = rec.Body.String()
	idxGamma := strings.Index(body, "Gamma")
	idxAlpha := strings.Index(body, "Alpha")
	if idxGamma < idxAlpha {
		t.Fatalf("index asc: Alpha should appear before Gamma, got Alpha@%d Gamma@%d", idxAlpha, idxGamma)
	}

	// Sort controls present
	if !strings.Contains(body, "sort-btn active") {
		t.Fatal("index should have sort controls")
	}
	if !strings.Contains(body, "limit-btn") {
		t.Fatal("index should have limit controls")
	}

	// Limit param
	req = httptest.NewRequest("GET", "/?limit=50", nil)
	rec = httptest.NewRecorder()
	withSession(h, h.Index).ServeHTTP(rec, req)
	body = rec.Body.String()
	if !strings.Contains(body, "limit=50") {
		t.Fatal("limit=50 should be reflected in controls and pagination")
	}

	// Negative page clamps to 1
	req = httptest.NewRequest("GET", "/?page=-5", nil)
	rec = httptest.NewRecorder()
	withSession(h, h.Index).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("index with page=-5: expected 200, got %d", rec.Code)
	}

	// Invalid limit defaults to 20
	req = httptest.NewRequest("GET", "/?limit=abc", nil)
	rec = httptest.NewRecorder()
	withSession(h, h.Index).ServeHTTP(rec, req)
	body = rec.Body.String()
	if strings.Contains(body, "limit=abc") {
		t.Fatal("invalid limit should not appear in page")
	}
}

func TestExportAPI_SortParam(t *testing.T) {
	h := setupTestHandler(t)

	now := time.Now()
	h.Store.CreateArticle(testArticle("es-1", "Sort Export 1", "url", now.Add(-2*time.Hour)))
	h.Store.CreateArticle(testArticle("es-2", "Sort Export 2", "url", now.Add(-1*time.Hour)))

	// ASC sort
	req := httptest.NewRequest("GET", "/api/v1/export?limit=1&offset=0&sort=asc", nil)
	rec := httptest.NewRecorder()
	h.APIExport(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("export asc: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp model.ExportResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].Title != "Sort Export 1" {
		t.Fatalf("asc sort: first should be Sort Export 1 (oldest), got %s", resp.Results[0].Title)
	}
	// Next URL should preserve sort=asc
	if resp.Next != "" && !strings.Contains(resp.Next, "sort=asc") {
		t.Fatalf("next URL should preserve sort=asc, got %s", resp.Next)
	}

	// DESC sort is default
	req = httptest.NewRequest("GET", "/api/v1/export?limit=1&offset=0", nil)
	rec = httptest.NewRecorder()
	h.APIExport(rec, req)
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Results) == 1 && resp.Results[0].Title != "Sort Export 2" {
		t.Fatalf("desc sort (default): first should be Sort Export 2 (newest), got %s", resp.Results[0].Title)
	}
	// Default sort should NOT include sort=asc in next URL
	if resp.Next != "" && strings.Contains(resp.Next, "sort=asc") {
		t.Fatal("next URL without explicit sort should not include sort=asc")
	}
}

func TestExportAPI_SortWithoutCount(t *testing.T) {
	h := setupTestHandler(t)

	now := time.Now()
	for i := 0; i < 3; i++ {
		h.Store.CreateArticle(testArticle(fmt.Sprintf("swc-%d", i), fmt.Sprintf("NoCount %d", i), "url", now.Add(time.Duration(i-2)*time.Hour)))
	}

	// ASC sort without count
	req := httptest.NewRequest("GET", "/api/v1/export?limit=2&content=false&count=false&sort=asc", nil)
	rec := httptest.NewRecorder()
	h.APIExport(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("export asc no-count: expected 200, got %d", rec.Code)
	}

	var resp model.ExportResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}
	// Oldest first: NoCount 0 (idx -2h) should be first
	if resp.Results[0].Title != "NoCount 0" {
		t.Fatalf("asc no-count: first should be NoCount 0, got %s", resp.Results[0].Title)
	}
	if resp.Next != "" && !strings.Contains(resp.Next, "sort=asc") {
		t.Fatalf("no-count next URL should preserve sort=asc, got %s", resp.Next)
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

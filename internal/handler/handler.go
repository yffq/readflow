package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/alexedwards/scs/v2"
	"github.com/readflow/readflow/internal/extract"
	"github.com/readflow/readflow/internal/middleware"
	"github.com/readflow/readflow/internal/model"
	"github.com/readflow/readflow/internal/sanitize"
	"github.com/readflow/readflow/internal/store"
)

type Handler struct {
	Store     *store.Store
	Templates *template.Template
	SM        *scs.SessionManager
	SetupDone bool
}

func New(s *store.Store, tmpl *template.Template, sm *scs.SessionManager) *Handler {
	h := &Handler{Store: s, Templates: tmpl, SM: sm}
	exists, _ := s.UserExists()
	h.SetupDone = exists
	return h
}

func (h *Handler) getSessionString(ctx context.Context, key string) string {
	if h.SM == nil {
		return ""
	}
	defer func() { recover() }()
	return h.SM.GetString(ctx, key)
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, tmplName string, data map[string]any) {
	if data == nil {
		data = make(map[string]any)
	}
	data["UserID"] = h.getSessionString(r.Context(), "user_id")
	data["SetupDone"] = h.SetupDone
	token := h.getSessionString(r.Context(), "csrf_token")
	if token == "" {
		b := make([]byte, 32)
		rand.Read(b)
		token = hex.EncodeToString(b)
		func() {
			defer func() { recover() }()
			h.SM.Put(r.Context(), "csrf_token", token)
			h.SM.Commit(r.Context())
		}()
	}
	data["CSRFToken"] = token
	if err := h.Templates.ExecuteTemplate(w, tmplName, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if h.getSessionString(r.Context(), "user_id") != "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	h.render(w, r, "page_login", map[string]any{"Error": r.URL.Query().Get("error")})
}

func (h *Handler) SetupPage(w http.ResponseWriter, r *http.Request) {
	if h.SetupDone {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	h.render(w, r, "page_setup", nil)
}

func (h *Handler) Setup(w http.ResponseWriter, r *http.Request) {
	if h.SetupDone {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	password := r.FormValue("password")
	if password == "" || len(password) < 4 {
		http.Redirect(w, r, "/setup?error=Password+must+be+at+least+4+characters", http.StatusSeeOther)
		return
	}
	if _, err := h.Store.CreateUser(password); err != nil {
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}
	h.SetupDone = true
	h.SM.Put(r.Context(), "user_id", "default")
	h.SM.Commit(r.Context())
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")
	if password == "" {
		http.Redirect(w, r, "/login?error=Password+required", http.StatusSeeOther)
		return
	}
	valid, err := h.Store.ValidatePassword(password)
	if err != nil || !valid {
		http.Redirect(w, r, "/login?error=Invalid+password", http.StatusSeeOther)
		return
	}
	h.SM.Put(r.Context(), "user_id", "default")
	h.SM.Commit(r.Context())
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.SM.Destroy(r.Context())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "unread"
	}
	page := intValue(r.URL.Query().Get("page"), 1)
	limit := 20
	offset := (page - 1) * limit
	articles, total, _ := h.Store.ListArticles(status, limit, offset)
	h.render(w, r, "page_index", map[string]any{
		"Articles": articles,
		"Total":    total,
		"Page":     page,
		"Limit":    limit,
		"Status":   status,
	})
}

func (h *Handler) ReadPage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	article, err := h.Store.GetArticle(id)
	if err != nil || article == nil {
		http.NotFound(w, r)
		return
	}
	age := time.Since(article.CreatedAt)
	relativeTime := formatRelativeTime(age)
	h.render(w, r, "page_read", map[string]any{
		"Article":   article,
		"RelTime":   relativeTime,
		"WordCount": article.WordCount,
		"ReadTime":  estimateReadTime(article.WordCount),
	})
}

func (h *Handler) SavePage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "page_save", nil)
}

func (h *Handler) SaveForm(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	urlStr := r.FormValue("url")
	htmlStr := r.FormValue("article_html")
	title := r.FormValue("title")

	if htmlStr != "" {
		h.saveHTML(w, r, &model.SaveRequest{HTML: htmlStr, Title: title, URL: urlStr})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if urlStr != "" {
		h.saveURL(w, r, &model.SaveRequest{URL: urlStr})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	h.render(w, r, "page_save", map[string]any{"Error": "Please provide a URL or paste article content."})
}

func (h *Handler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	userID := h.getSessionString(r.Context(), "user_id")
	keys, _ := h.Store.ListAPIKeys(userID)
	newKey := h.getSessionString(r.Context(), "new_api_key")
	h.render(w, r, "page_settings", map[string]any{
		"APIKeys":   keys,
		"NewAPIKey": newKey,
	})
}

func (h *Handler) GenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := h.getSessionString(r.Context(), "user_id")
	name := r.FormValue("name")
	if name == "" {
		name = "API Key"
	}
	_, rawKey, keyHash := middleware.GenerateAPIKey()
	if err := h.Store.CreateAPIKey(userID, rawKey[:11], keyHash, name); err != nil {
		http.Error(w, "failed to create api key", http.StatusInternalServerError)
		return
	}
	h.SM.Put(r.Context(), "new_api_key", rawKey)
	h.SM.Commit(r.Context())
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func (h *Handler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := h.getSessionString(r.Context(), "user_id")
	keyID := r.PathValue("id")
	h.Store.DeleteAPIKey(userID, keyID)
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func (h *Handler) APISave(w http.ResponseWriter, r *http.Request) {
	var req model.SaveRequest
	if err := decodeJSON(r, &req); err != nil {
		h.jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.HTML != "" {
		h.saveHTML(w, r, &req)
		return
	}
	if req.URL != "" {
		h.saveURL(w, r, &req)
		return
	}
	h.jsonError(w, http.StatusBadRequest, "either 'url' or 'html' is required")
}

func (h *Handler) saveURL(w http.ResponseWriter, r *http.Request, req *model.SaveRequest) {
	if id, dup := h.Store.CheckArticleByURL(req.URL); dup {
		h.jsonResponse(w, http.StatusOK, fmt.Sprintf(`{"id":"%s","status":"duplicate"}`, id))
		return
	}
	article := &model.Article{
		ID:     newID(),
		URL:    req.URL,
		Source: "url",
		Status: "unread",
	}
	result, err := extract.Extract(req.URL)
	if err != nil {
		article.Title = req.URL
		article.ExtractionFailed = true
		article.ContentHTML = `<div class="extraction-failed"><h2>Content Not Available</h2><p>Readflow could not extract the content of this page. The article may require login or block automated access.</p><div class="ef-info"><strong>URL:</strong> <code>` + template.HTMLEscapeString(req.URL) + `</code></div><div class="ef-actions"><a href="` + template.HTMLEscapeString(req.URL) + `" target="_blank" rel="noopener" class="btn-primary">Open Original</a></div></div>`
	} else {
		article.Title = result.Title
		article.Author = result.Author
		article.SiteName = result.SiteName
		article.WordCount = result.Length
		cleaned := sanitize.Sanitize(result.Content)
		article.ContentHTML = cleaned
		converter := md.NewConverter("", true, nil)
		markdown, _ := converter.ConvertString(cleaned)
		article.ContentMD = markdown
	}
	article.CreatedAt = time.Now()
	article.UpdatedAt = article.CreatedAt
	if err := h.Store.CreateArticle(article); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to save article: "+err.Error())
		return
	}
	h.jsonResponse(w, http.StatusOK, fmt.Sprintf(`{"id":"%s","title":"%s","status":"created"}`, article.ID, template.JSEscapeString(article.Title)))
}

func (h *Handler) saveHTML(w http.ResponseWriter, r *http.Request, req *model.SaveRequest) {
	cleaned := sanitize.Sanitize(req.HTML)
	title := req.Title
	if title == "" {
		title = "Saved Article"
	}
	converter := md.NewConverter("", true, nil)
	markdown, _ := converter.ConvertString(cleaned)
	wordCount := countWords(cleaned)
	article := &model.Article{
		ID:          newID(),
		Title:       title,
		URL:         req.URL,
		ContentHTML: cleaned,
		ContentMD:   markdown,
		WordCount:   wordCount,
		Source:      "paste",
		Status:      "unread",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := h.Store.CreateArticle(article); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to save article: "+err.Error())
		return
	}
	h.jsonResponse(w, http.StatusOK, fmt.Sprintf(`{"id":"%s","title":"%s","status":"created"}`, article.ID, template.JSEscapeString(article.Title)))
}

func (h *Handler) APIExport(w http.ResponseWriter, r *http.Request) {
	updatedAfterStr := r.URL.Query().Get("updated_after")
	updatedAfter := time.Time{}
	if updatedAfterStr != "" {
		var err error
		updatedAfter, err = time.Parse(time.RFC3339, updatedAfterStr)
		if err != nil {
			updatedAfter, _ = time.Parse("2006-01-02T15:04:05Z", updatedAfterStr)
		}
	}
	limit := intValue(r.URL.Query().Get("limit"), 100)
	offset := intValue(r.URL.Query().Get("offset"), 0)
	articles, total, err := h.Store.ListArticlesSince(updatedAfter, limit, offset)
	if err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to query articles")
		return
	}
	var next string
	if offset+limit < total {
		params := url.Values{}
		if updatedAfterStr != "" {
			params.Set("updated_after", updatedAfterStr)
		}
		params.Set("limit", fmt.Sprint(limit))
		params.Set("offset", fmt.Sprint(offset+limit))
		next = "/api/v1/export?" + params.Encode()
	}
	resp := model.ExportResponse{Count: total, Next: next, Results: articles}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) ArchiveArticle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.Store.UpdateArticleStatus(id, "archived")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) DeleteArticle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.Store.DeleteArticle(id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type deleteBatchRequest struct {
	IDs       []string `json:"ids"`
	CSRFToken string   `json:"csrf_token"`
}

func (h *Handler) DeleteArticles(w http.ResponseWriter, r *http.Request) {
	sessionToken := h.getSessionString(r.Context(), "csrf_token")

	var req deleteBatchRequest
	if err := decodeJSON(r, &req); err != nil {
		h.jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if sessionToken == "" || req.CSRFToken != sessionToken {
		h.jsonError(w, http.StatusForbidden, "invalid csrf token")
		return
	}

	if len(req.IDs) == 0 {
		h.jsonError(w, http.StatusBadRequest, "no article ids provided")
		return
	}

	if err := h.Store.DeleteArticles(req.IDs); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to delete articles")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"deleted": len(req.IDs)})
}

func (h *Handler) APIDeleteArticles(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := decodeJSON(r, &req); err != nil {
		h.jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.IDs) == 0 {
		h.jsonError(w, http.StatusBadRequest, "no article ids provided")
		return
	}
	if err := h.Store.DeleteArticles(req.IDs); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to delete articles")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": len(req.IDs)})
}

func (h *Handler) UnreadArticle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.Store.UpdateArticleStatus(id, "unread")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) ReadIframePage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var urlStr string
	if id != "" {
		article, _ := h.Store.GetArticle(id)
		if article != nil {
			urlStr = article.URL
		}
	}
	if urlStr == "" {
		urlStr = r.URL.Query().Get("url")
	}
	if urlStr == "" {
		http.NotFound(w, r)
		return
	}
	h.render(w, r, "page_iframe", map[string]any{"URL": urlStr})
}

func (h *Handler) jsonError(w http.ResponseWriter, status int, msg string) {
	h.jsonResponse(w, status, fmt.Sprintf(`{"error":"%s"}`, template.JSEscapeString(msg)))
}

func (h *Handler) jsonResponse(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(body))
}

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

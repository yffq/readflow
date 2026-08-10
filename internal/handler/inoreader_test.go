package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInoreaderWebhook(t *testing.T) {
	h := setupTestHandler(t)
	payload := map[string]any{
		"rule": map[string]string{"name": "Readflow"},
		"items": []map[string]any{{
			"id":        "tag:google.com,2005:reader/item/1",
			"title":     "Webhook Article",
			"author":    "Test Author",
			"canonical": []map[string]string{{"href": "https://example.com/inoreader-article"}},
			"summary":   map[string]string{"content": "<p>Hello <strong>Readflow</strong></p><script>alert(1)</script>"},
			"origin":    map[string]string{"title": "Example Feed"},
		}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal webhook: %v", err)
	}

	response := callInoreaderWebhook(t, h, body, http.StatusOK)
	if response["created"] != float64(1) || response["duplicates"] != float64(0) || response["failed"] != float64(0) {
		t.Fatalf("unexpected webhook counts: %+v", response)
	}
	results := response["results"].([]any)
	id := results[0].(map[string]any)["id"].(string)

	article, err := h.Store.GetArticle(id)
	if err != nil {
		t.Fatalf("get saved article: %v", err)
	}
	if article == nil {
		t.Fatal("expected saved article")
	}
	if article.Source != "inoreader" || article.Author != "Test Author" || article.SiteName != "Example Feed" {
		t.Fatalf("unexpected article metadata: %+v", article)
	}
	if strings.Contains(article.ContentHTML, "<script") || !strings.Contains(article.ContentMD, "**Readflow**") {
		t.Fatalf("article content was not sanitized and converted: %#v", article)
	}

	response = callInoreaderWebhook(t, h, body, http.StatusOK)
	if response["duplicates"] != float64(1) || response["created"] != float64(0) {
		t.Fatalf("expected duplicate response, got %+v", response)
	}
}

func TestInoreaderWebhookBoundaries(t *testing.T) {
	h := setupTestHandler(t)
	callInoreaderWebhook(t, h, []byte("not json"), http.StatusBadRequest)
	callInoreaderWebhook(t, h, []byte("{\"items\":[]}"), http.StatusBadRequest)

	payload := map[string]any{
		"items": []map[string]any{
			{
				"id":        "empty",
				"title":     "Empty",
				"canonical": []map[string]string{{"href": "https://example.com/empty"}},
				"summary":   map[string]string{"content": ""},
			},
			{
				"id":        "bad-url",
				"title":     "Bad URL",
				"canonical": []map[string]string{{"href": "http://127.0.0.1/private"}},
				"summary":   map[string]string{"content": "<p>private</p>"},
			},
			{
				"id":        "alternate",
				"title":     "",
				"alternate": []map[string]string{{"href": "https://example.com/alternate"}},
				"summary":   map[string]string{"content": "<p>Valid</p>"},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal boundary payload: %v", err)
	}
	response := callInoreaderWebhook(t, h, body, http.StatusOK)
	if response["created"] != float64(1) || response["failed"] != float64(2) {
		t.Fatalf("unexpected partial counts: %+v", response)
	}
}

func callInoreaderWebhook(t *testing.T, h *Handler, body []byte, wantStatus int) map[string]any {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/v1/webhooks/inoreader", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.InoreaderWebhook(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("expected %d, got %d: %s", wantStatus, rec.Code, rec.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode webhook response: %v", err)
	}
	return response
}

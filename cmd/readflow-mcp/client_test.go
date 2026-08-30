package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListArticles(t *testing.T) {
	var gotPath, gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"count": 1,
			"has_more": false,
			"results": [
				{"id": "abc", "title": "Example", "saved_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"}
			]
		}`))
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "rf_test")
	data, err := c.ListArticles(context.Background(), 20, 0, "")
	if err != nil {
		t.Fatalf("ListArticles: %v", err)
	}

	if gotAuth != "Bearer rf_test" {
		t.Fatalf("expected Bearer auth, got %q", gotAuth)
	}
	if !strings.Contains(gotPath, "content=false") || !strings.Contains(gotPath, "count=false") {
		t.Fatalf("expected content=false&count=false in query, got %q", gotPath)
	}

	results := data["results"].([]any)
	item := results[0].(map[string]any)
	if item["created_at"] != "2024-01-01T00:00:00Z" {
		t.Fatalf("expected created_at normalized from saved_at, got %v", item["created_at"])
	}
	if _, ok := item["saved_at"]; ok {
		t.Fatalf("saved_at should be renamed away, still present")
	}
}

func TestListArticlesPassesParams(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"count":0,"results":[]}`))
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "rf_test")
	if _, err := c.ListArticles(context.Background(), 50, 10, "2024-06-01T00:00:00Z"); err != nil {
		t.Fatalf("ListArticles: %v", err)
	}

	for _, want := range []string{"limit=50", "offset=10", "updated_after=2024-06-01T00%3A00%3A00Z"} {
		if !strings.Contains(gotPath, want) {
			t.Fatalf("expected query to contain %q, got %q", want, gotPath)
		}
	}
}

func TestGetArticle(t *testing.T) {
	var gotPath, gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"abc","title":"Example","content_html":"<p>hi</p>","content_markdown":"hi","saved_at":"2024-01-01T00:00:00Z"}`))
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "rf_test")
	data, err := c.GetArticle(context.Background(), "abc")
	if err != nil {
		t.Fatalf("GetArticle: %v", err)
	}

	if gotPath != "/api/v1/article/abc" {
		t.Fatalf("expected path /api/v1/article/abc, got %q", gotPath)
	}
	if gotAuth != "Bearer rf_test" {
		t.Fatalf("expected Bearer auth, got %q", gotAuth)
	}
	if data["created_at"] != "2024-01-01T00:00:00Z" {
		t.Fatalf("expected created_at normalized, got %v", data["created_at"])
	}
	if _, ok := data["saved_at"]; ok {
		t.Fatalf("saved_at should be renamed away, still present")
	}
	if data["content_markdown"] != "hi" {
		t.Fatalf("expected content_markdown preserved, got %v", data["content_markdown"])
	}
}

func TestGetArticleRequiresID(t *testing.T) {
	c := NewClient("http://example.com", "rf_test")
	if _, err := c.GetArticle(context.Background(), "  "); err == nil {
		t.Fatal("expected error for blank id")
	}
}

func TestGetArticleHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"article not found"}`))
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "rf_test")
	if _, err := c.GetArticle(context.Background(), "missing"); err == nil {
		t.Fatal("expected error for 404")
	} else if !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected error to mention status 404, got %v", err)
	}
}

func TestGetArticleInvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "rf_test")
	if _, err := c.GetArticle(context.Background(), "abc"); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestNewClientTrimsTrailingSlash(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"id":"x"}`))
	}))
	defer ts.Close()

	c := NewClient(ts.URL+"/", "rf_test")
	if _, err := c.GetArticle(context.Background(), "x"); err != nil {
		t.Fatalf("GetArticle: %v", err)
	}
	if gotPath != "/api/v1/article/x" {
		t.Fatalf("expected single-slash path, got %q", gotPath)
	}
}

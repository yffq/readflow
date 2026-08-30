package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestListArticlesTool(t *testing.T) {
	var gotAuth, gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"count": 1,
			"has_more": false,
			"results": [
				{"id": "abc", "title": "Example", "saved_at": "2024-01-01T00:00:00Z"}
			]
		}`))
	}))
	defer ts.Close()

	s := newMCPServer(NewClient(ts.URL, "rf_test"))
	tool := s.GetTool("list_articles")
	if tool == nil {
		t.Fatal("list_articles tool not registered")
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = "list_articles"
	req.Params.Arguments = map[string]any{"limit": float64(10), "offset": float64(0)}

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}

	if gotAuth != "Bearer rf_test" {
		t.Fatalf("expected Bearer auth, got %q", gotAuth)
	}
	if !strings.Contains(gotPath, "content=false") || !strings.Contains(gotPath, "count=false") {
		t.Fatalf("expected content=false&count=false in query, got %q", gotPath)
	}

	data, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected map structured content, got %T", result.StructuredContent)
	}
	results := data["results"].([]any)
	item := results[0].(map[string]any)
	if item["created_at"] != "2024-01-01T00:00:00Z" {
		t.Fatalf("expected created_at normalized, got %v", item["created_at"])
	}
}

func TestGetArticleTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/article/abc" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"abc","title":"Example","content_markdown":"hi","saved_at":"2024-01-01T00:00:00Z"}`))
	}))
	defer ts.Close()

	s := newMCPServer(NewClient(ts.URL, "rf_test"))
	tool := s.GetTool("get_article")
	if tool == nil {
		t.Fatal("get_article tool not registered")
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = "get_article"
	req.Params.Arguments = map[string]any{"id": "abc"}

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}

	data := result.StructuredContent.(map[string]any)
	if data["content_markdown"] != "hi" {
		t.Fatalf("expected content_markdown preserved, got %v", data["content_markdown"])
	}
	if data["created_at"] != "2024-01-01T00:00:00Z" {
		t.Fatalf("expected created_at normalized, got %v", data["created_at"])
	}
}

func TestGetArticleToolMissingID(t *testing.T) {
	s := newMCPServer(NewClient("http://example.com", "rf_test"))
	tool := s.GetTool("get_article")

	req := mcp.CallToolRequest{}
	req.Params.Name = "get_article"
	req.Params.Arguments = map[string]any{}

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for missing id")
	}
}

func TestListArticlesToolDefaultsOnBadParams(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"count":0,"results":[]}`))
	}))
	defer ts.Close()

	s := newMCPServer(NewClient(ts.URL, "rf_test"))
	tool := s.GetTool("list_articles")

	req := mcp.CallToolRequest{}
	req.Params.Name = "list_articles"
	req.Params.Arguments = map[string]any{"limit": float64(-5), "offset": float64(-1)}

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}
	if !strings.Contains(gotPath, "limit=20") {
		t.Fatalf("expected limit defaulted to 20, got %q", gotPath)
	}
	if !strings.Contains(gotPath, "offset=0") {
		t.Fatalf("expected offset defaulted to 0, got %q", gotPath)
	}
}

func TestGetArticleToolHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"article not found"}`))
	}))
	defer ts.Close()

	s := newMCPServer(NewClient(ts.URL, "rf_test"))
	tool := s.GetTool("get_article")

	req := mcp.CallToolRequest{}
	req.Params.Name = "get_article"
	req.Params.Arguments = map[string]any{"id": "missing"}

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for HTTP 404")
	}
}

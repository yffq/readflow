package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/readflow/readflow/internal/model"
	"github.com/readflow/readflow/internal/store"
)

const testSchema = `
CREATE TABLE IF NOT EXISTS articles (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    url TEXT DEFAULT '',
    content_html TEXT NOT NULL DEFAULT '',
    content_md TEXT NOT NULL DEFAULT '',
    author TEXT NOT NULL DEFAULT '',
    site_name TEXT NOT NULL DEFAULT '',
    word_count INTEGER NOT NULL DEFAULT 0,
    source TEXT NOT NULL DEFAULT 'url',
    extraction_failed INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'unread',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);`

func newTestClient(t *testing.T, save SaveFunc) (*store.Store, *client.Client) {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Migrate(testSchema); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	c, err := client.NewInProcessClient(NewServer(s, save))
	if err != nil {
		t.Fatalf("new inprocess client: %v", err)
	}
	t.Cleanup(func() { c.Close() })

	if _, err := c.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "test", Version: "1.0"},
			Capabilities:    mcp.ClientCapabilities{},
		},
	}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	return s, c
}

func seedArticle(t *testing.T, s *store.Store, id, title string) {
	t.Helper()
	now := time.Now()
	a := &model.Article{
		ID:          id,
		Title:       title,
		URL:         "https://example.com/" + id,
		ContentHTML: "<p>hi</p>",
		ContentMD:   "hi",
		WordCount:   2,
		Source:      "url",
		Status:      "unread",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.CreateArticle(a); err != nil {
		t.Fatalf("seed article: %v", err)
	}
}

func TestListArticles(t *testing.T) {
	s, c := newTestClient(t, nil)
	seedArticle(t, s, "art-1", "One")
	seedArticle(t, s, "art-2", "Two")

	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "list_articles",
			Arguments: map[string]any{"limit": float64(10)},
		},
	})
	if err != nil {
		t.Fatalf("call list_articles: %v", err)
	}
	if result.IsError {
		t.Fatalf("list_articles error: %v", result.Content)
	}

	data := result.StructuredContent.(map[string]any)
	results := data["results"].([]any)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	first := results[0].(map[string]any)
	if first["id"] != "art-2" {
		t.Fatalf("expected newest first (art-2), got %v", first["id"])
	}
	if _, ok := first["created_at"]; !ok {
		t.Fatalf("expected created_at field, got %v", first)
	}
	if _, ok := first["content_html"]; ok {
		t.Fatalf("summary should omit content_html")
	}
}

func TestListArticlesInvalidUpdatedAfter(t *testing.T) {
	_, c := newTestClient(t, nil)

	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "list_articles",
			Arguments: map[string]any{"updated_after": "yesterday"},
		},
	})
	if err != nil {
		t.Fatalf("call list_articles: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for invalid updated_after")
	}
}

func TestGetArticle(t *testing.T) {
	s, c := newTestClient(t, nil)
	seedArticle(t, s, "art-1", "One")

	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_article",
			Arguments: map[string]any{"id": "art-1"},
		},
	})
	if err != nil {
		t.Fatalf("call get_article: %v", err)
	}
	if result.IsError {
		t.Fatalf("get_article error: %v", result.Content)
	}

	data := result.StructuredContent.(map[string]any)
	if data["content_markdown"] != "hi" {
		t.Fatalf("expected content_markdown hi, got %v", data["content_markdown"])
	}
	if _, ok := data["created_at"]; !ok {
		t.Fatalf("expected created_at field, got %v", data)
	}
}

func TestGetArticleNotFound(t *testing.T) {
	_, c := newTestClient(t, nil)

	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_article",
			Arguments: map[string]any{"id": "missing"},
		},
	})
	if err != nil {
		t.Fatalf("call get_article: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing article")
	}
}

func TestSaveArticle(t *testing.T) {
	var got *model.SaveRequest
	save := func(req *model.SaveRequest) (*model.Article, string, error) {
		got = req
		return &model.Article{ID: "new-1", Title: "Saved", Status: "unread"}, "created", nil
	}
	_, c := newTestClient(t, save)

	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "save_article",
			Arguments: map[string]any{"url": "https://example.com/x", "title": "Saved"},
		},
	})
	if err != nil {
		t.Fatalf("call save_article: %v", err)
	}
	if result.IsError {
		t.Fatalf("save_article error: %v", result.Content)
	}
	if got == nil || got.URL != "https://example.com/x" {
		t.Fatalf("expected save func called with url, got %+v", got)
	}

	data := result.StructuredContent.(map[string]any)
	if data["status"] != "created" || data["id"] != "new-1" {
		t.Fatalf("expected created result, got %v", data)
	}
}

func TestDeleteArticle(t *testing.T) {
	s, c := newTestClient(t, nil)
	seedArticle(t, s, "art-1", "One")

	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "delete_article",
			Arguments: map[string]any{"ids": []any{"art-1"}},
		},
	})
	if err != nil {
		t.Fatalf("call delete_article: %v", err)
	}
	if result.IsError {
		t.Fatalf("delete_article error: %v", result.Content)
	}

	data := result.StructuredContent.(map[string]any)
	if data["deleted"] != float64(1) {
		t.Fatalf("expected deleted 1, got %v", data["deleted"])
	}

	got, _ := s.GetArticle("art-1")
	if got != nil {
		t.Fatalf("expected article to be deleted, still present")
	}
}

func TestDeleteArticleRequiresIDs(t *testing.T) {
	_, c := newTestClient(t, nil)

	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "delete_article",
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("call delete_article: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing ids")
	}
}

func TestDeleteArticleNonExistent(t *testing.T) {
	s, c := newTestClient(t, nil)
	seedArticle(t, s, "art-1", "One")

	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "delete_article",
			Arguments: map[string]any{"ids": []any{"missing"}},
		},
	})
	if err != nil {
		t.Fatalf("call delete_article: %v", err)
	}
	if result.IsError {
		t.Fatalf("delete_article error: %v", result.Content)
	}

	data := result.StructuredContent.(map[string]any)
	if data["deleted"] != float64(0) {
		t.Fatalf("expected deleted 0 for missing id, got %v", data["deleted"])
	}

	got, _ := s.GetArticle("art-1")
	if got == nil {
		t.Fatalf("expected art-1 to remain after deleting missing id")
	}
}

func TestListArticlesHasMore(t *testing.T) {
	s, c := newTestClient(t, nil)
	seedArticle(t, s, "art-1", "One")
	seedArticle(t, s, "art-2", "Two")

	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "list_articles",
			Arguments: map[string]any{"limit": float64(1), "offset": float64(0)},
		},
	})
	if err != nil {
		t.Fatalf("call list_articles: %v", err)
	}
	if result.IsError {
		t.Fatalf("list_articles error: %v", result.Content)
	}

	data := result.StructuredContent.(map[string]any)
	if data["has_more"] != true {
		t.Fatalf("expected has_more true with limit 1, got %v", data["has_more"])
	}
	results := data["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestGetArticleEmptyID(t *testing.T) {
	_, c := newTestClient(t, nil)

	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_article",
			Arguments: map[string]any{"id": ""},
		},
	})
	if err != nil {
		t.Fatalf("call get_article: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for empty id")
	}
}

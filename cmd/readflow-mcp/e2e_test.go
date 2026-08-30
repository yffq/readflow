package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestMCPEndToEnd drives the full stack: MCP client -> JSON-RPC -> MCP server
// -> HTTP -> a Readflow API stand-in, exercising initialize, tools/list and
// tools/call with parameter schema validation.
func TestMCPEndToEnd(t *testing.T) {
	ts := newReadflowStub(t)
	defer ts.Close()

	c := newInProcessMCPClient(t, NewClient(ts.URL, "rf_test"))
	defer c.Close()
	ctx := context.Background()

	initResult, err := c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "e2e-test", Version: "1.0"},
			Capabilities:    mcp.ClientCapabilities{},
		},
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if initResult.ServerInfo.Name != serverName {
		t.Fatalf("expected server name %q, got %q", serverName, initResult.ServerInfo.Name)
	}

	tools, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools.Tools))
	}

	t.Run("list_articles", func(t *testing.T) {
		result, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "list_articles",
				Arguments: map[string]any{"limit": float64(10)},
			},
		})
		if err != nil {
			t.Fatalf("call list_articles: %v", err)
		}
		if result.IsError {
			t.Fatalf("list_articles returned error: %v", result.Content)
		}
		data := result.StructuredContent.(map[string]any)
		results := data["results"].([]any)
		item := results[0].(map[string]any)
		if item["created_at"] != "2024-01-01T00:00:00Z" {
			t.Fatalf("expected created_at normalized, got %v", item["created_at"])
		}
		if _, ok := item["saved_at"]; ok {
			t.Fatalf("saved_at should be renamed away, still present")
		}
	})

	t.Run("get_article", func(t *testing.T) {
		result, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "get_article",
				Arguments: map[string]any{"id": "abc"},
			},
		})
		if err != nil {
			t.Fatalf("call get_article: %v", err)
		}
		if result.IsError {
			t.Fatalf("get_article returned error: %v", result.Content)
		}
		data := result.StructuredContent.(map[string]any)
		if data["content_markdown"] != "hi" {
			t.Fatalf("expected content_markdown, got %v", data["content_markdown"])
		}
		if data["created_at"] != "2024-01-01T00:00:00Z" {
			t.Fatalf("expected created_at normalized, got %v", data["created_at"])
		}
	})
}

func TestMCPEndToEndInvalidUpdatedAfter(t *testing.T) {
	ts := newReadflowStub(t)
	defer ts.Close()

	c := newInProcessMCPClient(t, NewClient(ts.URL, "rf_test"))
	defer c.Close()
	mustInitialize(t, c)

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
		t.Fatal("expected error result for invalid updated_after")
	}
}

func TestMCPEndToEndMissingID(t *testing.T) {
	ts := newReadflowStub(t)
	defer ts.Close()

	c := newInProcessMCPClient(t, NewClient(ts.URL, "rf_test"))
	defer c.Close()
	mustInitialize(t, c)

	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_article",
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("call get_article: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for missing id")
	}
}

func newReadflowStub(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/export":
			w.Write([]byte(`{"count":1,"has_more":false,"results":[{"id":"abc","title":"Example","saved_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}]}`))
		case r.URL.Path == "/api/v1/article/abc":
			w.Write([]byte(`{"id":"abc","title":"Example","content_html":"<p>hi</p>","content_markdown":"hi","saved_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
		}
	}))
}

func newInProcessMCPClient(t *testing.T, readflow *Client) *client.Client {
	t.Helper()
	c, err := client.NewInProcessClient(newMCPServer(readflow))
	if err != nil {
		t.Fatalf("new inprocess client: %v", err)
	}
	return c
}

func mustInitialize(t *testing.T, c *client.Client) {
	t.Helper()
	if _, err := c.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "e2e-test", Version: "1.0"},
			Capabilities:    mcp.ClientCapabilities{},
		},
	}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
}

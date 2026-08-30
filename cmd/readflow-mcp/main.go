package main

import (
	"context"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "readflow"
	serverVersion = "0.1.0"
)

func main() {
	baseURL := os.Getenv("READFLOW_BASE_URL")
	apiKey := os.Getenv("READFLOW_API_KEY")
	if baseURL == "" || apiKey == "" {
		log.Fatal("READFLOW_BASE_URL and READFLOW_API_KEY must be set")
	}

	client := NewClient(baseURL, apiKey)
	mcpServer := newMCPServer(client)

	stdio := server.NewStdioServer(mcpServer)
	if err := stdio.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
		log.Fatalf("mcp server: %v", err)
	}
}

// newMCPServer builds the MCP server and registers the Readflow tools.
func newMCPServer(client *Client) *server.MCPServer {
	s := server.NewMCPServer(serverName, serverVersion)

	s.AddTool(
		mcp.NewTool("list_articles",
			mcp.WithDescription("List saved Readflow articles (summaries only, no full content). Returns a paginated result with count, has_more and results."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithNumber("limit",
				mcp.DefaultNumber(20.0),
				mcp.Description("Maximum items per page. Defaults to 20."),
			),
			mcp.WithNumber("offset",
				mcp.DefaultNumber(0.0),
				mcp.Description("Number of items to skip. Defaults to 0."),
			),
			mcp.WithString("updated_after",
				mcp.Description("Optional RFC3339 timestamp; only return articles updated after this time."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			limit := req.GetInt("limit", 20)
			if limit <= 0 {
				limit = 20
			}
			offset := req.GetInt("offset", 0)
			if offset < 0 {
				offset = 0
			}
			updatedAfter := req.GetString("updated_after", "")

			data, err := client.ListArticles(ctx, limit, offset, updatedAfter)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultJSON(data)
		},
	)

	s.AddTool(
		mcp.NewTool("get_article",
			mcp.WithDescription("Get a single saved Readflow article by id, including full content (content_html and content_markdown)."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Article id."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			data, err := client.GetArticle(ctx, id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultJSON(data)
		},
	)

	return s
}

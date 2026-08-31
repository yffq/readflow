// Package mcp exposes Readflow as a Model Context Protocol (MCP) server so AI
// assistants can read and manage saved articles. It is mounted into the main
// Readflow HTTP server under /mcp with API key authentication.
package mcp

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/readflow/readflow/internal/model"
	"github.com/readflow/readflow/internal/store"
)

const (
	serverName    = "readflow"
	serverVersion = "0.2.0"
)

const maxPageSize = 100

// SaveFunc saves a URL or HTML article and returns the created article and its
// status ("created" or "duplicate").
type SaveFunc func(req *model.SaveRequest) (*model.Article, string, error)

// NewServer builds the MCP server with the Readflow tools.
func NewServer(s *store.Store, save SaveFunc) *server.MCPServer {
	mcpServer := server.NewMCPServer(serverName, serverVersion)

	mcpServer.AddTool(
		mcp.NewTool("list_articles",
			mcp.WithDescription("List saved Readflow articles (summaries only, no full content). Paginated with has_more and results; pass offset to page forward."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithNumber("limit",
				mcp.DefaultNumber(20.0),
				mcp.Description("Maximum items per page (1-100). Defaults to 20."),
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
			if limit > maxPageSize {
				limit = maxPageSize
			}
			offset := req.GetInt("offset", 0)
			if offset < 0 {
				offset = 0
			}

			var updatedAfter time.Time
			if raw := req.GetString("updated_after", ""); raw != "" {
				parsed, err := time.Parse(time.RFC3339, raw)
				if err != nil {
					return mcp.NewToolResultError("invalid updated_after: must be RFC3339, e.g. 2024-01-01T00:00:00Z"), nil
				}
				updatedAfter = parsed.UTC()
			}

			// Fetch limit+1 to detect whether another page exists.
			exports, err := s.ListArticleSummaryPageSince(updatedAfter, limit+1, offset, false)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			hasMore := len(exports) > limit
			if hasMore {
				exports = exports[:limit]
			}

			results := make([]articleOutput, 0, len(exports))
			for _, a := range exports {
				results = append(results, articleOutputFromExport(a))
			}
			return mcp.NewToolResultJSON(listResult{HasMore: hasMore, Results: results})
		},
	)

	mcpServer.AddTool(
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
			article, err := s.GetArticle(id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if article == nil {
				return mcp.NewToolResultError("article not found"), nil
			}
			return mcp.NewToolResultJSON(articleOutputFromArticle(article))
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("save_article",
			mcp.WithDescription("Save an article to Readflow from a URL, or paste raw HTML content. Returns the saved article id and status (created or duplicate)."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithString("url",
				mcp.Description("Article URL to fetch and save. Required unless html is provided."),
			),
			mcp.WithString("html",
				mcp.Description("Raw article HTML to save. Alternative to url."),
			),
			mcp.WithString("title",
				mcp.Description("Optional title. Used when saving html."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			article, status, err := save(&model.SaveRequest{
				URL:   req.GetString("url", ""),
				HTML:  req.GetString("html", ""),
				Title: req.GetString("title", ""),
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultJSON(saveResult{
				ID:     article.ID,
				Title:  article.Title,
				Status: status,
			})
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("delete_article",
			mcp.WithDescription("Delete saved Readflow articles by id. Returns how many were deleted."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithArray("ids",
				mcp.Required(),
				mcp.Description("Article ids to delete."),
				mcp.WithStringItems(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ids := req.GetStringSlice("ids", nil)
			if len(ids) == 0 {
				return mcp.NewToolResultError("ids is required"), nil
			}
			deleted, err := s.DeleteArticles(ids)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultJSON(deleteResult{Deleted: deleted})
		},
	)

	return mcpServer
}

// articleOutput is the normalized JSON shape returned to MCP clients. It uses
// created_at consistently (the HTTP API's export list uses saved_at instead).
type articleOutput struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	URL              string    `json:"url,omitempty"`
	ContentHTML      string    `json:"content_html,omitempty"`
	ContentMarkdown  string    `json:"content_markdown,omitempty"`
	Author           string    `json:"author,omitempty"`
	SiteName         string    `json:"site_name,omitempty"`
	WordCount        int       `json:"word_count"`
	Source           string    `json:"source"`
	ExtractionFailed bool      `json:"extraction_failed"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func articleOutputFromExport(a model.ArticleExport) articleOutput {
	return articleOutput{
		ID:               a.ID,
		Title:            a.Title,
		URL:              a.URL,
		ContentHTML:      a.ContentHTML,
		ContentMarkdown:  a.ContentMarkdown,
		Author:           a.Author,
		SiteName:         a.SiteName,
		WordCount:        a.WordCount,
		Source:           a.Source,
		ExtractionFailed: a.ExtractionFailed,
		Status:           a.Status,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
	}
}

func articleOutputFromArticle(a *model.Article) articleOutput {
	return articleOutput{
		ID:               a.ID,
		Title:            a.Title,
		URL:              a.URL,
		ContentHTML:      a.ContentHTML,
		ContentMarkdown:  a.ContentMD,
		Author:           a.Author,
		SiteName:         a.SiteName,
		WordCount:        a.WordCount,
		Source:           a.Source,
		ExtractionFailed: a.ExtractionFailed,
		Status:           a.Status,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
	}
}

type listResult struct {
	HasMore bool            `json:"has_more"`
	Results []articleOutput `json:"results"`
}

type saveResult struct {
	ID     string `json:"id"`
	Title  string `json:"title,omitempty"`
	Status string `json:"status"`
}

type deleteResult struct {
	Deleted int `json:"deleted"`
}

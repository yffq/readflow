package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// maxResponseBytes caps how much of a Readflow response we read into memory.
const maxResponseBytes = 32 << 20 // 32 MiB

// Client is a minimal HTTP client for the Readflow JSON API.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewClient builds a Client. baseURL may have a trailing slash; it is trimmed.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:  strings.TrimSpace(apiKey),
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// ListArticles returns article summaries (no full content) from /api/v1/export.
func (c *Client) ListArticles(ctx context.Context, limit, offset int, updatedAfter string) (map[string]any, error) {
	q := url.Values{}
	q.Set("content", "false")
	q.Set("count", "false")
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	if updatedAfter != "" {
		q.Set("updated_after", updatedAfter)
	}
	return c.get(ctx, "/api/v1/export?"+q.Encode())
}

// GetArticle returns a single article including full content from /api/v1/article/{id}.
func (c *Client) GetArticle(ctx context.Context, id string) (map[string]any, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("article id is required")
	}
	return c.get(ctx, "/api/v1/article/"+url.PathEscape(id))
}

func (c *Client) get(ctx context.Context, path string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request readflow: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("readflow returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	normalize(out)
	return out, nil
}

// normalize renames Readflow's "saved_at" field to "created_at" so the two
// response shapes (export list vs article detail) expose a consistent name.
// It rewrites the top-level object and each element of a "results" array.
func normalize(m map[string]any) {
	renameKey(m, "saved_at", "created_at")
	if results, ok := m["results"].([]any); ok {
		for _, r := range results {
			if item, ok := r.(map[string]any); ok {
				renameKey(item, "saved_at", "created_at")
			}
		}
	}
}

func renameKey(m map[string]any, from, to string) {
	if v, ok := m[from]; ok {
		m[to] = v
		delete(m, from)
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

package extract

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
)

var client = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

type ExtractResult struct {
	Title    string
	Content  string
	Text     string
	Author   string
	SiteName string
	Length   int
}

func Extract(url string) (*ExtractResult, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Readflow/1.0 (+https://github.com/readflow/readflow)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") && !strings.Contains(contentType, "application/xhtml") {
		return nil, fmt.Errorf("not an HTML page, content-type: %s", contentType)
	}

	article, err := readability.FromReader(resp.Body, resp.Request.URL)
	if err != nil {
		return nil, fmt.Errorf("extract content: %w", err)
	}

	title := article.Title
	if title == "" {
		title = "Untitled"
	}

	content := article.Content
	if content == "" {
		content = fmt.Sprintf("<p>%s</p>", article.TextContent)
	}

	return &ExtractResult{
		Title:    title,
		Content:  content,
		Text:     article.TextContent,
		Author:   article.Byline,
		SiteName: article.SiteName,
		Length:   len(article.TextContent),
	}, nil
}

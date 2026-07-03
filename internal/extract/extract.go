package extract

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
)

var (
	client = &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	titleRe   = regexp.MustCompile(`(?is)<title[^>]*>(.+?)</title>`)
	imgAttrRe = regexp.MustCompile(`(?i)\s+(width|height)\s*=\s*"[^"]*"`)
)

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
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

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

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	article, err := readability.FromReader(bytes.NewReader(body), resp.Request.URL)
	if err != nil {
		return nil, fmt.Errorf("extract content: %w", err)
	}

	title := article.Title
	if title == "" {
		title = "Untitled"
	}

	// Fix: when og:title is the site name (short) instead of article title,
	// prefer the <title> tag which usually contains the real article title.
	htmlTitle := extractTitleTag(string(body))
	if htmlTitle != "" && len([]rune(title)) < len([]rune(htmlTitle))/2 {
		title = htmlTitle
	}

	content := article.Content
	if content == "" {
		content = fmt.Sprintf("<p>%s</p>", article.TextContent)
	}
	content = stripImgAttrs(content)

	return &ExtractResult{
		Title:    title,
		Content:  content,
		Text:     article.TextContent,
		Author:   cleanAuthor(article.Byline),
		SiteName: article.SiteName,
		Length:   len(article.TextContent),
	}, nil
}

func cleanAuthor(author string) string {
	author = strings.TrimSpace(author)
	if after, ok := strings.CutPrefix(author, "By "); ok {
		return strings.TrimSpace(after)
	}
	if after, ok := strings.CutPrefix(author, "by "); ok {
		return strings.TrimSpace(after)
	}
	return author
}

func stripImgAttrs(html string) string {
	return imgAttrRe.ReplaceAllString(html, "")
}

func extractTitleTag(html string) string {
	m := titleRe.FindStringSubmatch(html)
	if len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

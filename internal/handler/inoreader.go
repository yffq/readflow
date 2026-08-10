package handler

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/readflow/readflow/internal/extract"
	"github.com/readflow/readflow/internal/model"
	"github.com/readflow/readflow/internal/sanitize"
	"golang.org/x/net/html"
)

type inoreaderWebhook struct {
	Items []inoreaderItem `json:"items"`
}

type inoreaderItem struct {
	ID        string           `json:"id"`
	Title     string           `json:"title"`
	Author    string           `json:"author"`
	Canonical []inoreaderLink  `json:"canonical"`
	Alternate []inoreaderLink  `json:"alternate"`
	Summary   inoreaderSummary `json:"summary"`
	Origin    inoreaderOrigin  `json:"origin"`
}

type inoreaderLink struct {
	Href string `json:"href"`
}

type inoreaderSummary struct {
	Content string `json:"content"`
}

type inoreaderOrigin struct {
	Title string `json:"title"`
}

type inoreaderResult struct {
	InoreaderID string `json:"inoreader_id,omitempty"`
	ID          string `json:"id,omitempty"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

func (h *Handler) InoreaderWebhook(w http.ResponseWriter, r *http.Request) {
	var payload inoreaderWebhook
	if err := decodeJSON(r, &payload); err != nil {
		h.jsonError(w, http.StatusBadRequest, "invalid Inoreader webhook body")
		return
	}
	if len(payload.Items) == 0 {
		h.jsonError(w, http.StatusBadRequest, "Inoreader webhook contains no items")
		return
	}

	results := make([]inoreaderResult, 0, len(payload.Items))
	created := 0
	duplicates := 0
	for _, item := range payload.Items {
		result := h.saveInoreaderItem(item)
		results = append(results, result)
		switch result.Status {
		case "created":
			created++
		case "duplicate":
			duplicates++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"created":    created,
		"duplicates": duplicates,
		"failed":     len(results) - created - duplicates,
		"results":    results,
	})
}

func (h *Handler) saveInoreaderItem(item inoreaderItem) inoreaderResult {
	result := inoreaderResult{InoreaderID: item.ID, Status: "error"}
	urlStr := firstInoreaderURL(item)
	if urlStr != "" {
		if err := extract.ValidatePublicURL(urlStr); err != nil {
			result.Error = fmt.Sprintf("invalid article URL: %v", err)
			return result
		}
		if id, duplicate := h.Store.CheckArticleByURL(urlStr); duplicate {
			result.ID = id
			result.Status = "duplicate"
			return result
		}
	}

	content := strings.TrimSpace(item.Summary.Content)
	if content == "" {
		result.Error = "article content is empty"
		return result
	}
	cleaned := sanitize.Sanitize(content)
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(cleaned)
	if err != nil {
		result.Error = "failed to convert article content"
		return result
	}

	title := inoreaderTitle(item.Title, content)
	if title == "" {
		title = "Saved from Inoreader"
	}
	now := time.Now()
	article := &model.Article{
		ID:          newID(),
		Title:       title,
		URL:         urlStr,
		ContentHTML: cleaned,
		ContentMD:   markdown,
		Author:      strings.TrimSpace(item.Author),
		SiteName:    strings.TrimSpace(item.Origin.Title),
		WordCount:   countWords(cleaned),
		Source:      "inoreader",
		Status:      "unread",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := h.Store.CreateArticle(article); err != nil {
		log.Printf("failed to save Inoreader article: %v", err)
		result.Error = "failed to save article"
		return result
	}

	result.ID = article.ID
	result.Status = "created"
	return result
}

func inoreaderTitle(rawTitle, content string) string {
	title := htmlText(rawTitle)
	contentText := htmlText(content)
	if utf8.RuneCountInString(title) <= 200 && title != contentText {
		return title
	}
	if heading := firstHTMLHeading(content); heading != "" {
		return truncateRunes(heading, 200)
	}
	return truncateRunes(title, 200)
}

func firstHTMLHeading(value string) string {
	root, err := html.Parse(strings.NewReader(value))
	if err != nil {
		return ""
	}
	var find func(*html.Node) string
	find = func(node *html.Node) string {
		if node.Type == html.ElementNode && (node.Data == "h1" || node.Data == "h2") {
			return strings.Join(strings.Fields(nodeText(node)), " ")
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if heading := find(child); heading != "" {
				return heading
			}
		}
		return ""
	}
	return find(root)
}

func htmlText(value string) string {
	root, err := html.Parse(strings.NewReader(value))
	if err != nil {
		return strings.Join(strings.Fields(value), " ")
	}
	return strings.Join(strings.Fields(nodeText(root)), " ")
}

func nodeText(node *html.Node) string {
	var text strings.Builder
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			text.WriteString(current.Data)
			text.WriteByte(' ')
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return text.String()
}

func truncateRunes(value string, limit int) string {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:limit])) + "..."
}

func firstInoreaderURL(item inoreaderItem) string {
	for _, links := range [][]inoreaderLink{item.Canonical, item.Alternate} {
		for _, link := range links {
			if href := strings.TrimSpace(link.Href); href != "" {
				return href
			}
		}
	}
	return ""
}

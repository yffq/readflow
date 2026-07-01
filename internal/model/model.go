package model

import (
	"time"
)

type User struct {
	ID           string    `json:"id"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type APIKey struct {
	ID        string    `json:"id"`
	UserID    string    `json:"-"`
	KeyPrefix string    `json:"key_prefix"`
	KeyHash   string    `json:"-"`
	Name      string    `json:"name"`
	LastUsed  time.Time `json:"last_used"`
	CreatedAt time.Time `json:"created_at"`
}

type Article struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	URL              string    `json:"url,omitempty"`
	ContentHTML      string    `json:"-"`
	ContentMD        string    `json:"-"`
	Author           string    `json:"author,omitempty"`
	SiteName         string    `json:"site_name,omitempty"`
	WordCount        int       `json:"word_count"`
	Source           string    `json:"source"`
	ExtractionFailed bool      `json:"extraction_failed"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ArticleExport struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	URL              string    `json:"url,omitempty"`
	ContentMarkdown  string    `json:"content_markdown"`
	Author           string    `json:"author,omitempty"`
	SiteName         string    `json:"site_name,omitempty"`
	WordCount        int       `json:"word_count"`
	Source           string    `json:"source"`
	ExtractionFailed bool      `json:"extraction_failed"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"saved_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type SaveRequest struct {
	URL   string `json:"url,omitempty"`
	HTML  string `json:"html,omitempty"`
	Title string `json:"title,omitempty"`
}

type ExportResponse struct {
	Count   int              `json:"count"`
	Next    string           `json:"next,omitempty"`
	Results []ArticleExport  `json:"results"`
}

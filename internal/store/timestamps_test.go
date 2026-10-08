package store

import (
	"testing"
	"time"

	"github.com/readflow/readflow/internal/model"
)

func newTimestampTestStore(t *testing.T) *Store {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	migrationSQL := `
	CREATE TABLE IF NOT EXISTS articles (id TEXT PRIMARY KEY, title TEXT NOT NULL DEFAULT '', url TEXT DEFAULT '', content_html TEXT NOT NULL DEFAULT '', content_md TEXT NOT NULL DEFAULT '', author TEXT NOT NULL DEFAULT '', site_name TEXT NOT NULL DEFAULT '', word_count INTEGER NOT NULL DEFAULT 0, source TEXT NOT NULL DEFAULT 'url', extraction_failed INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL DEFAULT 'unread', created_at DATETIME NOT NULL DEFAULT (datetime('now')), updated_at DATETIME NOT NULL DEFAULT (datetime('now')));
	`
	if err := s.Migrate(migrationSQL); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return s
}

func TestCreateArticleStoresUTC(t *testing.T) {
	s := newTimestampTestStore(t)
	defer s.Close()

	// 本地 +8 时间写入，应被转为 UTC 存储
	created := time.Date(2026, 7, 1, 22, 48, 31, 123456789, time.FixedZone("CST", 8*3600))
	a := &model.Article{ID: "t1", Title: "T", CreatedAt: created, UpdatedAt: created}
	if err := s.CreateArticle(a); err != nil {
		t.Fatalf("CreateArticle: %v", err)
	}

	var createdAt, updatedAt time.Time
	if err := s.db.QueryRow("SELECT created_at, updated_at FROM articles WHERE id = 't1'").Scan(&createdAt, &updatedAt); err != nil {
		t.Fatalf("scan: %v", err)
	}
	want := time.Date(2026, 7, 1, 14, 48, 31, 123456789, time.UTC)
	if !createdAt.Equal(want) || !updatedAt.Equal(want) {
		t.Fatalf("stored = (%v, %v), want UTC %v", createdAt, updatedAt, want)
	}
}

func TestNormalizeArticleTimestamps(t *testing.T) {
	s := newTimestampTestStore(t)
	defer s.Close()

	// 旧格式（Go time.Time.String()，含单调时钟与本地时区）
	old := "2026-07-01 22:48:31.252863 +0800 CST m=+63.991626959"
	if _, err := s.db.Exec("INSERT INTO articles (id, title, created_at, updated_at) VALUES ('old', 'Old', ?, ?)", old, old); err != nil {
		t.Fatalf("insert old: %v", err)
	}

	if err := s.NormalizeArticleTimestamps(); err != nil {
		t.Fatalf("NormalizeArticleTimestamps: %v", err)
	}

	var createdAt time.Time
	if err := s.db.QueryRow("SELECT created_at FROM articles WHERE id = 'old'").Scan(&createdAt); err != nil {
		t.Fatalf("scan old: %v", err)
	}
	want := time.Date(2026, 7, 1, 14, 48, 31, 252863000, time.UTC)
	if !createdAt.Equal(want) {
		t.Fatalf("old created_at = %v, want %v", createdAt, want)
	}
}

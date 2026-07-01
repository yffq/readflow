package store

import (
	"testing"
	"time"

	"github.com/readflow/readflow/internal/model"
)

func TestStore_CRUD(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	migrationSQL := `
	CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY, password_hash TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT (datetime('now')));
	CREATE TABLE IF NOT EXISTS api_keys (id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, key_prefix TEXT NOT NULL, key_hash TEXT NOT NULL UNIQUE, name TEXT NOT NULL DEFAULT '', last_used DATETIME, created_at DATETIME NOT NULL DEFAULT (datetime('now')));
	CREATE TABLE IF NOT EXISTS articles (id TEXT PRIMARY KEY, title TEXT NOT NULL DEFAULT '', url TEXT DEFAULT '', content_html TEXT NOT NULL DEFAULT '', content_md TEXT NOT NULL DEFAULT '', author TEXT NOT NULL DEFAULT '', site_name TEXT NOT NULL DEFAULT '', word_count INTEGER NOT NULL DEFAULT 0, source TEXT NOT NULL DEFAULT 'url', extraction_failed INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL DEFAULT 'unread', created_at DATETIME NOT NULL DEFAULT (datetime('now')), updated_at DATETIME NOT NULL DEFAULT (datetime('now')));
	`
	if err := s.Migrate(migrationSQL); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	t.Run("User", func(t *testing.T) {
		exists, err := s.UserExists()
		if err != nil {
			t.Fatalf("UserExists: %v", err)
		}
		if exists {
			t.Fatal("expected no user")
		}

		_, err = s.CreateUser("testpass123")
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}

		exists, err = s.UserExists()
		if err != nil || !exists {
			t.Fatal("expected user to exist")
		}

		valid, err := s.ValidatePassword("testpass123")
		if err != nil || !valid {
			t.Fatal("expected valid password")
		}

		valid, err = s.ValidatePassword("wrong")
		if err != nil || valid {
			t.Fatal("expected invalid password")
		}
	})

	t.Run("APIKeys", func(t *testing.T) {
		err := s.CreateAPIKey("default", "rf_abc123", "hash_abc", "TestKey")
		if err != nil {
			t.Fatalf("CreateAPIKey: %v", err)
		}

		keys, err := s.ListAPIKeys("default")
		if err != nil {
			t.Fatalf("ListAPIKeys: %v", err)
		}
		if len(keys) != 1 {
			t.Fatalf("expected 1 key, got %d", len(keys))
		}

		err = s.ValidateAPIKey("hash_abc")
		if err != nil {
			t.Fatalf("ValidateAPIKey: %v", err)
		}

		err = s.ValidateAPIKey("bad_hash")
		if err == nil {
			t.Fatal("expected error for invalid key")
		}

		err = s.DeleteAPIKey("default", keys[0].ID)
		if err != nil {
			t.Fatalf("DeleteAPIKey: %v", err)
		}

		keys, _ = s.ListAPIKeys("default")
		if len(keys) != 0 {
			t.Fatal("expected 0 keys after delete")
		}
	})

	t.Run("Articles", func(t *testing.T) {
		now := time.Now()
		a := &model.Article{
			ID:          "test-article-1",
			Title:       "Test Article",
			URL:         "https://example.com/test",
			ContentHTML: "<p>Hello World</p>",
			ContentMD:   "Hello World",
			Author:      "Test Author",
			SiteName:    "Example",
			WordCount:   2,
			Source:      "url",
			Status:      "unread",
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		err := s.CreateArticle(a)
		if err != nil {
			t.Fatalf("CreateArticle: %v", err)
		}

		id, dup := s.CheckArticleByURL("https://example.com/test")
		if !dup || id != "test-article-1" {
			t.Fatal("expected duplicate detection")
		}

		id, dup = s.CheckArticleByURL("https://example.com/other")
		if dup || id != "" {
			t.Fatal("expected no duplicate for different URL")
		}

		got, err := s.GetArticle("test-article-1")
		if err != nil || got == nil {
			t.Fatal("expected article to exist")
		}
		if got.Title != "Test Article" {
			t.Fatalf("expected 'Test Article', got '%s'", got.Title)
		}

		articles, total, err := s.ListArticles("unread", 10, 0)
		if err != nil {
			t.Fatalf("ListArticles: %v", err)
		}
		if total != 1 || len(articles) != 1 {
			t.Fatalf("expected 1 article, got %d/%d", len(articles), total)
		}

		err = s.UpdateArticleStatus("test-article-1", "archived")
		if err != nil {
			t.Fatalf("UpdateArticleStatus: %v", err)
		}

		got, _ = s.GetArticle("test-article-1")
		if got.Status != "archived" {
			t.Fatal("expected archived status")
		}

		articles, total, _ = s.ListArticles("unread", 10, 0)
		if total != 0 {
			t.Fatal("expected 0 unread after archive")
		}

		exports, count, err := s.ListArticlesSince(time.Time{}, 10, 0)
		if err != nil || count != 1 {
			t.Fatalf("ListArticlesSince: err=%v count=%d", err, count)
		}
		if len(exports) != 1 {
			t.Fatal("expected 1 export result")
		}

		err = s.DeleteArticle("test-article-1")
		if err != nil {
			t.Fatalf("DeleteArticle: %v", err)
		}

		got, _ = s.GetArticle("test-article-1")
		if got != nil {
			t.Fatal("expected nil after delete")
		}
	})
}

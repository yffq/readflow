package store

import (
	"database/sql"
	"time"

	"github.com/readflow/readflow/internal/model"
)

func (s *Store) CreateArticle(a *model.Article) error {
	_, err := s.db.Exec(`
		INSERT INTO articles (id, title, url, content_html, content_md, author, site_name, word_count, source, extraction_failed, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		a.ID, a.Title, a.URL, a.ContentHTML, a.ContentMD, a.Author, a.SiteName,
		a.WordCount, a.Source, boolToInt(a.ExtractionFailed), a.Status, a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (s *Store) GetArticle(id string) (*model.Article, error) {
	a := &model.Article{}
	var extractionFailed int
	var createdAt, updatedAt string
	err := s.db.QueryRow(`
		SELECT id, title, url, content_html, content_md, author, site_name, word_count, source, extraction_failed, status, created_at, updated_at
		FROM articles WHERE id = ?
	`, id).Scan(
		&a.ID, &a.Title, &a.URL, &a.ContentHTML, &a.ContentMD, &a.Author, &a.SiteName,
		&a.WordCount, &a.Source, &extractionFailed, &a.Status, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.ExtractionFailed = extractionFailed != 0
	a.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	a.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	return a, nil
}

func (s *Store) ListArticles(status string, limit, offset int) ([]model.Article, int, error) {
	var count int
	countSQL := "SELECT COUNT(*) FROM articles"
	querySQL := `SELECT id, title, url, content_html, content_md, author, site_name, word_count, source, extraction_failed, status, created_at, updated_at FROM articles`

	var args []interface{}
	if status != "" {
		countSQL += " WHERE status = ?"
		querySQL += " WHERE status = ?"
		args = append(args, status)
	}
	querySQL += " ORDER BY created_at DESC LIMIT ? OFFSET ?"

	if err := s.db.QueryRow(countSQL, args...).Scan(&count); err != nil {
		return nil, 0, err
	}

	queryArgs := append(args, limit, offset)
	rows, err := s.db.Query(querySQL, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var articles []model.Article
	for rows.Next() {
		var a model.Article
		var ef int
		var createdAt, updatedAt string
		if err := rows.Scan(&a.ID, &a.Title, &a.URL, &a.ContentHTML, &a.ContentMD, &a.Author, &a.SiteName,
			&a.WordCount, &a.Source, &ef, &a.Status, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}
		a.ExtractionFailed = ef != 0
		a.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		a.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
		articles = append(articles, a)
	}
	return articles, count, rows.Err()
}

func (s *Store) UpdateArticleStatus(id, status string) error {
	_, err := s.db.Exec("UPDATE articles SET status = ?, updated_at = datetime('now') WHERE id = ?", status, id)
	return err
}

func (s *Store) DeleteArticle(id string) error {
	_, err := s.db.Exec("DELETE FROM articles WHERE id = ?", id)
	return err
}

func (s *Store) CheckArticleByURL(url string) (string, bool) {
	var id string
	err := s.db.QueryRow("SELECT id FROM articles WHERE url = ? AND url != ''", url).Scan(&id)
	return id, err == nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

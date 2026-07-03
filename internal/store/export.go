package store

import (
	"time"

	"github.com/readflow/readflow/internal/model"
)

func (s *Store) ListArticlesSince(updatedAfter time.Time, limit, offset int) ([]model.ArticleExport, int, error) {
	var count int
	countSQL := "SELECT COUNT(*) FROM articles WHERE updated_at > ?"
	querySQL := `SELECT id, title, url, content_html, content_md, author, site_name, word_count, source, extraction_failed, status, created_at, updated_at
		FROM articles WHERE updated_at > ? ORDER BY updated_at DESC LIMIT ? OFFSET ?`

	if err := s.db.QueryRow(countSQL, updatedAfter.Format("2006-01-02 15:04:05")).Scan(&count); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(querySQL, updatedAfter.Format("2006-01-02 15:04:05"), limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []model.ArticleExport
	for rows.Next() {
		var a model.ArticleExport
		var ef int
		var createdAt, updatedAt string
		if err := rows.Scan(&a.ID, &a.Title, &a.URL, &a.ContentHTML, &a.ContentMarkdown, &a.Author, &a.SiteName,
			&a.WordCount, &a.Source, &ef, &a.Status, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}
		a.ExtractionFailed = ef != 0
		a.CreatedAt = parseTime(createdAt)
		a.UpdatedAt = parseTime(updatedAt)
		results = append(results, a)
	}
	return results, count, rows.Err()
}

func (s *Store) CountArticles(status string) (int, error) {
	var count int
	var err error
	if status == "" {
		err = s.db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&count)
	} else {
		err = s.db.QueryRow("SELECT COUNT(*) FROM articles WHERE status = ?", status).Scan(&count)
	}
	return count, err
}

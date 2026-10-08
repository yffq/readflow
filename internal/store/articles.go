package store

import (
	"database/sql"
	"strings"
	"time"

	"github.com/readflow/readflow/internal/model"
)

const timeLayout = "2006-01-02 15:04:05.000000000"

func (s *Store) CreateArticle(a *model.Article) error {
	_, err := s.db.Exec(`
		INSERT INTO articles (id, title, url, content_html, content_md, author, site_name, word_count, source, extraction_failed, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		a.ID, a.Title, a.URL, a.ContentHTML, a.ContentMD, a.Author, a.SiteName,
		a.WordCount, a.Source, boolToInt(a.ExtractionFailed), a.Status,
		a.CreatedAt.UTC().Format(timeLayout), a.UpdatedAt.UTC().Format(timeLayout),
	)
	return err
}

// NormalizeArticleTimestamps 将历史遗留的时间戳统一转换为固定宽度、UTC 的
// timeLayout 字符串，保证 updated_at 的字符串比较正确。
func (s *Store) NormalizeArticleTimestamps() error {
	rows, err := s.db.Query("SELECT id, created_at, updated_at FROM articles")
	if err != nil {
		return err
	}

	type item struct {
		id, createdAt, updatedAt string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.createdAt, &it.updatedAt); err != nil {
			rows.Close()
			return err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	// 先释放连接：MaxOpenConns(1) 下，后续 UPDATE 需要复用同一条连接
	rows.Close()

	for _, it := range items {
		c := parseTime(it.createdAt).UTC().Format(timeLayout)
		u := parseTime(it.updatedAt).UTC().Format(timeLayout)
		if c != it.createdAt || u != it.updatedAt {
			if _, err := s.db.Exec("UPDATE articles SET created_at = ?, updated_at = ? WHERE id = ?", c, u, it.id); err != nil {
				return err
			}
		}
	}
	return nil
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
	a.CreatedAt = parseTime(createdAt)
	a.UpdatedAt = parseTime(updatedAt)
	return a, nil
}

func (s *Store) ListArticles(status string, limit, offset int, asc bool) ([]model.Article, int, error) {
	var count int
	countSQL := "SELECT COUNT(*) FROM articles"
	querySQL := `SELECT id, title, url, content_html, content_md, author, site_name, word_count, source, extraction_failed, status, created_at, updated_at FROM articles`

	var args []interface{}
	if status != "" {
		countSQL += " WHERE status = ?"
		querySQL += " WHERE status = ?"
		args = append(args, status)
	}
	order := "DESC"
	if asc {
		order = "ASC"
	}
	querySQL += " ORDER BY created_at " + order + " LIMIT ? OFFSET ?"

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
		a.CreatedAt = parseTime(createdAt)
		a.UpdatedAt = parseTime(updatedAt)
		articles = append(articles, a)
	}
	return articles, count, rows.Err()
}

func (s *Store) UpdateArticleStatus(id, status string) error {
	_, err := s.db.Exec("UPDATE articles SET status = ?, updated_at = ? WHERE id = ?", status, time.Now().UTC().Format(timeLayout), id)
	return err
}

func (s *Store) DeleteArticle(id string) error {
	_, err := s.db.Exec("DELETE FROM articles WHERE id = ?", id)
	return err
}

func (s *Store) DeleteArticles(ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := make([]byte, 0, len(ids)*2)
	args := make([]interface{}, len(ids))
	placeholders = append(placeholders, '?')
	args[0] = ids[0]
	for i := 1; i < len(ids); i++ {
		placeholders = append(placeholders, ',', '?')
		args[i] = ids[i]
	}
	res, err := s.db.Exec("DELETE FROM articles WHERE id IN ("+string(placeholders)+")", args...)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

func (s *Store) CheckArticleByURL(url string) (string, bool) {
	var id string
	err := s.db.QueryRow("SELECT id FROM articles WHERE url = ? AND url != ''", url).Scan(&id)
	return id, err == nil
}

func parseTime(s string) time.Time {
	s = strings.TrimSpace(s)
	// 扫描时 modernc 驱动会把时间串解析为 time.Time，database/sql 再格式化为 RFC3339Nano 字符串
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	// 兜底：SQLite datetime('now') 的原始格式（UTC）
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t
	}
	return time.Now()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

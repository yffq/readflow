package sessionstore

import (
	"database/sql"
	"time"
)

type SQLStore struct {
	db *sql.DB
}

func New(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) Delete(token string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func (s *SQLStore) Find(token string) ([]byte, bool, error) {
	var data []byte
	var expiry time.Time
	err := s.db.QueryRow("SELECT data, expiry FROM sessions WHERE token = ?", token).Scan(&data, &expiry)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if time.Now().After(expiry) {
		s.db.Exec("DELETE FROM sessions WHERE token = ?", token)
		return nil, false, nil
	}
	return data, true, nil
}

func (s *SQLStore) Commit(token string, data []byte, expiry time.Time) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO sessions (token, data, expiry) VALUES (?, ?, ?)",
		token, data, expiry,
	)
	return err
}

func (s *SQLStore) All() (map[string]time.Time, error) {
	rows, err := s.db.Query("SELECT token, expiry FROM sessions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]time.Time)
	for rows.Next() {
		var token string
		var expiry time.Time
		if err := rows.Scan(&token, &expiry); err != nil {
			return nil, err
		}
		result[token] = expiry
	}
	return result, rows.Err()
}

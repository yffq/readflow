package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

func (s *Store) UserExists() (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count > 0, err
}

func (s *Store) CreateUser(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	id := "default"
	_, err = s.db.Exec("INSERT INTO users (id, password_hash) VALUES (?, ?)", id, string(hash))
	return id, err
}

func (s *Store) ValidatePassword(password string) (bool, error) {
	var hash string
	err := s.db.QueryRow("SELECT password_hash FROM users WHERE id = 'default'").Scan(&hash)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil, nil
}

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Store) CreateAPIKey(userID, keyPrefix, keyHash, name string) error {
	_, err := s.db.Exec(
		"INSERT INTO api_keys (id, user_id, key_prefix, key_hash, name) VALUES (?, ?, ?, ?, ?)",
		newID(), userID, keyPrefix, keyHash, name,
	)
	return err
}

type APIKeyRow struct {
	ID        string
	KeyPrefix string
	Name      string
	LastUsed  sql.NullTime
	CreatedAt string
}

func (s *Store) ListAPIKeys(userID string) ([]APIKeyRow, error) {
	rows, err := s.db.Query(
		"SELECT id, key_prefix, name, last_used, created_at FROM api_keys WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKeyRow
	for rows.Next() {
		var k APIKeyRow
		if err := rows.Scan(&k.ID, &k.KeyPrefix, &k.Name, &k.LastUsed, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (s *Store) DeleteAPIKey(userID, keyID string) error {
	_, err := s.db.Exec("DELETE FROM api_keys WHERE id = ? AND user_id = ?", keyID, userID)
	return err
}

func (s *Store) ValidateAPIKey(keyHash string) error {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM api_keys WHERE key_hash = ?", keyHash).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrInvalidAPIKey
	}
	s.db.Exec("UPDATE api_keys SET last_used = datetime('now') WHERE key_hash = ?", keyHash)
	return nil
}

type ErrInvalidAPIKeyType struct{}

func (e ErrInvalidAPIKeyType) Error() string { return "invalid api key" }

var ErrInvalidAPIKey = ErrInvalidAPIKeyType{}

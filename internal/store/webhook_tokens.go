package store

import (
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
)

type WebhookTokenRow struct {
	ID          string
	TokenPrefix string
	Name        string
	LastUsed    sql.NullTime
	CreatedAt   string
}

func (s *Store) CreateWebhookToken(userID, tokenPrefix, tokenHash, name string) error {
	_, err := s.db.Exec(
		"INSERT INTO webhook_tokens (id, user_id, token_prefix, token_hash, name) VALUES (?, ?, ?, ?, ?)",
		newID(), userID, tokenPrefix, tokenHash, name,
	)
	return err
}

func (s *Store) ListWebhookTokens(userID string) ([]WebhookTokenRow, error) {
	rows, err := s.db.Query(
		"SELECT id, token_prefix, name, last_used, created_at FROM webhook_tokens WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []WebhookTokenRow
	for rows.Next() {
		var token WebhookTokenRow
		if err := rows.Scan(&token.ID, &token.TokenPrefix, &token.Name, &token.LastUsed, &token.CreatedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

func (s *Store) DeleteWebhookToken(userID, tokenID string) error {
	_, err := s.db.Exec("DELETE FROM webhook_tokens WHERE id = ? AND user_id = ?", tokenID, userID)
	return err
}

func (s *Store) ValidateWebhookToken(token string) error {
	prefix := token
	if len(prefix) > 11 {
		prefix = prefix[:11]
	}
	rows, err := s.db.Query("SELECT token_hash FROM webhook_tokens WHERE token_prefix = ?", prefix)
	if err != nil {
		return err
	}

	hash := sha256.Sum256([]byte(token))
	expected := hex.EncodeToString(hash[:])
	matchedHash := ""
	for rows.Next() {
		var storedHash string
		if err := rows.Scan(&storedHash); err != nil {
			rows.Close()
			return err
		}
		storedBytes, decodeErr := hex.DecodeString(storedHash)
		if decodeErr != nil {
			rows.Close()
			return decodeErr
		}
		expectedBytes, decodeErr := hex.DecodeString(expected)
		if decodeErr != nil {
			rows.Close()
			return decodeErr
		}
		if subtle.ConstantTimeCompare(storedBytes, expectedBytes) == 1 {
			matchedHash = storedHash
			break
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if matchedHash == "" {
		return ErrInvalidAPIKey
	}
	_, err = s.db.Exec("UPDATE webhook_tokens SET last_used = datetime('now') WHERE token_hash = ?", matchedHash)
	return err
}

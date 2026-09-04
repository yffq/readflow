package store

// WebAuthnCredentialRow is a stored WebAuthn credential. Data is the
// JSON-serialized credential (serialized by the handler layer to keep this
// package independent of the WebAuthn library types).
type WebAuthnCredentialRow struct {
	ID   string
	Data []byte
}

// CreateWebAuthnCredential stores a credential for a user.
func (s *Store) CreateWebAuthnCredential(userID, credentialID string, data []byte) error {
	_, err := s.db.Exec(
		"INSERT INTO webauthn_credentials (id, user_id, credential_data) VALUES (?, ?, ?)",
		credentialID, userID, string(data),
	)
	return err
}

// ListWebAuthnCredentials returns all credentials for a user.
func (s *Store) ListWebAuthnCredentials(userID string) ([]WebAuthnCredentialRow, error) {
	rows, err := s.db.Query(
		"SELECT id, credential_data FROM webauthn_credentials WHERE user_id = ? ORDER BY created_at ASC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []WebAuthnCredentialRow
	for rows.Next() {
		var c WebAuthnCredentialRow
		var data string
		if err := rows.Scan(&c.ID, &data); err != nil {
			return nil, err
		}
		c.Data = []byte(data)
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

// DeleteWebAuthnCredential removes a credential for a user.
func (s *Store) DeleteWebAuthnCredential(userID, credentialID string) error {
	_, err := s.db.Exec(
		"DELETE FROM webauthn_credentials WHERE id = ? AND user_id = ?",
		credentialID, userID,
	)
	return err
}

// CountWebAuthnCredentials returns the number of credentials for a user.
func (s *Store) CountWebAuthnCredentials(userID string) (int, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM webauthn_credentials WHERE user_id = ?",
		userID,
	).Scan(&count)
	return count, err
}

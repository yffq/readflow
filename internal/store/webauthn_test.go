package store

import "testing"

func TestWebAuthnCredentialCRUD(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	schema := `CREATE TABLE IF NOT EXISTS webauthn_credentials (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL DEFAULT 'default',
		credential_data TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT (datetime('now'))
	);`
	if err := s.Migrate(schema); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := s.CreateWebAuthnCredential("default", "cred-1", []byte(`{"id":"cred-1"}`)); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.CreateWebAuthnCredential("default", "cred-2", []byte(`{"id":"cred-2"}`)); err != nil {
		t.Fatalf("create: %v", err)
	}

	rows, err := s.ListWebAuthnCredentials("default")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(rows))
	}
	if rows[0].ID != "cred-1" || string(rows[0].Data) != `{"id":"cred-1"}` {
		t.Fatalf("unexpected row 0: %+v", rows[0])
	}

	count, err := s.CountWebAuthnCredentials("default")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}

	if err := s.DeleteWebAuthnCredential("default", "cred-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	rows, err = s.ListWebAuthnCredentials("default")
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "cred-2" {
		t.Fatalf("expected 1 credential (cred-2), got %+v", rows)
	}
}

func TestWebAuthnCredentialEmpty(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	if err := s.Migrate(`CREATE TABLE IF NOT EXISTS webauthn_credentials (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL DEFAULT 'default',
		credential_data TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT (datetime('now'))
	);`); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	rows, err := s.ListWebAuthnCredentials("default")
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 credentials, got %d", len(rows))
	}

	count, err := s.CountWebAuthnCredentials("default")
	if err != nil {
		t.Fatalf("count empty: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected count 0, got %d", count)
	}
}

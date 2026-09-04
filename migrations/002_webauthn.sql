CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL DEFAULT 'default',
    credential_data TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_webauthn_user ON webauthn_credentials(user_id);

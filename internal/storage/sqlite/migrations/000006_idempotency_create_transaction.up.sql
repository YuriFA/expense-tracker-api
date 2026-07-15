CREATE TABLE idempotency_keys (
    id TEXT PRIMARY KEY,
    idempotency_key TEXT NOT NULL,
    user_id TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'completed', 'failed')),
    response_status INTEGER,
    response_headers TEXT,
    response_body BLOB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP DEFAULT (DATETIME('now', '+1 day')),
    UNIQUE (user_id, idempotency_key)
);

CREATE INDEX idx_idempotency_keys_expires ON idempotency_keys (expires_at);

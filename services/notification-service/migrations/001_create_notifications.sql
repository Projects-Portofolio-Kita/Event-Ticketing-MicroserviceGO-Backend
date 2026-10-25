CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE notifications (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,

    kind VARCHAR(40) NOT NULL,
    title VARCHAR(120) NOT NULL,
    body TEXT NOT NULL,
    reference VARCHAR(64),

    read_at TIMESTAMP,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notifications_user_id
ON notifications(user_id, created_at DESC);

CREATE INDEX idx_notifications_kind
ON notifications(kind);

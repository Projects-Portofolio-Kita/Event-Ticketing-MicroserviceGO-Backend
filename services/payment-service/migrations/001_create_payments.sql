CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE payments (
    id UUID PRIMARY KEY,
    reservation_id UUID NOT NULL,
    user_id UUID NOT NULL,

    provider_payment_id VARCHAR(255) UNIQUE,
    qris_reference VARCHAR(255),

    qr_string TEXT,
    qr_url TEXT,

    amount DECIMAL(12,2) NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'IDR',

    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    gateway VARCHAR(30) NOT NULL,

    failure_description TEXT,

    expired_at TIMESTAMP NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_payments_reservation_id
ON payments(reservation_id);

CREATE INDEX idx_payments_user_id
ON payments(user_id);

CREATE INDEX idx_payments_status
ON payments(status);
CREATE TABLE IF NOT EXISTS reservations (
    id            UUID PRIMARY KEY,
    user_id       UUID NOT NULL,
    event_id      UUID NOT NULL,
    booking_ref   VARCHAR(50) NOT NULL UNIQUE,
    status        VARCHAR(20) NOT NULL DEFAULT 'LOCKED',
    expires_at    TIMESTAMPTZ NOT NULL,
    total_amount  DECIMAL(12, 2) NOT NULL,
    currency      VARCHAR(10) NOT NULL DEFAULT 'IDR',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reservations_user_id ON reservations (user_id);
CREATE INDEX IF NOT EXISTS idx_reservations_event_id ON reservations (event_id);

CREATE TABLE IF NOT EXISTS reservation_seats (
    id             UUID PRIMARY KEY,
    reservation_id UUID NOT NULL REFERENCES reservations (id) ON DELETE CASCADE,
    seat_id        UUID NOT NULL,
    section        VARCHAR(100),
    row            VARCHAR(10),
    number         VARCHAR(10),
    price          DECIMAL(12, 2)
);

CREATE INDEX IF NOT EXISTS idx_reservation_seats_reservation_id ON reservation_seats (reservation_id);
CREATE INDEX IF NOT EXISTS idx_reservation_seats_seat_id ON reservation_seats (seat_id);

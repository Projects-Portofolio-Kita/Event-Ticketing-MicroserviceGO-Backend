-- 002: quota-based ticket tiers (VIP/VVIP/Regular or custom) alongside the
-- existing seat locks. GORM AutoMigrate applies the same shape at startup;
-- this file keeps the manual/CI path in sync, following the project's
-- migration convention.

-- Which flow a reservation used: SEATED (seat rows) or TIER (quota items).
ALTER TABLE reservations
    ADD COLUMN IF NOT EXISTS mode VARCHAR(10) NOT NULL DEFAULT 'SEATED';

-- One line per tier of a TIER reservation: how many tickets, at the price
-- catalog-service reported (clients never send prices).
CREATE TABLE IF NOT EXISTS reservation_items (
    id             UUID PRIMARY KEY,
    reservation_id UUID           NOT NULL REFERENCES reservations (id) ON DELETE CASCADE,
    category_id    UUID           NOT NULL,
    category_name  VARCHAR(100)   NOT NULL DEFAULT '',
    quantity       INTEGER        NOT NULL,
    unit_price     DECIMAL(12, 2) NOT NULL,
    subtotal       DECIMAL(12, 2) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_reservation_items_reservation_id ON reservation_items (reservation_id);
CREATE INDEX IF NOT EXISTS idx_reservation_items_category_id    ON reservation_items (category_id);
CREATE INDEX IF NOT EXISTS idx_reservations_mode                ON reservations (mode);

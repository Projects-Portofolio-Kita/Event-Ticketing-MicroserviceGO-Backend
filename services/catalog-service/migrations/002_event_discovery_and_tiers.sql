-- 002: discovery metadata (city/category/poster) and quota-based ticket tiers.
-- GORM AutoMigrate applies the same shape at startup; this file keeps the
-- manual/CI path in sync, following the project's migration convention.

ALTER TABLE events
    ADD COLUMN IF NOT EXISTS city        VARCHAR(100) NOT NULL DEFAULT 'Jakarta',
    ADD COLUMN IF NOT EXISTS category    VARCHAR(50)  NOT NULL DEFAULT 'concert',
    ADD COLUMN IF NOT EXISTS description TEXT         NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS poster_url  VARCHAR(512) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sale_mode   VARCHAR(10)  NOT NULL DEFAULT 'SEATED';

ALTER TABLE seat_categories
    ADD COLUMN IF NOT EXISTS quota       INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS description TEXT    NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_events_city      ON events (city);
CREATE INDEX IF NOT EXISTS idx_events_category  ON events (category);
CREATE INDEX IF NOT EXISTS idx_events_created_by ON events (created_by);
CREATE INDEX IF NOT EXISTS idx_events_sale_mode ON events (sale_mode);

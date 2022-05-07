-- +migrate Up
ALTER TABLE play_url_caches 
ADD COLUMN IF NOT EXISTS format_type SMALLINT NOT NULL DEFAULT 3;

-- +migrate Down
ALTER TABLE play_url_caches 
DROP COLUMN IF EXISTS format_type;

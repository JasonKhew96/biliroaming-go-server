-- +migrate Up
ALTER TABLE play_url_caches 
ADD COLUMN IF NOT EXISTS quality SMALLINT NOT NULL DEFAULT 127;

-- +migrate Down
ALTER TABLE play_url_caches 
DROP COLUMN IF EXISTS quality;

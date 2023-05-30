-- +migrate Up
ALTER TABLE play_url_caches 
ADD COLUMN IF NOT EXISTS prefer_code_type BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE play_url_caches
ALTER COLUMN prefer_code_type DROP DEFAULT;

-- +migrate Down
ALTER TABLE play_url_caches 
DROP COLUMN IF EXISTS prefer_code_type;

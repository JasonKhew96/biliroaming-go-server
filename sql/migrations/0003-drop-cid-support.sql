-- +migrate Up
ALTER TABLE play_url_caches 
DROP COLUMN IF EXISTS cid;

-- +migrate Down
DELETE FROM play_url_caches;
ALTER TABLE play_url_caches 
ADD COLUMN cid BIGINT NOT NULL;

-- +migrate Up
ALTER TABLE users
    ALTER COLUMN uid SET NOT NULL,
    ALTER COLUMN name TYPE varchar(16) USING name::varchar(16),
    ALTER COLUMN name SET NOT NULL,
    ALTER COLUMN name DROP DEFAULT,
    ALTER COLUMN vip_due_date TYPE timestamp USING vip_due_date::timestamp,
    ALTER COLUMN vip_due_date SET NOT NULL,
    ALTER COLUMN created_at TYPE timestamp using created_at::timestamp,
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at TYPE timestamp using created_at::timestamp,
    ALTER COLUMN updated_at SET NOT NULL;
ALTER TABLE access_keys
    ALTER COLUMN key TYPE char(32) USING key::char(32),
    ALTER COLUMN uid SET NOT NULL,
    ALTER COLUMN created_at TYPE timestamp USING created_at::timestamp,
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at TYPE timestamp USING created_at::timestamp,
    ALTER COLUMN updated_at SET NOT NULL,
    ADD CONSTRAINT uid_fkey FOREIGN KEY (uid) REFERENCES users (uid);
DROP TABLE play_url_caches IF EXIST;
CREATE TABLE play_url_caches(
    episode_id BIGINT PRIMARY KEY NOT NULL,
    is_vip BOOLEAN NOT NULL,
    cid BIGINT NOT NULL,
    area SMALLINT NOT NULL,
    device_type SMALLINT NOT NULL,
    data JSON NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
ALTER TABLE th_season_caches RENAME COLUMN json_data TO data;
ALTER TABLE th_season_caches
    ALTER COLUMN season_id DROP DEFAULT,
    ALTER COLUMN data TYPE json USING data::json,
    ALTER COLUMN data SET NOT NULL,
    ALTER COLUMN is_vip DROP DEFAULT,
    ALTER COLUMN is_vip SET NOT NULL,
    ALTER COLUMN created_at TYPE timestamp USING created_at::timestamp,
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at TYPE timestamp USING created_at::timestamp,
    ALTER COLUMN updated_at SET NOT NULL;
ALTER TABLE th_season_episode_caches
    ALTER COLUMN episode_id DROP DEFAULT,
    ALTER COLUMN season_id SET NOT NULL,
    ALTER COLUMN created_at TYPE timestamp USING created_at::timestamp,
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at TYPE timestamp USING created_at::timestamp,
    ALTER COLUMN updated_at SET NOT NULL,
    ADD CONSTRAINT season_id_fkey FOREIGN KEY (season_id) REFERENCES th_season_caches (season_id);
ALTER TABLE th_subtitle_caches RENAME COLUMN json_data TO data;
ALTER TABLE th_subtitle_caches
    ALTER COLUMN episode_id DROP DEFAULT,
    ALTER COLUMN data TYPE json USING data::json,
    ALTER COLUMN data SET NOT NULL,
    ALTER COLUMN created_at TYPE timestamp USING created_at::timestamp,
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at TYPE timestamp USING created_at::timestamp,
    ALTER COLUMN updated_at SET NOT NULL;
DROP SEQUENCE
    users_uid_seq,
    play_url_caches_id_seq,
    th_season_caches_season_id_seq,
    th_subtitle_caches_episode_id_seq,
    th_season_episode_caches_episode_id_seq CASCADE;

-- +migrate Up
DROP TABLE play_url_caches;
CREATE TABLE play_url_caches(
    id SERIAL PRIMARY KEY,
    episode_id BIGINT NOT NULL,
    is_vip BOOLEAN NOT NULL,
    cid BIGINT NOT NULL,
    area SMALLINT NOT NULL,
    device_type SMALLINT NOT NULL,
    data JSON NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

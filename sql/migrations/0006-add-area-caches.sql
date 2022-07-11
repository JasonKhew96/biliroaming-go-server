-- +migrate Up
CREATE TABLE IF NOT EXISTS season_area_caches(
    season_id BIGINT PRIMARY KEY NOT NULL,
    cn BOOLEAN,
    hk BOOLEAN,
    tw BOOLEAN,
    th BOOLEAN,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE TABLE IF NOT EXISTS episode_area_caches(
    episode_id BIGINT PRIMARY KEY NOT NULL,
    cn BOOLEAN,
    hk BOOLEAN,
    tw BOOLEAN,
    th BOOLEAN,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- +migrate Down
DROP TABLE IF EXISTS season_area_caches;
DROP TABLE IF EXISTS episode_area_caches;

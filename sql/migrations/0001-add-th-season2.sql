-- +migrate Up
CREATE TABLE IF NOT EXISTS th_season2_caches(
    season_id BIGINT PRIMARY KEY NOT NULL,
    is_vip BOOLEAN NOT NULL,
    data JSON NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE TABLE IF NOT EXISTS th_season2_episode_caches(
    episode_id BIGINT PRIMARY KEY NOT NULL,
    season_id BIGINT REFERENCES th_season2_caches(season_id) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);


-- +migrate Down
DROP TABLE IF EXISTS th_season2_episode_caches;
DROP TABLE IF EXISTS th_season2_caches;

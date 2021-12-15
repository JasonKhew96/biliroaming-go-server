-- +migrate Up
CREATE TABLE th_episode_caches(
    episode_id BIGINT PRIMARY KEY NOT NULL,
    data JSON NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- +migrate Down
DROP TABLE th_episode_caches;

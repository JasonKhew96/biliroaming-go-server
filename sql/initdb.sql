CREATE TABLE users(
    uid BIGINT PRIMARY KEY NOT NULL,
    name VARCHAR(16) NOT NULL,
    vip_due_date TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE TABLE access_keys(
    key CHAR(32) PRIMARY KEY NOT NULL,
    uid BIGINT REFERENCES users(uid) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
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
CREATE TABLE th_season_caches(
    season_id BIGINT PRIMARY KEY NOT NULL,
    is_vip BOOLEAN NOT NULL,
    data JSON NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE TABLE th_season_episode_caches(
    episode_id BIGINT PRIMARY KEY NOT NULL,
    season_id BIGINT REFERENCES th_season_caches(season_id) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE TABLE th_subtitle_caches(
    episode_id BIGINT PRIMARY KEY NOT NULL,
    data JSON NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
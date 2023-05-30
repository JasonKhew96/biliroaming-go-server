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
    client_type VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE TABLE play_url_caches(
    id SERIAL PRIMARY KEY,
    episode_id BIGINT NOT NULL,
    is_vip BOOLEAN NOT NULL,
    area SMALLINT NOT NULL,
    device_type SMALLINT NOT NULL,
    format_type SMALLINT NOT NULL,
    quality SMALLINT NOT NULL,
    prefer_code_type BOOLEAN NOT NULL,
    data JSON NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE TABLE th_season_caches(
    id SERIAL PRIMARY KEY,
    season_id BIGINT NOT NULL,
    is_vip BOOLEAN NOT NULL,
    data JSON NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE TABLE th_season_episode_caches(
    episode_id BIGINT PRIMARY KEY NOT NULL,
    season_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE TABLE th_season2_caches(
    id SERIAL PRIMARY KEY,
    season_id BIGINT NOT NULL,
    is_vip BOOLEAN NOT NULL,
    data JSON NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE TABLE th_season2_episode_caches(
    episode_id BIGINT PRIMARY KEY NOT NULL,
    season_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE TABLE th_episode_caches(
    episode_id BIGINT PRIMARY KEY NOT NULL,
    data JSON NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE TABLE th_subtitle_caches(
    episode_id BIGINT PRIMARY KEY NOT NULL,
    data JSON NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE TABLE season_area_caches(
    season_id BIGINT PRIMARY KEY NOT NULL,
    cn BOOLEAN,
    hk BOOLEAN,
    tw BOOLEAN,
    th BOOLEAN,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE TABLE episode_area_caches(
    episode_id BIGINT PRIMARY KEY NOT NULL,
    cn BOOLEAN,
    hk BOOLEAN,
    tw BOOLEAN,
    th BOOLEAN,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
-- +migrate Up
DELETE FROM th_season_episode_caches;
DELETE FROM th_season_caches;
ALTER TABLE th_season_episode_caches DROP CONSTRAINT th_season_episode_caches_season_id_fkey;
ALTER TABLE th_season_caches DROP CONSTRAINT th_season_caches_pkey;
ALTER TABLE th_season_caches ADD COLUMN id SERIAL PRIMARY KEY;

DELETE FROM th_season2_caches;
DELETE FROM th_season2_episode_caches;
ALTER TABLE th_season2_episode_caches DROP CONSTRAINT th_season2_episode_caches_season_id_fkey;
ALTER TABLE th_season2_caches DROP CONSTRAINT th_season2_caches_pkey;
ALTER TABLE th_season2_caches ADD COLUMN id SERIAL PRIMARY KEY;


-- +migrate Down
DELETE FROM th_season_episode_caches;
DELETE FROM th_season_caches;
ALTER TABLE th_season_caches DROP CONSTRAINT th_season_caches_pkey;
ALTER TABLE th_season_caches DROP COLUMN id;
ALTER TABLE th_season_caches ADD PRIMARY KEY (season_id);
ALTER TABLE th_season_episode_caches ADD CONSTRAINT th_season_episode_caches_season_id_fkey FOREIGN KEY (season_id) REFERENCES th_season_caches(season_id);

DELETE FROM th_season2_caches;
DELETE FROM th_season2_episode_caches;
ALTER TABLE th_season2_caches DROP CONSTRAINT th_season2_caches_pkey;
ALTER TABLE th_season2_caches DROP COLUMN id;
ALTER TABLE th_season2_caches ADD PRIMARY KEY (season_id);
ALTER TABLE th_season2_episode_caches ADD CONSTRAINT th_season2_episode_caches_season_id_fkey FOREIGN KEY (season_id) REFERENCES th_season2_caches(season_id);

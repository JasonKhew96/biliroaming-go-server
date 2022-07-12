-- +migrate Up
ALTER TABLE th_season_caches DROP CONSTRAINT th_season_caches_pkey;
ALTER TABLE th_season_caches ADD COLUMN id SERIAL constraint;
ALTER TABLE th_season_caches ADD PRIMARY KEY (id);


-- +migrate Down
ALTER TABLE th_season_caches DROP CONSTRAINT th_season_caches_pkey;
ALTER TABLE th_season_caches DROP COLUMN id;
ALTER TABLE th_season_caches ADD PRIMARY KEY (season_id);

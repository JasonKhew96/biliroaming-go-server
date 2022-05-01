package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/JasonKhew96/biliroaming-go-server/models"
	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"golang.org/x/net/context"
)

// Config database configurations
type Config struct {
	Host     string
	User     string
	Password string
	DBName   string
	Port     int
	Debug    bool
}

// DbHelper database helper
type DbHelper struct {
	ctx context.Context
	db  *sql.DB
}

// NewDBConnection new database connection
func NewDBConnection(c *Config) (*DbHelper, error) {
	boil.DebugMode = c.Debug
	// connect to database
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		c.Host, c.User, c.Password, c.DBName, c.Port,
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// sql migrate
	migrations := &migrate.FileMigrationSource{
		Dir: "sql/migrations",
	}
	n, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Applied %d migrations!\n", n)

	return &DbHelper{ctx: context.Background(), db: db}, err
}

// GetKey get access key data
func (h *DbHelper) GetKey(key string) (*models.AccessKey, error) {
	return models.AccessKeys(models.AccessKeyWhere.Key.EQ(key)).One(h.ctx, h.db)
}

// InsertOrUpdateKey insert or update access key data
func (h *DbHelper) InsertOrUpdateKey(key string, uid int64) error {
	var accessKeyTable models.AccessKey
	accessKeyTable.Key = key
	accessKeyTable.UID = uid
	return accessKeyTable.Upsert(h.ctx, h.db, true, []string{"key"}, boil.Whitelist("updated_at"), boil.Infer())
}

// CleanupAccessKeys cleanup access keys if exceeds duration
func (h *DbHelper) CleanupAccessKeys(d time.Duration) (int64, error) {
	startTS := time.Now().Add(-d)
	return models.AccessKeys(models.AccessKeyWhere.UpdatedAt.LTE(startTS)).DeleteAll(h.ctx, h.db)
}

// GetUser get user from uid
func (h *DbHelper) GetUser(uid int64) (*models.User, error) {
	return models.Users(models.UserWhere.UID.EQ(uid)).One(h.ctx, h.db)
}

// GetUserFromKey get user from access key
func (h *DbHelper) GetUserFromKey(key string) (*models.User, error) {
	return models.Users(
		qm.InnerJoin("access_keys ON access_keys.uid = users.uid"),
		models.AccessKeyWhere.Key.EQ(key),
	).One(h.ctx, h.db)
}

// InsertOrUpdateUser insert or update user data
func (h *DbHelper) InsertOrUpdateUser(uid int64, name string, vipDueDate time.Time) error {
	var userTable models.User
	userTable.UID = uid
	userTable.Name = name
	userTable.VipDueDate = vipDueDate
	return userTable.Upsert(h.ctx, h.db, true, []string{"uid"}, boil.Whitelist("name", "vip_due_date", "updated_at"), boil.Infer())
}

// CleanupUsers cleanup users if exceeds duration
func (h *DbHelper) CleanupUsers(duration time.Duration) (int64, error) {
	startTS := time.Now().Add(-duration)
	return models.Users(models.UserWhere.UpdatedAt.LTE(startTS)).DeleteAll(h.ctx, h.db)
}

// GetPlayURLCache get play url caching with device type, area or episode ID
func (h *DbHelper) GetPlayURLCache(deviceType DeviceType, formatType FormatType, area Area, isVIP bool, episodeID int64) (*models.PlayURLCach, error) {
	return models.PlayURLCaches(
		models.PlayURLCachWhere.DeviceType.EQ(int16(deviceType)),
		models.PlayURLCachWhere.FormatType.EQ(int16(formatType)),
		models.PlayURLCachWhere.Area.EQ(int16(area)),
		models.PlayURLCachWhere.IsVip.EQ(isVIP),
		models.PlayURLCachWhere.EpisodeID.EQ(episodeID),
	).One(h.ctx, h.db)
}

// InsertOrUpdatePlayURLCache insert or update play url cache data
func (h *DbHelper) InsertOrUpdatePlayURLCache(deviceType DeviceType, formatType FormatType, area Area, isVIP bool, episodeID int64, data []byte) error {
	var playUrlTable models.PlayURLCach

	oldData, err := h.GetPlayURLCache(deviceType, formatType, area, isVIP, episodeID)
	if err == nil {
		playUrlTable.ID = oldData.ID
	}

	playUrlTable.DeviceType = int16(deviceType)
	playUrlTable.FormatType = int16(formatType)
	playUrlTable.Area = int16(area)
	playUrlTable.IsVip = isVIP
	playUrlTable.EpisodeID = episodeID
	playUrlTable.Data = data
	return playUrlTable.Upsert(h.ctx, h.db, true, []string{"id"}, boil.Whitelist("data", "updated_at"), boil.Greylist("device_type", "area", "is_vip"))
}

// CleanupPlayURLCache cleanup playurl if exceeds duration
func (h *DbHelper) CleanupPlayURLCache(duration time.Duration) (int64, error) {
	startTS := time.Now().Add(-duration)
	return models.PlayURLCaches(models.PlayURLCachWhere.UpdatedAt.LTE(startTS)).DeleteAll(h.ctx, h.db)
}

// GetTHSeasonCache get season api cache from season id
func (h *DbHelper) GetTHSeasonCache(seasonID int64, isVIP bool) (*models.THSeasonCach, error) {
	return models.THSeasonCaches(
		models.THSeasonCachWhere.SeasonID.EQ(seasonID),
		models.THSeasonCachWhere.IsVip.EQ(isVIP),
	).One(h.ctx, h.db)
}

// InsertOrUpdateTHSeasonCache insert or update season api cache
func (h *DbHelper) InsertOrUpdateTHSeasonCache(seasonID int64, isVIP bool, data []byte) error {
	var thSeasonCacheTable models.THSeasonCach
	thSeasonCacheTable.SeasonID = seasonID
	thSeasonCacheTable.IsVip = isVIP
	thSeasonCacheTable.Data = data
	return thSeasonCacheTable.Upsert(h.ctx, h.db, true, []string{"season_id"}, boil.Whitelist("data", "updated_at"), boil.Greylist("is_vip"))
}

// CleanupTHSeasonCache cleanup th season if exceeds duration
func (h *DbHelper) CleanupTHSeasonCache(duration time.Duration) (int64, error) {
	startTS := time.Now().Add(-duration)
	return models.THSeasonCaches(models.THSeasonCachWhere.UpdatedAt.LTE(startTS)).DeleteAll(h.ctx, h.db)
}

// GetTHSeasonCache get season api cache from episode id
func (h *DbHelper) GetTHSeasonEpisodeCache(episodeID int64, isVIP bool) (*models.THSeasonCach, error) {
	return models.THSeasonCaches(
		qm.InnerJoin("th_season_episode_caches ON th_season_episode_caches.season_id = th_season_caches.season_id"),
		models.THSeasonEpisodeCachWhere.EpisodeID.EQ(episodeID),
		models.THSeasonCachWhere.IsVip.EQ(isVIP),
	).One(h.ctx, h.db)
}

// InsertOrUpdateTHSeasonCache insert or update season api cache
func (h *DbHelper) InsertOrUpdateTHSeasonEpisodeCache(episodeID int64, seasonID int64) error {
	var thSeasonEpisodeCacheTable models.THSeasonEpisodeCach
	thSeasonEpisodeCacheTable.EpisodeID = episodeID
	thSeasonEpisodeCacheTable.SeasonID = seasonID
	return thSeasonEpisodeCacheTable.Upsert(h.ctx, h.db, false, nil, boil.Infer(), boil.Infer())
}

// GetTHSubtitleCache get th subtitle api cache from season id
func (h *DbHelper) GetTHSubtitleCache(episodeID int64) (*models.THSubtitleCach, error) {
	return models.THSubtitleCaches(models.THSubtitleCachWhere.EpisodeID.EQ(episodeID)).One(h.ctx, h.db)
}

// InsertOrUpdateTHSubtitleCache insert or update th subtitle api cache
func (h *DbHelper) InsertOrUpdateTHSubtitleCache(episodeID int64, data []byte) error {
	var thSubtitleCacheTable models.THSubtitleCach
	thSubtitleCacheTable.EpisodeID = episodeID
	thSubtitleCacheTable.Data = data
	return thSubtitleCacheTable.Upsert(h.ctx, h.db, true, []string{"episode_id"}, boil.Whitelist("data", "updated_at"), boil.Infer())
}

// CleanupTHSubtitleCache cleanup th subtitle if exceeds duration
func (h *DbHelper) CleanupTHSubtitleCache(duration time.Duration) (int64, error) {
	startTS := time.Now().Add(-duration)
	return models.THSubtitleCaches(models.THSubtitleCachWhere.UpdatedAt.LTE(startTS)).DeleteAll(h.ctx, h.db)
}

// GetTHSeason2Cache get season2 api cache from season id
func (h *DbHelper) GetTHSeason2Cache(seasonID int64, isVIP bool) (*models.THSeason2Cach, error) {
	return models.THSeason2Caches(
		models.THSeason2CachWhere.SeasonID.EQ(seasonID),
		models.THSeason2CachWhere.IsVip.EQ(isVIP),
	).One(h.ctx, h.db)
}

// InsertOrUpdateTHSeason2Cache insert or update season2 api cache
func (h *DbHelper) InsertOrUpdateTHSeason2Cache(seasonID int64, isVIP bool, data []byte) error {
	var thSeason2CacheTable models.THSeason2Cach
	thSeason2CacheTable.SeasonID = seasonID
	thSeason2CacheTable.IsVip = isVIP
	thSeason2CacheTable.Data = data
	return thSeason2CacheTable.Upsert(h.ctx, h.db, true, []string{"season_id"}, boil.Whitelist("data", "updated_at"), boil.Greylist("is_vip"))
}

// GetTHSeason2EpisodeCache get season api cache from episode id
func (h *DbHelper) GetTHSeason2EpisodeCache(episodeID int64, isVIP bool) (*models.THSeason2Cach, error) {
	return models.THSeason2Caches(
		qm.InnerJoin("th_season2_episode_caches ON th_season2_episode_caches.season_id = th_season2_caches.season_id"),
		models.THSeason2EpisodeCachWhere.EpisodeID.EQ(episodeID),
		models.THSeason2CachWhere.IsVip.EQ(isVIP),
	).One(h.ctx, h.db)
}

// InsertOrUpdateTHSeason2EpisodeCache insert or update season api cache
func (h *DbHelper) InsertOrUpdateTHSeason2EpisodeCache(episodeID int64, seasonID int64) error {
	var thSeason2EpisodeCacheTable models.THSeason2EpisodeCach
	thSeason2EpisodeCacheTable.EpisodeID = episodeID
	thSeason2EpisodeCacheTable.SeasonID = seasonID
	return thSeason2EpisodeCacheTable.Upsert(h.ctx, h.db, false, nil, boil.Infer(), boil.Infer())
}

// GetTHEpisodeCache get th episode api cache from episode id
func (h *DbHelper) GetTHEpisodeCache(episodeID int64) (*models.THEpisodeCach, error) {
	return models.THEpisodeCaches(models.THEpisodeCachWhere.EpisodeID.EQ(episodeID)).One(h.ctx, h.db)
}

// InsertOrUpdateTHEpisodeCache insert or update th episode api cache
func (h *DbHelper) InsertOrUpdateTHEpisodeCache(episodeID int64, data []byte) error {
	var thEpisodeCacheTable models.THEpisodeCach
	thEpisodeCacheTable.EpisodeID = episodeID
	thEpisodeCacheTable.Data = data
	return thEpisodeCacheTable.Upsert(h.ctx, h.db, true, []string{"episode_id"}, boil.Whitelist("data", "updated_at"), boil.Infer())
}

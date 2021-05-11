package database

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config database configurations
type Config struct {
	Host     string
	User     string
	Password string
	DBName   string
	Port     int
}

// Database database helper
type Database struct {
	*gorm.DB
}

// NewDBConnection new database connection
func NewDBConnection(c *Config) (*Database, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d",
		c.Host, c.User, c.Password, c.DBName, c.Port,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	db.AutoMigrate(
		&AccessKeys{},
		&Users{},
		&PlayURLCache{},
	)

	return &Database{db}, err
}

// GetKey ...
func (db *Database) GetKey(key string) (*AccessKeys, error) {
	var data AccessKeys
	err := db.Where(&AccessKeys{Key: key}).First(&data).Error
	return &data, err
}

// InsertOrUpdateKey ...
func (db *Database) InsertOrUpdateKey(key string, uid int, dueDate time.Time) (int64, error) {
	data, err := db.GetKey(key)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		result := db.Create(&AccessKeys{Key: key, UID: uid, DueDate: dueDate})
		return result.RowsAffected, result.Error
	} else if err != nil {
		return 0, err
	}
	result := db.Model(data).Updates(AccessKeys{Key: key, UID: uid, DueDate: dueDate})
	return result.RowsAffected, result.Error
}

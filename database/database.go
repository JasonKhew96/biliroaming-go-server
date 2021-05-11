package database

import (
	"fmt"

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

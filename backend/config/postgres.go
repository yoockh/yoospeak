package config

import (
	"errors"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var PostgresDB *gorm.DB

func InitPostgres() error {
	uri := os.Getenv("POSTGRES_URI")
	if uri == "" {
		return errors.New("POSTGRES_URI environment variable is not set")
	}
	db, err := gorm.Open(postgres.Open(uri), &gorm.Config{})
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// Connection Pooling settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	PostgresDB = db
	return nil
}

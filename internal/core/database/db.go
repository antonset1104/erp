package database

import (
	"fmt"
	"log"

	"erp-golang/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB initializes GORM with SQLite or PostgreSQL based on config
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.DBType {
	case "postgres":
		dialector = postgres.Open(cfg.DBDSN)
	case "sqlite":
		fallthrough
	default:
		dialector = sqlite.Open(cfg.DBDSN)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	DB = db
	log.Printf("[DB] Connected successfully to %s (%s)", cfg.DBType, cfg.DBDSN)
	return db, nil
}

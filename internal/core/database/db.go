package database

import (
	"fmt"
	"log"
	"strings"

	"erp-golang/internal/config"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB initializes GORM with SQLite (pure-Go modernc.org/sqlite via glebarez/sqlite) or PostgreSQL based on config
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

	// Ensure in-memory SQLite databases share state across all requests in pure-Go SQLite
	if cfg.DBType != "postgres" && strings.Contains(cfg.DBDSN, ":memory:") {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.SetMaxOpenConns(1)
		}
	}

	DB = db
	log.Printf("[DB] Connected successfully to %s (%s)", cfg.DBType, cfg.DBDSN)
	return db, nil
}

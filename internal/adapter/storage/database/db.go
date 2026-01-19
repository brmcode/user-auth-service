package database

import (
	"fmt"
	"time"

	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/pkg/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type DB struct {
	*gorm.DB
}

func New(config *config.DB) (*DB, error) {
	dsn := fmt.Sprintf("%s://%s:%s@%s:%s/%s?sslmode=disable",
		config.Connection,
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Name,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false,
		},
	})
	if err != nil {
		return nil, err
	}

	// Configure connection pool for better concurrency
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Set connection pool settings for optimal concurrent performance
	sqlDB.SetMaxOpenConns(25)                 // Maximum number of open connections
	sqlDB.SetMaxIdleConns(10)                 // Maximum number of idle connections
	sqlDB.SetConnMaxLifetime(5 * time.Minute) // Maximum connection lifetime
	sqlDB.SetConnMaxIdleTime(1 * time.Minute) // Maximum idle connection time

	return &DB{DB: db}, nil
}

func (db *DB) Migrate() error {
	return db.AutoMigrate(
		&domain.User{},
		&domain.Session{},
		&domain.OauthAccount{},
	)
}

func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

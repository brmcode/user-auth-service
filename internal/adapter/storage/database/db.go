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
	if err := db.AutoMigrate(
		&domain.Role{},
		&domain.User{},
		&domain.UserRole{},
		&domain.Session{},
		&domain.OauthAccount{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}
	return db.seedRoles()
}

func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (db *DB) seedRoles() error {
	defaultRoles := []domain.Role{
		{
			Code:        domain.USER_ROLE,
			Name:        "User",
			Description: "Standard user with self-service access",
		},
		{
			Code:        domain.ADMIN_ROLE,
			Name:        "Administrator",
			Description: "Full administrative access",
		},
	}

	for _, role := range defaultRoles {
		result := db.Where(domain.Role{Code: role.Code}).FirstOrCreate(&role)
		if result.Error != nil {
			return fmt.Errorf("seeding role %s: %w", role.Code, result.Error)
		}
	}
	return nil
}

package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OauthAccount struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Username       string         `gorm:"type:varchar(60);uniqueIndex:idx_user_provider;not null" json:"username"`
	Provider       string         `gorm:"type:varchar(20);not null;uniqueIndex:idx_provider_user;uniqueIndex:idx_user_provider" json:"provider"`
	ProviderUserID string         `gorm:"type:varchar(255);not null;uniqueIndex:idx_provider_user" json:"provider_user_id"`
	Email          string         `gorm:"type:varchar(100);not null" json:"email"`
	CreatedAt      time.Time      `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

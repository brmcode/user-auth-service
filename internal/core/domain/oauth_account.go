package domain

import (
	"time"

	"github.com/google/uuid"
)

type OauthAccount struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Username       string    `gorm:"type:varchar(60);index;not null" json:"username"`
	Provider       string    `gorm:"type:varchar(20);not null;uniqueIndex:idx_provider_user" json:"provider"`
	ProviderUserID string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_provider_user" json:"provider_user_id"`
	Email          string    `gorm:"type:varchar(100);not null" json:"email"`
	CreatedAt      time.Time `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

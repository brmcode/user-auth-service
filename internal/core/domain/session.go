package domain

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Username     string    `gorm:"type:varchar(60);index;not null" json:"username"`
	RefreshToken string    `gorm:"type:text;index;not null" json:"refresh_token"`
	UserAgent    string    `gorm:"type:varchar(255);not null" json:"user_agent"`
	ClientIP     string    `gorm:"type:varchar(60);not null" json:"client_ip"`
	IsBlocked    bool      `gorm:"type:boolean;not null;default:false" json:"is_blocked"`
	ExpiresAt    time.Time `gorm:"type:timestamptz;not null" json:"expires_at"`
	CreatedAt    time.Time `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

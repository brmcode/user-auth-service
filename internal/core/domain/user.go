package domain

import (
	"time"

	"gorm.io/gorm"
)

const (
	USER_ROLE  = "USER"
	ADMIN_ROLE = "ADMIN"
)

type User struct {
	Username          string         `gorm:"type:varchar(60);primaryKey" json:"username"`
	FirstName         string         `gorm:"type:varchar(20);not null" json:"first_name"`
	LastName          string         `gorm:"type:varchar(20);not null" json:"last_name"`
	Email             string         `gorm:"type:varchar(100);unique;not null" json:"email"`
	HashedPassword    string         `gorm:"type:varchar(255);not null" json:"-"`
	Role              string         `gorm:"type:varchar(10);not null" json:"role"`
	PasswordChangedAt time.Time      `gorm:"type:timestamptz;not null;default:'0001-01-01T00:00:00Z'" json:"password_changed_at"`
	CreatedAt         time.Time      `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	Session           Session        `gorm:"foreignKey:Username" json:"-"`
	OauthAccounts     []OauthAccount `gorm:"foreignKey:Username;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (u *User) BeforeDelete(tx *gorm.DB) error {
	return tx.Unscoped().
		Where("username = ?", u.Username).
		Delete(&OauthAccount{}).Error
}

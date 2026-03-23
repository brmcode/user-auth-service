package domain

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	Username          string         `gorm:"type:varchar(60);primaryKey" json:"username"`
	FirstName         string         `gorm:"type:varchar(20);not null" json:"first_name"`
	LastName          string         `gorm:"type:varchar(20);not null" json:"last_name"`
	Email             string         `gorm:"type:varchar(100);unique;not null" json:"email"`
	ImageURL          string         `gorm:"type:text;not null" json:"image_url"`
	HashedPassword    string         `gorm:"type:varchar(255);not null" json:"-"`
	PasswordChangedAt time.Time      `gorm:"type:timestamptz;not null;default:'0001-01-01T00:00:00Z'" json:"password_changed_at"`
	CreatedAt         time.Time      `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	Roles         []Role         `gorm:"many2many:user_roles;joinForeignKey:Username;joinReferences:RoleCode;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"roles"`
	Session       Session        `gorm:"foreignKey:Username" json:"-"`
	OauthAccounts []OauthAccount `gorm:"foreignKey:Username;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (u *User) RoleCodes() []string {
	codes := make([]string, len(u.Roles))
	for i, r := range u.Roles {
		codes[i] = r.Code
	}
	return codes
}

func (u *User) HasRole(code string) bool {
	for _, r := range u.Roles {
		if r.Code == code {
			return true
		}
	}
	return false
}

func (u *User) PrimaryRole() string {
	if len(u.Roles) == 0 {
		return USER_ROLE
	}
	return u.Roles[0].Code
}

func (u *User) BeforeDelete(tx *gorm.DB) error {
	return tx.Unscoped().
		Where("username = ?", u.Username).
		Delete(&OauthAccount{}).Error
}

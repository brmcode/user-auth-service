package domain

import "time"

const (
	USER_ROLE  = "USER"
	ADMIN_ROLE = "ADMIN"
)

type Role struct {
	Code        string    `gorm:"primaryKey;type:varchar(10);unique;not null" json:"code"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Description string    `gorm:"type:varchar(255);not null" json:"description"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`

	Users []User `gorm:"many2many:user_roles;joinForeignKey:RoleCode;joinReferences:Username;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

type UserRole struct {
	Username string `gorm:"primaryKey;type:varchar(60);not null"`
	RoleCode string `gorm:"primaryKey;type:varchar(10);not null"`
}

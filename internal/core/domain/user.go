package domain

import (
	"time"

	"github.com/brmcode/user-auth-service/internal/adapter/auth"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	DEFAULT_USER = "user"
	USER_ROLE    = "USER"
	ADMIN_ROLE   = "ADMIN"
)

type User struct {
	Username          string         `gorm:"type:varchar(60);primaryKey" json:"username"`
	FirstName         string         `gorm:"type:varchar(20);not null" json:"first_name"`
	LastName          string         `gorm:"type:varchar(20);not null" json:"last_name"`
	Email             string         `gorm:"type:varchar(100);unique;not null" json:"email"`
	HashedPassword    string         `gorm:"type:varchar(255);not null" json:"-"`
	PasswordChangedAt time.Time      `gorm:"type:timestamptz;not null;default:'0001-01-01T00:00:00Z'" json:"password_changed_at"`
	CreatedAt         time.Time      `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	Role              string         `gorm:"type:varchar(10);not null" json:"role"`
	Session           Session        `gorm:"foreignKey:Username" json:"-"`
}

func GetUsername(ctx *gin.Context) string {
	value, exists := ctx.Get("authPayloadKey")
	if !exists {
		return DEFAULT_USER
	}
	return value.(*auth.Payload).Username
}
func GetRole(ctx *gin.Context) string {
	value, exists := ctx.Get("authPayloadKey")
	if !exists {
		return DEFAULT_USER
	}
	return value.(*auth.Payload).Role
}

func GetPayload(ctx *gin.Context) *auth.Payload {
	value, exists := ctx.Get("authPayloadKey")
	if !exists {
		return nil
	}
	return value.(*auth.Payload)
}

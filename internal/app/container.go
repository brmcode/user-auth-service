package app

import (
	"github.com/brmcode/user-auth-service/internal/adapter/google"
	"github.com/brmcode/user-auth-service/internal/adapter/storage/database"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/brmcode/user-auth-service/pkg/config"
)

type Container struct {
	Cfg              *config.Configuration
	DB               *database.DB
	Cache            port.CacheRepository
	UserRepo         port.UserRepository
	SessionRepo      port.SessionRepository
	OauthAccountRepo port.OauthAccountRepository
	UserService      port.UserService
	AuthService      port.AuthenticationService
	TokenService     port.TokenService
	IDTokenVerifier  *google.IDTokenVerifier
}

func (c *Container) Close() error {
	if c.Cache != nil {
		_ = c.Cache.Close()
	}
	if c.DB != nil {
		_ = c.DB.Close()
	}
	return nil
}

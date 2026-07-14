package middleware

import (
	"net/http"
	"strings"

	"github.com/brmcode/user-auth-service/internal/adapter/auth"
	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/response"
	"github.com/brmcode/user-auth-service/internal/adapter/storage/database"
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/brmcode/user-auth-service/pkg/i18n"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	authorizationHeaderKey = "Authorization"
	authorizationType      = "Bearer "
)

var tokenService port.TokenService
var _db *database.DB

func Set(ts port.TokenService, db *database.DB) {
	tokenService = ts
	_db = db
}

func Authorized(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(authorizationHeaderKey)
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(401, i18n.Translate("middleware.auth.header_missing")))
			return
		}
		if !strings.HasPrefix(authHeader, authorizationType) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(401, i18n.Translate("middleware.auth.invalid_format")))
			return
		}

		tokenString := strings.TrimPrefix(authHeader, authorizationType)
		if tokenService == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.NewError(500, i18n.Translate("middleware.auth.token_service_not_initialized")))
			return
		}

		payload, err := tokenService.VerifyToken(tokenString)
		if err != nil {
			if err == jwt.ErrTokenExpired {
				c.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(401, i18n.Translate("common.internal_error")))
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(401, i18n.Translate("common.internal_error")))
			return
		}

		if err := _db.First(&domain.User{}, "username = ?", payload.Username).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(401, i18n.Translate("middleware.auth.account_inactive")))
			return
		}

		if len(roles) > 0 {
			hasRequired := false
			for _, required := range roles {
				if payload.HasRole(required) {
					hasRequired = true
					break
				}
			}
			if !hasRequired {
				c.AbortWithStatusJSON(http.StatusForbidden, response.NewError(403, i18n.Translate("middleware.auth.insufficient_privileges")))
				return
			}
		}

		auth.SetPayload(c, payload)
		c.Next()
	}
}

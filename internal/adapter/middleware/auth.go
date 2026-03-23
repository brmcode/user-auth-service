package middleware

import (
	"net/http"
	"strings"

	"github.com/brmcode/user-auth-service/internal/adapter/auth"
	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/response"
	"github.com/brmcode/user-auth-service/internal/adapter/storage/database"
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/port"
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(401, "authorization header is missing"))
			return
		}
		if !strings.HasPrefix(authHeader, authorizationType) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(401, "invalid authorization format"))
			return
		}

		tokenString := strings.TrimPrefix(authHeader, authorizationType)
		if tokenService == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.NewError(500, "token service not initialized"))
			return
		}

		payload, err := tokenService.VerifyToken(tokenString)
		if err != nil {
			if err == jwt.ErrTokenExpired {
				c.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(401, err.Error()))
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(401, err.Error()))
			return
		}

		if err := _db.First(&domain.User{}, "username = ?", payload.Username).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(401, "your account is no longer active or may have been removed"))
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
				c.AbortWithStatusJSON(http.StatusForbidden, response.NewError(403, "insufficient privileges"))
				return
			}
		}

		auth.SetPayload(c, payload)
		c.Next()
	}
}

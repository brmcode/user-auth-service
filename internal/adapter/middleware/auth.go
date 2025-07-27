package middleware

import (
	"net/http"
	"strings"

	"github.com/brmcode/user-auth-service/internal/adapter/database"
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/dto/response"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/gin-gonic/gin"
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

func Authorized(role ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorizationHeader := c.GetHeader(authorizationHeaderKey)

		if authorizationHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(401, "authorization header is missing"))
			return
		}

		if !strings.HasPrefix(authorizationHeader, authorizationType) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(401, "invalid authorization format"))
			return
		}

		tokenString := strings.TrimPrefix(authorizationHeader, authorizationType)

		if tokenService == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.NewError(500, "token service not initialized"))
			return
		}

		payload, err := tokenService.VerifyToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(401, err.Error()))
			return
		}

		if err := _db.First(&domain.User{}, "username = ?", payload.Username).Error; err != nil {
			// This error can occur if the user was deleted or disabled after the token was issued.
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.NewError(401, "your account is no longer active or may have been removed"))
			return
		}
		if len(role) > 0 && role[0] != payload.Role {
			c.AbortWithStatusJSON(http.StatusForbidden, response.NewError(403, "insufficient privileges"))
			return
		}

		c.Set("authPayloadKey", payload)
		c.Next()
	}
}

package middleware

import (
	"net/http"
	"time"

	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/response"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

var limiter = rate.NewLimiter(rate.Every(time.Second), 40)

func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, response.NewError(429, "too many requests"))
			return
		}
		c.Next()
	}
}

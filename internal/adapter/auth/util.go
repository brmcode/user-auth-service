package auth

import (
	"github.com/gin-gonic/gin"
)

const (
	PayloadKey = "authPayloadKey"
)

func SetPayload(c *gin.Context, payload any) {
	c.Set(PayloadKey, payload)
}

func GetPayload(ctx *gin.Context) *Payload {
	value, exists := ctx.Get("authPayloadKey")
	if !exists {
		return nil
	}
	return value.(*Payload)
}

func GetUsername(ctx *gin.Context) string {
	payload := GetPayload(ctx)
	if payload == nil {
		return ""
	}
	return payload.Username
}

func GetRole(ctx *gin.Context) string {
	payload := GetPayload(ctx)
	if payload == nil {
		return ""
	}
	return payload.Role
}

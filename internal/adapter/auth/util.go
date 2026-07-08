package auth

import "github.com/gin-gonic/gin"

const PayloadKey = "authPayloadKey"

func SetPayload(c *gin.Context, payload any) { c.Set(PayloadKey, payload) }

func GetPayload(ctx *gin.Context) *Payload {
	v, exists := ctx.Get(PayloadKey)
	if !exists {
		return nil
	}
	return v.(*Payload)
}

func GetUsername(ctx *gin.Context) string {
	p := GetPayload(ctx)
	if p == nil {
		return ""
	}
	return p.Username
}

func GetRoles(ctx *gin.Context) []string {
	p := GetPayload(ctx)
	if p == nil {
		return nil
	}
	return p.Roles
}

func HasRole(ctx *gin.Context, role string) bool {
	p := GetPayload(ctx)
	if p == nil {
		return false
	}
	return p.HasRole(role)
}

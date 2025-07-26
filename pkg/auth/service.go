package auth

import "time"

type TokenService interface {
	GenerateToken(username string, role string, duration time.Duration) (string, error)
	VerifyToken(tokenString string) (*Payload, error)
}

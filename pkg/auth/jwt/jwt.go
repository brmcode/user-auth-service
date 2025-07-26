package jwt

import (
	"errors"
	"fmt"

	"time"

	"github.com/brmcode/user-auth-service/pkg/auth"
	"github.com/golang-jwt/jwt/v5"
)

const minSecretKeyLength = 32

var ErrSecretKeyTooShort = fmt.Errorf("secret key too short (min %d chars)", minSecretKeyLength)

type JWTService struct {
	secretKey []byte
}

// GenerateToken implements port.TokenService.
func (j *JWTService) GenerateToken(username string, role string, duration time.Duration) (string, *auth.Payload, error) {
	payload, err := auth.NewPayload(username, role, duration)
	if err != nil {
		return "", payload, err
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	token, err := jwtToken.SignedString(j.secretKey)
	return token, payload, err
}

// VerifyToken implements port.TokenService.
func (j *JWTService) VerifyToken(tokenString string) (*auth.Payload, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenUnverifiable
		}
		return j.secretKey, nil
	}

	payload := &auth.Payload{}
	jwtToken, err := jwt.ParseWithClaims(tokenString, payload, keyFunc)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, jwt.ErrTokenExpired
		}

		return nil, auth.ErrInvalidToken
	}

	finalPayload, ok := jwtToken.Claims.(*auth.Payload)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return finalPayload, nil
}

func New(secretKey string) (auth.TokenService, error) {
	if len(secretKey) < minSecretKeyLength {
		return nil, ErrSecretKeyTooShort
	}
	return &JWTService{secretKey: []byte(secretKey)}, nil
}

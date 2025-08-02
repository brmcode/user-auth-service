package jwt

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"time"

	"github.com/brmcode/user-auth-service/internal/adapter/auth"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/brmcode/user-auth-service/pkg/util"
	"github.com/golang-jwt/jwt/v5"
)

const minSecretKeyLength = 32

var ErrSecretKeyTooShort = fmt.Errorf("secret key too short (min %d chars)", minSecretKeyLength)

type JWTService struct {
	secretKey []byte
}

// VerifyRefreshToken implements port.TokenService.
func (j *JWTService) VerifyRefreshToken(tokenString string) (*auth.Payload, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid refresh token format")
	}

	base64Payload := parts[0]
	// secureToken := parts[1] // Can be used for Redis lookup or blacklist check later

	// Decode base64
	payloadJSON, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(base64Payload)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 payload: %w", err)
	}

	// Unmarshal JSON to Payload
	var payload auth.Payload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil, fmt.Errorf("invalid payload format: %w", err)
	}

	return &payload, nil
}

// GenerateRefreshToken implements port.TokenService.
func (j *JWTService) GenerateRefreshToken(username string, role string, duration time.Duration) (string, *auth.Payload, error) {
	payload, err := auth.NewPayload(username, role, duration)
	if err != nil {
		return "", payload, err
	}

	// Serialize payload to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}

	// Encode to base64
	base64Payload := base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(payloadJSON)
	secureToken, err := util.GenerateSecureToken(64)
	token := fmt.Sprintf("%s.%s", base64Payload, secureToken)

	return token, payload, err
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

func New(secretKey string) (port.TokenService, error) {
	if len(secretKey) < minSecretKeyLength {
		return nil, ErrSecretKeyTooShort
	}
	return &JWTService{secretKey: []byte(secretKey)}, nil
}

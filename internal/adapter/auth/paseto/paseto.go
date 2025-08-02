package paseto

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"time"

	"github.com/aead/chacha20poly1305"
	"github.com/brmcode/user-auth-service/internal/adapter/auth"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/brmcode/user-auth-service/pkg/util"

	"github.com/o1egl/paseto"
)

const minSymmetricKeLength = chacha20poly1305.KeySize

var ErrSecretKeyTooShort = fmt.Errorf("symmetric key too short (min %d chars)", minSymmetricKeLength)

type PasetoService struct {
	paseto       *paseto.V2
	symmetricKey []byte
}

// VerifyRefreshToken implements port.TokenService.
func (p *PasetoService) VerifyRefreshToken(tokenString string) (*auth.Payload, error) {
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
func (p *PasetoService) GenerateRefreshToken(username string, role string, duration time.Duration) (string, *auth.Payload, error) {
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
func (p *PasetoService) GenerateToken(username string, role string, duration time.Duration) (string, *auth.Payload, error) {
	payload, err := auth.NewPayload(username, role, duration)
	if err != nil {
		return "", payload, err
	}
	token, err := p.paseto.Encrypt(p.symmetricKey, payload, nil)

	return token, payload, err
}

// VerifyToken implements port.TokenService.
func (p *PasetoService) VerifyToken(tokenString string) (*auth.Payload, error) {
	payload := &auth.Payload{}
	err := p.paseto.Decrypt(tokenString, p.symmetricKey, payload, nil)
	if err != nil {
		return nil, auth.ErrInvalidToken
	}

	err = payload.Valid()
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func New(symmetricKey string) (port.TokenService, error) {
	if len(symmetricKey) < minSymmetricKeLength {
		return nil, ErrSecretKeyTooShort
	}

	return &PasetoService{
		paseto:       paseto.NewV2(),
		symmetricKey: []byte(symmetricKey),
	}, nil
}

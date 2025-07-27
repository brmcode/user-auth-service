package paseto

import (
	"fmt"

	"time"

	"github.com/aead/chacha20poly1305"
	"github.com/brmcode/user-auth-service/internal/adapter/auth"
	"github.com/brmcode/user-auth-service/internal/core/port"

	"github.com/o1egl/paseto"
)

const minSymmetricKeLength = chacha20poly1305.KeySize

var ErrSecretKeyTooShort = fmt.Errorf("symmetric key too short (min %d chars)", minSymmetricKeLength)

type PasetoService struct {
	paseto       *paseto.V2
	symmetricKey []byte
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

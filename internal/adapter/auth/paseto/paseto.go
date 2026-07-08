package paseto

import (
	"fmt"
	"time"

	"github.com/aead/chacha20poly1305"
	"github.com/brmcode/user-auth-service/internal/adapter/auth"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/google/uuid"
	"github.com/o1egl/paseto"
)

const minSymmetricKeyLength = chacha20poly1305.KeySize

var ErrSecretKeyTooShort = fmt.Errorf("symmetric key too short (min %d chars)", minSymmetricKeyLength)

type PasetoService struct {
	paseto       *paseto.V2
	symmetricKey []byte
}

// GenerateToken implements port.TokenService.
func (p *PasetoService) GenerateToken(tokenID uuid.UUID, username string, roles []string, duration time.Duration) (string, *auth.Payload, error) {
	payload, err := auth.NewPayload(tokenID, username, roles, duration)
	if err != nil {
		return "", nil, err
	}
	token, err := p.paseto.Encrypt(p.symmetricKey, payload, nil)
	return token, payload, err
}

func (p *PasetoService) VerifyToken(tokenString string) (*auth.Payload, error) {
	payload := &auth.Payload{}
	if err := p.paseto.Decrypt(tokenString, p.symmetricKey, payload, nil); err != nil {
		return nil, auth.ErrInvalidToken
	}
	if err := payload.Valid(); err != nil {
		return nil, err
	}
	return payload, nil
}

func New(symmetricKey string) (port.TokenService, error) {
	if len(symmetricKey) < minSymmetricKeyLength {
		return nil, ErrSecretKeyTooShort
	}
	return &PasetoService{
		paseto:       paseto.NewV2(),
		symmetricKey: []byte(symmetricKey),
	}, nil
}

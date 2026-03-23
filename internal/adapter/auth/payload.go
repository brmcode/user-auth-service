package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("token is invalid")
)

type Payload struct {
	ID        uuid.UUID  `json:"id"`
	Username  string     `json:"username"`
	Roles     []string   `json:"roles"`
	IssuedAt  time.Time  `json:"issued_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	NotBefore *time.Time `json:"not_before,omitempty"`
	Issuer    string     `json:"issuer,omitempty"`
	Subject   string     `json:"subject,omitempty"`
	Audience  []string   `json:"audience,omitempty"`
}

func (payload *Payload) HasRole(role string) bool {
	for _, r := range payload.Roles {
		if r == role {
			return true
		}
	}
	return false
}

func (payload *Payload) PrimaryRole() string {
	if len(payload.Roles) == 0 {
		return "USER"
	}
	return payload.Roles[0]
}

func (payload *Payload) Valid() error {
	now := time.Now()
	if payload.NotBefore != nil && now.Before(*payload.NotBefore) {
		return jwt.ErrTokenNotValidYet
	}
	if now.After(payload.ExpiresAt) {
		return jwt.ErrTokenExpired
	}
	return nil
}

func (payload *Payload) GetExpirationTime() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(payload.ExpiresAt), nil
}

func (payload *Payload) GetIssuedAt() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(payload.IssuedAt), nil
}

func (payload *Payload) GetNotBefore() (*jwt.NumericDate, error) {
	if payload.NotBefore == nil {
		return nil, nil
	}
	return jwt.NewNumericDate(*payload.NotBefore), nil
}

func (payload *Payload) GetIssuer() (string, error)  { return payload.Issuer, nil }
func (payload *Payload) GetSubject() (string, error) { return payload.Subject, nil }
func (payload *Payload) GetAudience() (jwt.ClaimStrings, error) {
	return jwt.ClaimStrings(payload.Audience), nil
}

func NewPayload(tokenID uuid.UUID, username string, roles []string, duration time.Duration) (*Payload, error) {
	if tokenID == uuid.Nil {
		var err error
		tokenID, err = uuid.NewRandom()
		if err != nil {
			return nil, err
		}
	}

	now := time.Now()
	return &Payload{
		ID:        tokenID,
		Username:  username,
		Roles:     roles,
		IssuedAt:  now,
		ExpiresAt: now.Add(duration),
	}, nil
}

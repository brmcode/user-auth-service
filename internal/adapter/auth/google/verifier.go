package google

import (
	"context"
	"errors"

	"google.golang.org/api/idtoken"
)

type IDTokenVerifier struct {
	clientID string
}

func NewIDTokenVerifier(clientID string) *IDTokenVerifier {
	return &IDTokenVerifier{clientID: clientID}
}

func (v *IDTokenVerifier) Verify(ctx context.Context, rawToken string) (*Payload, error) {
	payload, err := idtoken.Validate(ctx, rawToken, v.clientID)
	if err != nil {
		return nil, errors.New("invalid google id token")
	}

	email, _ := payload.Claims["email"].(string)
	if email == "" {
		return nil, errors.New("google token missing email claim")
	}

	return &Payload{
		Subject:   payload.Subject,
		Email:     email,
		FirstName: stringClaim(payload.Claims, "given_name"),
		LastName:  stringClaim(payload.Claims, "family_name"),
		Name:      stringClaim(payload.Claims, "name"),
		AvatarURL: stringClaim(payload.Claims, "picture"),
	}, nil
}

func stringClaim(claims map[string]any, key string) string {
	v, _ := claims[key].(string)
	return v
}

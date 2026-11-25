package auth

import (
	"context"

	appauth "api-core/pkg/auth"
)

// GoogleProvider adapts pkg/auth.GoogleOAuth to IdentityProvider, keeping the
// service free of oauth2 types.
type GoogleProvider struct {
	client *appauth.GoogleOAuth
}

var _ IdentityProvider = (*GoogleProvider)(nil)

func NewGoogleProvider(client *appauth.GoogleOAuth) *GoogleProvider {
	return &GoogleProvider{client: client}
}

func (p *GoogleProvider) AuthCodeURL(state string) string {
	return p.client.AuthCodeURL(state)
}

func (p *GoogleProvider) Authenticate(ctx context.Context, code string) (*Identity, error) {
	token, err := p.client.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	info, err := p.client.FetchUserInfo(ctx, token)
	if err != nil {
		return nil, err
	}

	return &Identity{
		Subject:       info.ID,
		Email:         info.Email,
		EmailVerified: info.VerifiedEmail,
		Name:          info.Name,
		Picture:       info.Picture,
		Locale:        info.Locale,
	}, nil
}

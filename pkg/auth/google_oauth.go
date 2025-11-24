package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleOAuthConfig groups the required OAuth2 configuration.
type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	HTTPClient   *http.Client
}

// GoogleOAuth wraps oauth2.Config to simplify Google sign-in/retrieval.
type GoogleOAuth struct {
	oauthConfig *oauth2.Config
	httpClient  *http.Client
}

// DefaultGoogleScopes defines the minimal profile information we request.
var DefaultGoogleScopes = []string{
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
}

// NewGoogleOAuth validates the configuration and returns a ready-to-use helper.
func NewGoogleOAuth(cfg GoogleOAuthConfig) (*GoogleOAuth, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RedirectURL == "" {
		return nil, errors.New("google oauth: missing client id/secret or redirect url")
	}

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = DefaultGoogleScopes
	}

	config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &GoogleOAuth{
		oauthConfig: config,
		httpClient:  httpClient,
	}, nil
}

// AuthCodeURL builds the Google authorization URL.
func (g *GoogleOAuth) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	return g.oauthConfig.AuthCodeURL(state, opts...)
}

// Exchange swaps the authorization code for access+refresh tokens.
func (g *GoogleOAuth) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := g.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("google oauth exchange: %w", err)
	}
	return token, nil
}

// Client returns an HTTP client that automatically injects the OAuth token.
func (g *GoogleOAuth) Client(ctx context.Context, token *oauth2.Token) *http.Client {
	if g.httpClient != nil && token == nil {
		return g.httpClient
	}
	return g.oauthConfig.Client(ctx, token)
}

// GoogleUserInfo is the payload returned by Google userinfo endpoint.
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

const googleUserInfoEndpoint = "https://www.googleapis.com/oauth2/v2/userinfo"

// FetchUserInfo calls Google's userinfo endpoint.
func (g *GoogleOAuth) FetchUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	if token == nil {
		return nil, errors.New("google oauth: nil token")
	}

	client := g.oauthConfig.Client(ctx, token)
	resp, err := client.Get(googleUserInfoEndpoint)
	if err != nil {
		return nil, fmt.Errorf("google oauth userinfo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("google oauth userinfo failed: status=%d", resp.StatusCode)
	}

	var info GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("google oauth decode userinfo: %w", err)
	}

	return &info, nil
}

// RefreshToken refreshes the OAuth token if a refresh token is present.
func (g *GoogleOAuth) RefreshToken(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	if token == nil || token.RefreshToken == "" {
		return nil, errors.New("google oauth: no refresh token")
	}

	ts := g.oauthConfig.TokenSource(ctx, token)
	newToken, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("google oauth refresh token: %w", err)
	}

	return newToken, nil
}

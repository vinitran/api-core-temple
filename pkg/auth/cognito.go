package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"api-core/pkg/jwtx"
)

const jwksFetchTimeout = 10 * time.Second

// CognitoVerifier verifies AWS Cognito RS256 tokens. It satisfies the same
// verifier contract as jwtx.HMAC, so the two are interchangeable in the
// authentication middleware.
type CognitoVerifier struct {
	issuer string
	jwks   JWKS
}

type cognitoClaims struct {
	jwtx.Claims
	TokenUse string `json:"token_use"`
}

// NewCognitoVerifier fetches the user pool's JWKS once and returns a verifier.
func NewCognitoVerifier(ctx context.Context, region, userPoolID string) (*CognitoVerifier, error) {
	issuer := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", region, userPoolID)

	ctx, cancel := context.WithTimeout(ctx, jwksFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, issuer+"/.well-known/jwks.json", nil)
	if err != nil {
		return nil, fmt.Errorf("cognito: build jwks request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cognito: fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cognito: fetch jwks: unexpected status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("cognito: decode jwks: %w", err)
	}

	return &CognitoVerifier{issuer: issuer, jwks: jwks}, nil
}

// Verify checks signature, expiry, issuer and token_use and returns the claims.
func (v *CognitoVerifier) Verify(tokenStr string) (*jwtx.Claims, error) {
	claims := &cognitoClaims{}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}))

	token, err := parser.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("token does not contain kid")
		}

		key, ok := v.jwks.Find(kid)
		if !ok {
			return nil, errors.New("matching jwk not found")
		}

		return key.RSAPublicKey()
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", jwtx.ErrInvalidToken, err)
	}
	if !token.Valid {
		return nil, jwtx.ErrInvalidToken
	}
	if !claims.VerifyIssuer(v.issuer, true) {
		return nil, fmt.Errorf("%w: unexpected issuer", jwtx.ErrInvalidToken)
	}
	if claims.TokenUse != "id" && claims.TokenUse != "access" {
		return nil, fmt.Errorf("%w: token_use should be id or access", jwtx.ErrInvalidToken)
	}

	return &claims.Claims, nil
}

// Package jwtx issues and verifies the application's own JSON Web Tokens.
package jwtx

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

const defaultTTL = time.Hour

// ErrInvalidToken is returned for any token that fails verification.
var ErrInvalidToken = errors.New("invalid token")

// Claims are the claims carried by access tokens.
type Claims struct {
	Email string `json:"email,omitempty"`
	jwt.RegisteredClaims
}

// Token is a signed token together with its expiry.
type Token struct {
	Value     string
	ExpiresAt time.Time
}

// HMACConfig configures an HMAC (HS256) token authority.
type HMACConfig struct {
	Secret   string
	Issuer   string
	Audience string
	TTL      time.Duration
}

// HMAC issues and verifies HS256 tokens with a shared secret.
type HMAC struct {
	secret   []byte
	issuer   string
	audience string
	ttl      time.Duration
	now      func() time.Time
}

// NewHMAC validates cfg and returns a token authority.
func NewHMAC(cfg HMACConfig) (*HMAC, error) {
	if cfg.Secret == "" {
		return nil, errors.New("jwtx: empty secret")
	}
	if cfg.Issuer == "" {
		return nil, errors.New("jwtx: empty issuer")
	}

	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = defaultTTL
	}

	return &HMAC{
		secret:   []byte(cfg.Secret),
		issuer:   cfg.Issuer,
		audience: cfg.Audience,
		ttl:      ttl,
		now:      time.Now,
	}, nil
}

// Issue signs a new access token for subject.
func (h *HMAC) Issue(subject, email string) (*Token, error) {
	if subject == "" {
		return nil, errors.New("jwtx: empty subject")
	}

	now := h.now()
	expiresAt := now.Add(h.ttl)
	claims := Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    h.issuer,
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}
	if h.audience != "" {
		claims.Audience = jwt.ClaimStrings{h.audience}
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(h.secret)
	if err != nil {
		return nil, fmt.Errorf("jwtx: sign token: %w", err)
	}

	return &Token{Value: signed, ExpiresAt: expiresAt}, nil
}

// Verify parses tokenStr, checks signature, algorithm, time-based claims,
// issuer and audience, and returns its claims.
func (h *HMAC) Verify(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	token, err := parser.ParseWithClaims(tokenStr, claims, func(*jwt.Token) (any, error) {
		return h.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	if !claims.VerifyIssuer(h.issuer, true) {
		return nil, fmt.Errorf("%w: unexpected issuer", ErrInvalidToken)
	}
	if h.audience != "" && !claims.VerifyAudience(h.audience, true) {
		return nil, fmt.Errorf("%w: unexpected audience", ErrInvalidToken)
	}
	if claims.Subject == "" {
		return nil, fmt.Errorf("%w: missing subject", ErrInvalidToken)
	}

	return claims, nil
}

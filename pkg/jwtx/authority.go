package jwtx

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type JWK struct {
	Alg   string `json:"alg"`
	Type  string `json:"kty"`
	Curve string `json:"crv"`
	ID    string `json:"kid"`
	X     string `json:"x"`
}

type JWTClaims struct {
	Email string `json:"email,omitempty"`
	jwt.RegisteredClaims
}

func NewJWTClaims(claims jwt.Claims) (*JWTClaims, error) {
	err := validateMapClaims(claims)
	if err != nil {
		return nil, err
	}
	claimsMapping := claims.(jwt.MapClaims)
	return &JWTClaims{
		Email: claimsMapping["email"].(string),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    claimsMapping["iss"].(string),
			Subject:   claimsMapping["sub"].(string),
			Audience:  jwt.ClaimStrings{claimsMapping["aud"].(string)},
			ExpiresAt: jwt.NewNumericDate(time.Unix(int64(claimsMapping["exp"].(float64)), 0)),
			//NotBefore: jwt.NewNumericDate(time.Unix(int64(claimsMapping["nbf"].(float64)), 0)),
			IssuedAt: jwt.NewNumericDate(time.Unix(int64(claimsMapping["iat"].(float64)), 0)),
			ID:       claimsMapping["jti"].(string),
		},
	}, nil
}

var (
	ErrUnableToParse = errors.New("unable to parse")
	ErrInvalidClaims = errors.New("invalid token claims")
)

type Authority struct {
	issuer     string
	expiration time.Duration

	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
}

func (g *Authority) kid() string {
	hash := sha256.Sum256(g.pub)
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

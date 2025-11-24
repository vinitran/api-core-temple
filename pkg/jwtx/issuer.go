package jwtx

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type HMACIssuer struct {
	secret     []byte
	issuer     string
	expiration time.Duration
}

func NewHMACIssuer(secret string, issuer string, expiration time.Duration) (*HMACIssuer, error) {
	if secret == "" {
		return nil, errors.New("jwt issuer: empty secret")
	}
	if issuer == "" {
		issuer = "otp-core"
	}
	if expiration <= 0 {
		expiration = time.Hour
	}
	return &HMACIssuer{
		secret:     []byte(secret),
		issuer:     issuer,
		expiration: expiration,
	}, nil
}

type Claims struct {
	Email string `json:"email,omitempty"`
	jwt.RegisteredClaims
}

func (i *HMACIssuer) Issue(subject string, email string) (string, error) {
	now := time.Now()
	claims := Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.issuer,
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(now.Add(i.expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(i.secret)
}

package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
)

// JWKS is a JSON Web Key Set.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK is a single RSA JSON Web Key.
type JWK struct {
	Alg string `json:"alg"`
	E   string `json:"e"`
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	Use string `json:"use"`
}

// Find returns the key with the given kid.
func (s JWKS) Find(kid string) (JWK, bool) {
	for _, key := range s.Keys {
		if key.Kid == kid {
			return key, true
		}
	}

	return JWK{}, false
}

// RSAPublicKey decodes the key's modulus and exponent.
func (k JWK) RSAPublicKey() (*rsa.PublicKey, error) {
	if k.N == "" || k.E == "" {
		return nil, errors.New("jwk n or e is missing")
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode jwk modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode jwk exponent: %w", err)
	}

	e := new(big.Int).SetBytes(eBytes)
	if !e.IsInt64() || e.Int64() <= 1 || e.Int64() > 1<<31-1 {
		return nil, errors.New("jwk exponent out of range")
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(e.Int64()),
	}, nil
}

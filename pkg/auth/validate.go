package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"math/big"
	"time"
)

func GetRSAPublicKey(jwk CognitoKeys) (*rsa.PublicKey, error) {
	if jwk.N == "" || jwk.E == "" {
		return nil, errors.New("jwk n or e is missing")
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %s", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %s", err)
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}, nil
}

func VerifyToken(cognitoJWKs CognitoJWKs, tokenString, region, userPoolID string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("token does not contain kid")
		}

		for _, key := range cognitoJWKs.Keys {
			if key.Kid == kid {
				return GetRSAPublicKey(key)
			}
		}

		return nil, errors.New("matching JWK not found")
	})

	if err != nil {
		return nil, err
	}

	// Validate claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if err = ValidateExpired(claims); err != nil {
			return nil, err
		}

		if err = ValidateTokenUse(claims); err != nil {
			return nil, err
		}

		if err = ValidateIss(region, userPoolID, claims); err != nil {
			return nil, err
		}

		return token, nil
	}

	return nil, errors.New("invalid token")
}

func ValidateTokenUse(claims jwt.MapClaims) error {
	if tokenUse, ok := claims["token_use"]; ok {
		if tokenUseStr, ok := tokenUse.(string); ok {
			if tokenUseStr == "id" || tokenUseStr == "access" {
				return nil
			}
		}
	}
	return errors.New("token_use should be id or access")
}

func ValidateExpired(claims jwt.MapClaims) error {
	if exp, ok := claims["exp"].(float64); ok {
		now := time.Now().Unix()
		if int64(exp) < now {
			return errors.New("token is expired")
		}
		return nil
	}
	return errors.New("cannot parse token exp")
}

func ValidateIss(region, userPoolID string, claims jwt.MapClaims) error {
	issShoudBe := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", region, userPoolID)
	if claims["iss"] != issShoudBe {
		return errors.New("invalid issuer")
	}
	return nil
}

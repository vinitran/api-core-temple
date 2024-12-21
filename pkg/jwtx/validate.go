package jwtx

import (
	"fmt"
	"github.com/golang-jwt/jwt/v4"
)

func validateMapClaims(claims jwt.Claims) error {
	claimsMapping, ok := claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("invalid type for claims mapping")
	}

	requiredClaims := []string{"iss", "sub", "aud", "exp", "iat", "jti"}
	for _, claim := range requiredClaims {
		if _, ok := claimsMapping[claim]; !ok {
			return fmt.Errorf("missing required claim: %s", claim)
		}
	}

	// Convert and validate claim types
	_, ok = claimsMapping["iss"].(string)
	if !ok {
		return fmt.Errorf("invalid type for claim: iss")
	}
	_, ok = claimsMapping["sub"].(string)
	if !ok {
		return fmt.Errorf("invalid type for claim: sub")
	}
	_, ok = claimsMapping["aud"].(string)
	if !ok {
		return fmt.Errorf("invalid type for claim: aud")
	}
	_, ok = claimsMapping["exp"].(float64)
	if !ok {
		return fmt.Errorf("invalid type for claim: exp")
	}
	_, ok = claimsMapping["iat"].(float64)
	if !ok {
		return fmt.Errorf("invalid type for claim: iat")
	}
	_, ok = claimsMapping["jti"].(string)
	if !ok {
		return fmt.Errorf("invalid type for claim: jti")
	}

	return nil
}

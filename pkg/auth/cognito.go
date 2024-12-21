package auth

import (
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"io"
	"net/http"
)

type CognitoJWKs struct {
	Keys []CognitoKeys `json:"keys"`
}

type CognitoKeys struct {
	Alg string `json:"alg"`
	E   string `json:"e"`
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	Use string `json:"use"`
}

type AuthnCognito struct {
	UserPoolId  string
	Region      string
	CognitoJWKs CognitoJWKs
}

func NewAuthnCognito(userPoolID, region string) (*AuthnCognito, error) {
	var cognitoJWKs CognitoJWKs

	// Fetch the Cognito JWKs once at the start
	cognitoURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", region, userPoolID)
	resp, err := http.Get(cognitoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get Cognito JWKs: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body Cognito JWKs: %s", err)
	}

	err = json.Unmarshal(body, &cognitoJWKs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWKs: %s", err)
	}

	return &AuthnCognito{userPoolID, region, cognitoJWKs}, nil
}

func (authn *AuthnCognito) AuthenticateJWT(tokenStr string) (*jwt.Token, error) {
	return VerifyToken(authn.CognitoJWKs, tokenStr, authn.Region, authn.UserPoolId)
}

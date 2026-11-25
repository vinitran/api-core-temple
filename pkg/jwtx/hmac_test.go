package jwtx

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const testSecret = "test-secret-at-least-32-bytes-long!!"

func newTestHMAC(t *testing.T, cfg HMACConfig) *HMAC {
	t.Helper()

	if cfg.Secret == "" {
		cfg.Secret = testSecret
	}
	if cfg.Issuer == "" {
		cfg.Issuer = "api-core"
	}

	h, err := NewHMAC(cfg)
	if err != nil {
		t.Fatalf("NewHMAC() error = %v", err)
	}

	return h
}

func TestHMACRoundTrip(t *testing.T) {
	t.Parallel()

	h := newTestHMAC(t, HMACConfig{Audience: "api-core-clients"})

	tok, err := h.Issue("42", "a@example.com")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	claims, err := h.Verify(tok.Value)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if claims.Subject != "42" || claims.Email != "a@example.com" {
		t.Errorf("claims = %+v, want subject 42 and email a@example.com", claims)
	}
}

func TestHMACVerifyRejects(t *testing.T) {
	t.Parallel()

	h := newTestHMAC(t, HMACConfig{Audience: "aud"})
	valid, err := h.Issue("1", "")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	expired := newTestHMAC(t, HMACConfig{Audience: "aud"})
	expired.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	expiredTok, err := expired.Issue("1", "")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	otherIssuer := newTestHMAC(t, HMACConfig{Issuer: "someone-else", Audience: "aud"})
	otherIssuerTok, err := otherIssuer.Issue("1", "")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	otherSecret := newTestHMAC(t, HMACConfig{Secret: "another-secret-that-is-long-enough!!", Audience: "aud"})
	otherSecretTok, err := otherSecret.Issue("1", "")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	noneTok, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Issuer: "api-core", Subject: "1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none token: %v", err)
	}

	tests := map[string]string{
		"expired":       expiredTok.Value,
		"wrong issuer":  otherIssuerTok.Value,
		"wrong secret":  otherSecretTok.Value,
		"alg none":      noneTok,
		"garbage":       "not-a-jwt",
		"tampered body": valid.Value[:len(valid.Value)-2] + "xx",
	}

	for name, tok := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := h.Verify(tok); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("Verify() error = %v, want ErrInvalidToken", err)
			}
		})
	}
}

func TestNewHMACValidation(t *testing.T) {
	t.Parallel()

	if _, err := NewHMAC(HMACConfig{Issuer: "x"}); err == nil {
		t.Error("expected error for empty secret")
	}
	if _, err := NewHMAC(HMACConfig{Secret: "x"}); err == nil {
		t.Error("expected error for empty issuer")
	}
}

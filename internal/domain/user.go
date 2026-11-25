// Package domain holds the business entities shared across layers. It has no
// dependency on storage, transport or third-party SDKs.
package domain

import "time"

// User is an application user.
type User struct {
	ID            int64
	GoogleID      string
	Email         string
	Name          *string
	Picture       *string
	Locale        *string
	VerifiedEmail bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastLoginAt   *time.Time
}

// GoogleAccount is the identity Google returns after a successful sign-in.
type GoogleAccount struct {
	GoogleID      string
	Email         string
	Name          *string
	Picture       *string
	Locale        *string
	VerifiedEmail bool
	LoginAt       time.Time
}

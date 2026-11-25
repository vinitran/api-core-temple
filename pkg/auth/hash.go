package auth

import "golang.org/x/crypto/bcrypt"

// bcryptCost balances security and latency (~250ms on modern hardware).
// Cost 14 took ~1s per hash, which makes login endpoints trivially DoS-able.
const bcryptCost = 12

func Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(bytes), err
}

func CheckPasswordHash(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

package auth

import "golang.org/x/crypto/bcrypt"

// hashPassword hashes a raw password string using bcrypt
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// comparePassword compares a hashed password with a raw password string
func comparePassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// tokenType distinguishes access tokens from refresh tokens
type tokenType string

const (
	tokenTypeAccess  tokenType = "access"
	tokenTypeRefresh tokenType = "refresh"
)

// claims represents the claims in a JWT token
type claims struct {
	UserID uuid.UUID `json:"userID"`
	Role   string    `json:"role"`
	Type   tokenType `json:"type"`
	jwt.RegisteredClaims
}

// generateToken generates a new JWT token of the given type for a user
func generateToken(userID uuid.UUID, role string, typ tokenType, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	c := &claims{
		UserID: userID,
		Role:   role,
		Type:   typ,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString([]byte(secret))
}

// validateToken validates a JWT token, ensures it is of the expected type and returns the claims
func validateToken(tokenString string, secret string, expected tokenType) (*claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}

	c, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	if c.Type != expected {
		return nil, errors.New("invalid token type")
	}

	return c, nil
}

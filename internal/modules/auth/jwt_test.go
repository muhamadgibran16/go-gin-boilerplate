package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

const testSecret = "test-secret-with-at-least-32-characters"

func TestValidateTokenRejectsWrongType(t *testing.T) {
	userID := uuid.New()

	refresh, err := generateToken(userID, "user", tokenTypeRefresh, testSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validateToken(refresh, testSecret, tokenTypeAccess); err == nil {
		t.Fatal("refresh token must not be accepted as access token")
	}

	access, err := generateToken(userID, "user", tokenTypeAccess, testSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validateToken(access, testSecret, tokenTypeRefresh); err == nil {
		t.Fatal("access token must not be accepted as refresh token")
	}

	claims, err := validateToken(access, testSecret, tokenTypeAccess)
	if err != nil {
		t.Fatalf("valid access token rejected: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("got user %s, want %s", claims.UserID, userID)
	}
}

func TestValidateTokenRejectsExpiredAndWrongSecret(t *testing.T) {
	expired, _ := generateToken(uuid.New(), "user", tokenTypeAccess, testSecret, -time.Minute)
	if _, err := validateToken(expired, testSecret, tokenTypeAccess); err == nil {
		t.Fatal("expired token must be rejected")
	}

	token, _ := generateToken(uuid.New(), "user", tokenTypeAccess, testSecret, time.Hour)
	if _, err := validateToken(token, "another-secret-with-at-least-32-chars", tokenTypeAccess); err == nil {
		t.Fatal("token signed with a different secret must be rejected")
	}
}

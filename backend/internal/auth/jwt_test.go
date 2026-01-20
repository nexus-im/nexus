package auth

import (
	"testing"
	"time"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "super-secret-key"
	issuer := "nexus-im"
	validity := time.Hour
	auth := NewAuthenticator(secret, issuer, validity)

	userID := "user-123"
	username := "testuser"

	// Generate Token
	token, err := auth.GenerateToken(userID, username)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	if token == "" {
		t.Fatal("generated token is empty")
	}

	// Validate Token
	claims, err := auth.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected user ID %s, got %s", userID, claims.UserID)
	}
	if claims.Username != username {
		t.Errorf("expected username %s, got %s", username, claims.Username)
	}
	if claims.Issuer != issuer {
		t.Errorf("expected issuer %s, got %s", issuer, claims.Issuer)
	}
}

func TestExpiredToken(t *testing.T) {
	secret := "super-secret-key"
	auth := NewAuthenticator(secret, "nexus", -time.Minute) // Expired immediately

	token, err := auth.GenerateToken("u1", "user")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = auth.ValidateToken(token)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestInvalidSignature(t *testing.T) {
	auth1 := NewAuthenticator("secret1", "nexus", time.Hour)
	auth2 := NewAuthenticator("secret2", "nexus", time.Hour)

	token, _ := auth1.GenerateToken("u1", "user")

	_, err := auth2.ValidateToken(token)
	if err == nil {
		t.Fatal("expected error for invalid signature, got nil")
	}
}

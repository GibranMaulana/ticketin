package utils

import (
	"net/http"
	"os"
	"testing"
	"uuid"
)

func TestGetCookieConfig(t *testing.T) {
	// Dev environment test
	os.Setenv("APP_ENV", "dev")
	secure, sameSite := getCookieConfig()
	if secure != false {
		t.Errorf("expected secure=false in dev, got %v", secure)
	}
	if sameSite != http.SameSiteLaxMode {
		t.Errorf("expected sameSite=Lax in dev, got %v", sameSite)
	}

	// Production environment test (Vercel cross-site cookies)
	os.Setenv("APP_ENV", "production")
	secure, sameSite = getCookieConfig()
	if secure != true {
		t.Errorf("expected secure=true in production, got %v", secure)
	}
	if sameSite != http.SameSiteNoneMode {
		t.Errorf("expected sameSite=None in production, got %v", sameSite)
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	os.Setenv("JWT_SECRET", "super-secret-key-1234567890123456")

	userID := uuid.New()
	email := "test@example.com"

	token, err := GenerateAccessToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := VerifyAccessToken(token)
	if err != nil {
		t.Fatalf("VerifyAccessToken failed: %v", err)
	}

	if claims.ID != userID {
		t.Errorf("expected userID %s, got %s", userID, claims.ID)
	}
	if claims.Email != email {
		t.Errorf("expected email %s, got %s", email, claims.Email)
	}
}

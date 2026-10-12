package services

import (
	"os"
	"testing"
)

func TestNewEmailService(t *testing.T) {
	os.Setenv("EMAIL_FROM", "test@ticketin.com")
	os.Setenv("EMAIL_PROVIDER", "resend")
	os.Setenv("RESEND_API_KEY", "re_123456789")

	service := NewEmailService()
	if service.From != "test@ticketin.com" {
		t.Errorf("expected from 'test@ticketin.com', got '%s'", service.From)
	}
	if service.Provider != "resend" {
		t.Errorf("expected provider 'resend', got '%s'", service.Provider)
	}
	if service.ResendAPIKey != "re_123456789" {
		t.Errorf("expected resend API key 're_123456789', got '%s'", service.ResendAPIKey)
	}
}

func TestEmailServiceUnsupportedProvider(t *testing.T) {
	service := &EmailService{
		Provider: "unknown_provider",
	}

	err := service.SendOTP("test@example.com", "123456")
	if err == nil {
		t.Fatal("expected error for unsupported provider, got nil")
	}
}

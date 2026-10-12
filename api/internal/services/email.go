package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"time"
)

type EmailService struct {
	From         string
	Provider     string
	Host         string
	Port         string
	ResendAPIKey string
}

func NewEmailService() *EmailService {
	return &EmailService{
		From:         os.Getenv("EMAIL_FROM"),
		Provider:     os.Getenv("EMAIL_PROVIDER"),
		Host:         os.Getenv("SMTP_HOST"),
		Port:         os.Getenv("SMTP_PORT"),
		ResendAPIKey: os.Getenv("RESEND_API_KEY"),
	}
}

func (s *EmailService) SendOTP(toEmail string, otpCode string) error {
	subject := "Your Ticketin Account Verification Code"

	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<head></head>
		<body>
			<div class="container" style="font-family: sans-serif; max-width: 500px; margin: auto; padding: 20px; border: 1px solid #e2e8f0; border-radius: 8px;">
				<h2 style="color: #1e293b;">Verify Your Email</h2>
				<p style="color: #475569;">Use the verification code below to complete your authentication. This code will expire in <strong>5 minutes</strong>.</p>
				<div style="font-size: 32px; font-weight: bold; letter-spacing: 6px; color: #4f46e5; padding: 16px; background: #f1f5f9; text-align: center; border-radius: 6px; margin: 24px 0;">%s</div>
				<p style="color: #64748b; font-size: 14px;">If you didn't request this code, you can safely ignore this email.</p>
				<div style="color: #94a3b8; font-size: 12px; margin-top: 24px; border-top: 1px solid #e2e8f0; padding-top: 12px;">&copy; Ticketin. All rights reserved.</div>
			</div>
		</body>
	`, otpCode)

	switch s.Provider {
	case "resend":
		return s.SendResend(toEmail, subject, htmlBody)
	case "smtp":
		return s.SendSMTP(toEmail, subject, htmlBody)
	default:
		return fmt.Errorf("email provider '%s' is not supported", s.Provider)
	}
}

func (s *EmailService) SendResend(toEmail, subject, htmlBody string) error {
	if s.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY is not set")
	}

	payload := map[string]interface{}{
		"from":    s.From,
		"to":      []string{toEmail},
		"subject": subject,
		"html":    htmlBody,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Resend payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create Resend request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email via Resend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API returned error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (s *EmailService) SendSMTP(toEmail, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%s", s.Host, s.Port)

	sender, err := mail.ParseAddress(s.From)
	fromAddress := s.From
	if err == nil {
		fromAddress = sender.Address
	}

	var auth smtp.Auth
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASSWORD")
	if user != "" && pass != "" {
		auth = smtp.PlainAuth("", user, pass, s.Host)
	}

	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		s.From, toEmail, subject)

	msg := []byte(headers + htmlBody)

	return smtp.SendMail(addr, auth, fromAddress, []string{toEmail}, msg)
}


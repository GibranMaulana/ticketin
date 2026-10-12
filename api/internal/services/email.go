package services

import (
	"fmt"
	"net/mail"
	"net/smtp"
	"os"
)

type EmailService struct {
	From     string
	Provider string
	Host     string
	Port     string
}

func NewEmailService() *EmailService {
	return &EmailService{
		From:     os.Getenv("EMAIL_FROM"),
		Provider: os.Getenv("EMAIL_PROVIDER"),
		Host:     os.Getenv("SMTP_HOST"),
		Port:     os.Getenv("SMTP_PORT"),
	}
}

func (s *EmailService) SendOTP(toEmail string, otpCode string) error {
	subject := "Your Ticketin Account Verification Code"

	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<head></head>
		<body>
			<div class="container">
				<h2>Verify Your Email</h2>
				<p>Use the verification code below to complete your registration. This code will expire in <strong>5 minutes</strong>.</p>
				<div class="otp-box">%s</div>
				<p>If you didn't request this code, please ignore this email.</p>
				<div class="footer">&copy; Ticketin. All rights reserved.</div>
			</div>
		</body>
	`, otpCode)

	if s.Provider == "smtp" {
		return s.SendSMTP(toEmail, subject, htmlBody)
	}

	return fmt.Errorf("Provider currently not supported")
}

func (s *EmailService) SendSMTP(toEmail, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%s", s.Host, s.Port)

	sender, err := mail.ParseAddress(s.From)

	fromAddress := s.From
	if err == nil {
		fromAddress = sender.Address
	}

	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		s.From, toEmail, subject)

	msg := []byte(headers + htmlBody)

	return smtp.SendMail(addr, nil, fromAddress, []string{toEmail}, msg)
}

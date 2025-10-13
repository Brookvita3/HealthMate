package email

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
)

type GmailEmailService struct {
	host        string
	port        int
	username    string
	appPassword string
	senderName  string
}

func NewGmailEmailService(host string, port int, username, appPassword, senderName string) EmailService {
	return &GmailEmailService{
		host:        host,
		port:        port,
		username:    username,
		appPassword: appPassword,
		senderName:  senderName,
	}
}

func (s *GmailEmailService) SendOTP(ctx context.Context, recipientEmail, recipientName, otp string) error {

	from := fmt.Sprintf("%s <%s>", s.senderName, s.username)
	to := recipientEmail

	subject := "Your HealthMate Verification Code"

	body := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n" +
		"MIME-version: 1.0;\n" +
		"Content-Type: text/html; charset=\"UTF-8\";\n\n" +
		fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; line-height: 1.6;">
			<h3>Hello %s,</h3>
			<p>Thank you for registering with HealthMate. Please use the following verification code to complete your registration:</p>
			<p style="font-size: 24px; font-weight: bold; color: #1a73e8; letter-spacing: 2px;">%s</p>
			<p>This code is valid for 5 minutes.</p>
			<p>If you did not request this, please ignore this email.</p>
			<br>
			<p>Thanks,<br>The HealthMate Team</p>
		</div>
		`, recipientName, otp)

	auth := smtp.PlainAuth("", s.username, s.appPassword, s.host)

	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	err := smtp.SendMail(addr, auth, s.username, []string{to}, []byte(body))

	if err != nil {
		log.Printf("Gmail SMTP: Failed to send OTP email to %s. Error: %v", recipientEmail, err)
		return err
	}

	return nil
}

func (s *GmailEmailService) ResendOTP(ctx context.Context, recipientEmail, recipientName, otp string) error {

	from := fmt.Sprintf("%s <%s>", s.senderName, s.username)
	to := recipientEmail

	subject := "Your HealthMate Verification Code"

	body := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n" +
		"MIME-version: 1.0;\n" +
		"Content-Type: text/html; charset=\"UTF-8\";\n\n" +
		fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; line-height: 1.6;">
			<h3>Hello %s,</h3>
			<p>We received a request to resend your verification code. Please use the following code to complete your registration:</p>
			<p style="font-size: 24px; font-weight: bold; color: #1a73e8; letter-spacing: 2px;">%s</p>
			<p>This code is valid for 5 minutes. If you did not request this, please ignore this email.</p>
			<br>
			<p>Thanks,<br>The HealthMate Team</p>
		</div>
		`, recipientName, otp)

	auth := smtp.PlainAuth("", s.username, s.appPassword, s.host)

	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	err := smtp.SendMail(addr, auth, s.username, []string{to}, []byte(body))

	if err != nil {
		log.Printf("Gmail SMTP: Failed to send OTP email to %s. Error: %v", recipientEmail, err)
		return err // Trả về lỗi để có thể debug
	}

	return nil
}

func (s *GmailEmailService) SendWelcomeEmail(ctx context.Context, recipientEmail, recipientName string) error {
	from := fmt.Sprintf("%s <%s>", s.senderName, s.username)
	to := recipientEmail
	subject := "Welcome to HealthMate!"

	body := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n" +
		"MIME-version: 1.0;\n" +
		"Content-Type: text/html; charset=\"UTF-8\";\n\n" +
		fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; line-height: 1.6;">
			<h2 style="color: #4CAF50;">Welcome, %s!</h2>
			<p>We arae excited to have you join the HealthMate community. Your account has been successfully verified and is now active.</p>
			<p>You can now log in and start exploring all the features.</p>
			<br>
			<p>Best regards,<br>The HealthMate Team</p>
		</div>
		`, recipientName)

	auth := smtp.PlainAuth("", s.username, s.appPassword, s.host)
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	err := smtp.SendMail(addr, auth, s.username, []string{to}, []byte(body))

	if err != nil {
		log.Printf("Gmail SMTP: Failed to send WELCOME email to %s. Error: %v", recipientEmail, err)
		return err
	}

	return nil
}

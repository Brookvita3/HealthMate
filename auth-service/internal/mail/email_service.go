package email

import "context"

type EmailService interface {
	SendOTP(ctx context.Context, recipientEmail, recipientName, otp string) error
	ResendOTP(ctx context.Context, recipientEmail, recipientName, otp string) error
	SendWelcomeEmail(ctx context.Context, recipientEmail, recipientName string) error
}

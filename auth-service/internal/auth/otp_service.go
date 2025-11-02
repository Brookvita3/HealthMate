package auth

import (
	"context"
	"healthmate/internal/domain"
)

type OTPService interface {
	Generate(ctx context.Context, key string) (string, error)
	GetRecord(ctx context.Context, key string) (*domain.Record, error)
	Verify(ctx context.Context, key, inputOTP string) error
	Delete(ctx context.Context, key string) error
}

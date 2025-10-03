package auth

import "context"

type Record struct {
	Hash         string `json:"hash"`
	AttemptsLeft int    `json:"attempts_left"`
	CreatedAt    int64  `json:"created_at"`
}

type OTPService interface {
	Generate(ctx context.Context, key string) (string, error)
	GetRecord(ctx context.Context, key string) (*Record, error)
	Verify(ctx context.Context, key, inputOTP string) error
	Delete(ctx context.Context, key string) error
}

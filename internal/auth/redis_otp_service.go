package auth

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"healthmate/internal/cache"
	"io"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type RedisOTPService struct {
	cache cache.Cache
}

const (
	otpLength         = 6
	otpMaxAttempts    = 5
	otpExpiration     = 5 * time.Minute
	otpRedisKeyPrefix = "otp:"
)

func NewRedisOTPService(cache cache.Cache) *RedisOTPService {
	return &RedisOTPService{
		cache: cache,
	}
}

func (s *RedisOTPService) generateOTPString() (string, error) {
	buffer := make([]byte, otpLength)
	_, err := io.ReadFull(rand.Reader, buffer)
	if err != nil {
		return "", err
	}

	for i := 0; i < otpLength; i++ {
		buffer[i] = (buffer[i] % 10) + '0'
	}
	return string(buffer), nil
}

func (s *RedisOTPService) Generate(ctx context.Context, key string) (string, error) {
	otp, err := s.generateOTPString()
	if err != nil {
		return "", fmt.Errorf("failed to generate otp string: %w", err)
	}

	hashedOTP, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash otp: %w", err)
	}

	rec := Record{
		Hash:         string(hashedOTP),
		AttemptsLeft: otpMaxAttempts,
		CreatedAt:    time.Now().Unix(),
	}

	recordJSON, err := json.Marshal(rec)
	if err != nil {
		return "", fmt.Errorf("failed to marshal otp record: %w", err)
	}

	redisKey := otpRedisKeyPrefix + key
	if err := s.cache.Set(ctx, redisKey, recordJSON, otpExpiration); err != nil {
		return "", fmt.Errorf("failed to set otp in redis: %w", err)
	}

	return otp, nil
}

func (s *RedisOTPService) GetRecord(ctx context.Context, key string) (*Record, error) {
	redisKey := otpRedisKeyPrefix + key
	recordJSON, err := s.cache.Get(ctx, redisKey)
	if err != nil {
		if err == redis.Nil {
			return nil, ErrOTPNotFound
		}
		return nil, fmt.Errorf("failed to get otp from redis: %w", err)
	}

	var rec Record
	if err := json.Unmarshal([]byte(recordJSON), &rec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal otp record: %w", err)
	}

	return &rec, nil
}

func (s *RedisOTPService) Verify(ctx context.Context, key, inputOTP string) error {
	rec, err := s.GetRecord(ctx, key)
	if err != nil {
		return err
	}

	if rec.AttemptsLeft <= 0 {
		return ErrTooManyOTPAttempts
	}

	err = bcrypt.CompareHashAndPassword([]byte(rec.Hash), []byte(inputOTP))
	if err != nil {
		rec.AttemptsLeft--

		recordJSON, _ := json.Marshal(rec)

		ttl := otpExpiration - time.Since(time.Unix(rec.CreatedAt, 0))

		redisKey := otpRedisKeyPrefix + key
		s.cache.Set(ctx, redisKey, recordJSON, ttl)

		return ErrInvalidOTP
	}

	return nil
}

func (s *RedisOTPService) Delete(ctx context.Context, key string) error {
	redisKey := otpRedisKeyPrefix + key
	return s.cache.Delete(ctx, redisKey)
}

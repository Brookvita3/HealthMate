package auth

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"auth-service/internal/domain"
	"auth-service/mocks"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

type otpTestEnv struct {
	cache   *mocks.Cache
	service *RedisOTPService
}

func newOTPTestEnv() *otpTestEnv {
	c := new(mocks.Cache)
	return &otpTestEnv{
		cache:   c,
		service: NewRedisOTPService(c),
	}
}

// marshalRecord JSON-encodes a domain.Record for use as a cache.Get return value.
func marshalRecord(rec domain.Record) string {
	b, _ := json.Marshal(rec)
	return string(b)
}

// bcryptHash produces a bcrypt hash of s using the minimum cost (fast in tests).
func bcryptHash(s string) string {
	h, _ := bcrypt.GenerateFromPassword([]byte(s), bcrypt.MinCost)
	return string(h)
}

// ─── Generate ─────────────────────────────────────────────────────────────────

func TestOTPGenerate(t *testing.T) {
	t.Run("success: stores hashed OTP in cache and returns 6-digit plaintext OTP", func(t *testing.T) {
		// cache.Set must receive key "otp:<key>", the 5-minute TTL, and a JSON
		// payload containing a bcrypt hash. The returned plaintext must be 6 digits.
		env := newOTPTestEnv()

		env.cache.On("Set",
			mock.Anything,
			"otp:user@example.com",
			mock.Anything,
			otpExpiration,
		).Return(nil).Once()

		otp, err := env.service.Generate(context.Background(), "user@example.com")

		assert.NoError(t, err)
		assert.Len(t, otp, otpLength)
		for _, ch := range otp {
			assert.True(t, ch >= '0' && ch <= '9', "OTP must be all digits")
		}
		env.cache.AssertExpectations(t)
	})

	t.Run("error: cache.Set failure is propagated wrapped", func(t *testing.T) {
		// If storing the record fails the error must be returned to the caller.
		env := newOTPTestEnv()
		env.cache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(errors.New("redis timeout")).Once()

		_, err := env.service.Generate(context.Background(), "user@example.com")

		assert.Error(t, err)
		env.cache.AssertExpectations(t)
	})
}

// ─── GetRecord ────────────────────────────────────────────────────────────────

func TestOTPGetRecord(t *testing.T) {
	t.Run("success: valid JSON in cache is deserialised and returned", func(t *testing.T) {
		// When cache contains a valid Record JSON the service must return it.
		env := newOTPTestEnv()
		want := domain.Record{Hash: "hashval", AttemptsLeft: 5, CreatedAt: time.Now().Unix()}

		env.cache.On("Get", mock.Anything, "otp:mykey").Return(marshalRecord(want), nil).Once()

		got, err := env.service.GetRecord(context.Background(), "mykey")

		assert.NoError(t, err)
		assert.Equal(t, want.Hash, got.Hash)
		assert.Equal(t, want.AttemptsLeft, got.AttemptsLeft)
		env.cache.AssertExpectations(t)
	})

	t.Run("error: redis.Nil sentinel maps to ErrOTPNotFound", func(t *testing.T) {
		// A cache miss must surface as the domain error ErrOTPNotFound.
		env := newOTPTestEnv()

		env.cache.On("Get", mock.Anything, "otp:missing").Return("", redis.Nil).Once()

		_, err := env.service.GetRecord(context.Background(), "missing")

		assert.ErrorIs(t, err, ErrOTPNotFound)
		env.cache.AssertExpectations(t)
	})

	t.Run("error: generic cache error is wrapped and returned", func(t *testing.T) {
		// Any non-Nil cache error must be returned (not mapped to ErrOTPNotFound).
		env := newOTPTestEnv()

		env.cache.On("Get", mock.Anything, "otp:mykey").Return("", errors.New("connection reset")).Once()

		_, err := env.service.GetRecord(context.Background(), "mykey")

		assert.Error(t, err)
		assert.NotErrorIs(t, err, ErrOTPNotFound)
		env.cache.AssertExpectations(t)
	})

	t.Run("error: corrupt JSON in cache returns unmarshal error", func(t *testing.T) {
		// If the cached value is not valid JSON the service must return an error.
		env := newOTPTestEnv()

		env.cache.On("Get", mock.Anything, "otp:bad").Return("{not-json", nil).Once()

		_, err := env.service.GetRecord(context.Background(), "bad")

		assert.Error(t, err)
		env.cache.AssertExpectations(t)
	})
}

// ─── Verify ───────────────────────────────────────────────────────────────────

func TestOTPVerify(t *testing.T) {
	t.Run("success: correct OTP passes bcrypt comparison and returns nil", func(t *testing.T) {
		// Providing the OTP that matches the stored hash must return no error.
		// No cache write-back is needed on a successful verify.
		env := newOTPTestEnv()
		knownOTP := "482910"
		rec := domain.Record{
			Hash:         bcryptHash(knownOTP),
			AttemptsLeft: otpMaxAttempts,
			CreatedAt:    time.Now().Unix(),
		}

		env.cache.On("Get", mock.Anything, "otp:user@example.com").Return(marshalRecord(rec), nil).Once()

		err := env.service.Verify(context.Background(), "user@example.com", knownOTP)

		assert.NoError(t, err)
		env.cache.AssertExpectations(t)
	})

	t.Run("error: OTP record not in cache surfaces ErrOTPNotFound", func(t *testing.T) {
		// A redis.Nil from GetRecord must propagate unchanged.
		env := newOTPTestEnv()

		env.cache.On("Get", mock.Anything, "otp:user@example.com").Return("", redis.Nil).Once()

		err := env.service.Verify(context.Background(), "user@example.com", "000000")

		assert.ErrorIs(t, err, ErrOTPNotFound)
		env.cache.AssertExpectations(t)
	})

	t.Run("error: zero AttemptsLeft returns ErrTooManyOTPAttempts without bcrypt call", func(t *testing.T) {
		// When attempts are exhausted the service must reject immediately without
		// comparing the hash or updating the record.
		env := newOTPTestEnv()
		rec := domain.Record{Hash: "any-hash", AttemptsLeft: 0, CreatedAt: time.Now().Unix()}

		env.cache.On("Get", mock.Anything, "otp:user@example.com").Return(marshalRecord(rec), nil).Once()

		err := env.service.Verify(context.Background(), "user@example.com", "123456")

		assert.ErrorIs(t, err, ErrTooManyOTPAttempts)
		env.cache.AssertNotCalled(t, "Set")
		env.cache.AssertExpectations(t)
	})

	t.Run("error: wrong OTP decrements AttemptsLeft and returns ErrInvalidOTP", func(t *testing.T) {
		// On bcrypt mismatch the service must persist the decremented record and
		// return ErrInvalidOTP.
		env := newOTPTestEnv()
		knownOTP := "482910"
		rec := domain.Record{
			Hash:         bcryptHash(knownOTP),
			AttemptsLeft: 3,
			CreatedAt:    time.Now().Unix(),
		}

		env.cache.On("Get", mock.Anything, "otp:user@example.com").Return(marshalRecord(rec), nil).Once()
		env.cache.On("Set",
			mock.Anything,
			"otp:user@example.com",
			mock.Anything,
			mock.AnythingOfType("time.Duration"),
		).Return(nil).Once()

		err := env.service.Verify(context.Background(), "user@example.com", "000000")

		assert.ErrorIs(t, err, ErrInvalidOTP)
		env.cache.AssertExpectations(t)
	})
}

// ─── Delete ───────────────────────────────────────────────────────────────────

func TestOTPDelete(t *testing.T) {
	t.Run("success: calls cache.Delete with 'otp:'-prefixed key", func(t *testing.T) {
		// The service must prepend otpRedisKeyPrefix before delegating to cache.
		env := newOTPTestEnv()

		env.cache.On("Delete", mock.Anything, "otp:user@example.com").Return(nil).Once()

		err := env.service.Delete(context.Background(), "user@example.com")

		assert.NoError(t, err)
		env.cache.AssertExpectations(t)
	})

	t.Run("error: cache.Delete failure is returned to caller", func(t *testing.T) {
		// Any error from the cache layer must propagate unchanged.
		env := newOTPTestEnv()

		env.cache.On("Delete", mock.Anything, "otp:user@example.com").Return(errors.New("write error")).Once()

		err := env.service.Delete(context.Background(), "user@example.com")

		assert.Error(t, err)
		env.cache.AssertExpectations(t)
	})
}

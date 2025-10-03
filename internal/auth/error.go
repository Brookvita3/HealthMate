package auth

import (
	"healthmate/internal/common"
)

var (

	// ErrOTPNotFound is returned when an OTP is not found in the repository.
	ErrOTPNotFound = &common.BusinessError{Code: 404, Message: "OTP not found or expired"}

	// ErrInvalidOTP is returned when an OTP is invalid.
	ErrInvalidOTP = &common.BusinessError{Code: 400, Message: "invalid OTP"}

	// ErrTooManyOTPAttempts is returned when the user has exceeded the maximum number of OTP attempts.
	ErrTooManyOTPAttempts = &common.BusinessError{Code: 429, Message: "too many OTP attempts"}

	// ErrAccountNotVerified is returned when a user tries to log in before verifying their account by OTP through email.
	ErrAccountNotVerified = &common.BusinessError{Code: 403, Message: "account not verified"}

	// ErrAccountAlreadyVerified is returned when a user tries to verify their account more than once.
	ErrAccountAlreadyVerified = &common.BusinessError{Code: 400, Message: "account already verified"}
)

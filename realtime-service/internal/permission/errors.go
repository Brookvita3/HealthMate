package permission

import "realtime-service/internal/common"

var (
	// ErrSelfSubscription is returned when an observer tries to subscribe to themselves.
	ErrSelfSubscription = &common.BusinessError{Code: 400, Message: "observerUserID and targetUserID cannot be the same"}
	
	// ErrPermissionCheckFailed is returned when an unexpected error occurs while checking permissions.
	ErrPermissionCheckFailed = &common.BusinessError{Code: 500, Message: "internal error when checking permission"}
)

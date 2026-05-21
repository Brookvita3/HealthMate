package permission

import (
	"auth-service/internal/common"
)

var (
	// ErrPermissionDenied is returned when a user does not have permission to perform an action.
	ErrPermissionDenied = &common.BusinessError{Code: 403, Message: "permission denied"}

	// ErrInvalidMetricType is returned when an unsupported metric type is provided.
	ErrInvalidMetricType = &common.BusinessError{Code: 400, Message: "invalid metric type"}

	// ErrUsePutForPerMemberSharing is returned when POST is used for per-viewer rules.
	ErrUsePutForPerMemberSharing = &common.BusinessError{Code: 400, Message: "use PUT /groups/:id/permissions with target_user_id for per-member sharing"}
)

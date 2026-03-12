package metric

import "storage-service/internal/common"

var (
	ErrInvalidRange      = &common.BusinessError{Code: 400, Message: "invalid time range"}
	ErrMissingTimes      = &common.BusinessError{Code: 400, Message: "start_time and end_time are required for custom range"}
	ErrUnknownMetricType = &common.BusinessError{Code: 400, Message: "unknown metric type"}
	ErrMetricNotFound    = &common.BusinessError{Code: 404, Message: "metric data not found"}
)

package connection

import (
	"github.com/google/uuid"
	"time"
)

// Status định nghĩa các trạng thái có thể có của một connection.
// Dùng hằng số giúp code sạch sẽ và tránh lỗi chính tả.
type Status string

const (
	StatusPending  Status = "pending"
	StatusAccepted Status = "accepted"
)

// Connection đại diện cho một mối quan hệ giữa hai người dùng.
type Connection struct {
	UserOneID   uuid.UUID `json:"user_one_id" db:"user_one_id"`
	UserTwoID   uuid.UUID `json:"user_two_id" db:"user_two_id"`
	RequesterID uuid.UUID `json:"requester_id" db:"requester_id"`
	Status      Status    `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// UserInfo là một struct rút gọn để hiển thị thông tin user
// trong danh sách kết nối, tránh trả về các thông tin nhạy cảm.
// (Tùy chọn, nhưng là một good practice).
type UserInfo struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Email   string    `json:"email"`
	Picture string    `json:"picture"`
}

package domain

import (
	"time"

	"github.com/google/uuid"
)

type GroupMember struct {
	UserID    uuid.UUID  `json:"user_id"`
	Role      string     `json:"role"`
	Status    string     `json:"status"`
	InvitedBy uuid.UUID  `json:"invited_by"`
	JoinedAt  *time.Time `json:"joined_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type InvitationResponse struct {
	ID            uuid.UUID `json:"id"`
	SentAt        time.Time `json:"sent_at"`
	MemberCount   int       `json:"member_count"`
	SharedMetrics []string  `json:"shared_metrics"`
	Group         GroupInfo `json:"group"`
	Inviter       UserInfo  `json:"inviter"`
	InviteTo      UserInfo  `json:"invite_to"`
}

type GroupInfo struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	MemberCount int       `json:"member_count"`
}

type UserInfo struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

type SentInvitationResponse struct {
	GroupID     uuid.UUID `json:"group_id"`
	UserID      uuid.UUID `json:"user_id"`
	UserEmail   string    `json:"user_email"`
	UserName    string    `json:"user_name"`
	InvitedBy   uuid.UUID `json:"invited_by"`
	InviterName string    `json:"inviter_name"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

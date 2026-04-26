package medication

import (
	"context"
	"encoding/json"
	"time"
)

// --- Domain Models ---

type Medication struct {
	ID           string                `json:"id"`
	UserID       string                `json:"user_id"`
	Name         string                `json:"name"`
	Dosage       string                `json:"dosage"`
	Instructions string                `json:"instructions"`
	PrescribedBy string                `json:"prescribed_by"`
	IsActive     bool                  `json:"is_active"`
	Frequency    json.RawMessage       `json:"frequency"`
	StartDate    time.Time             `json:"start_date"`
	Timezone     string                `json:"timezone"` // Sourced from user profile
	Reminders    []MedicationReminder  `json:"reminders"`
	Shares       []MedicationShare     `json:"shares,omitempty"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

type MedicationReminder struct {
	ID           string     `json:"id"`
	MedicationID string     `json:"medication_id"`
	UserID       string     `json:"user_id"`
	Time         string     `json:"time"`
	IsEnabled    bool       `json:"is_enabled"`
	LastTaken    *time.Time `json:"last_taken"`
	MissedCount  int        `json:"missed_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type MedicationShare struct {
	ID                  string     `json:"id"`
	MedicationID        string     `json:"medication_id"`
	GroupID             string     `json:"group_id"`
	SharedWithUserID    string     `json:"shared_with_user_id"`
	NotifyOffsetMinutes int        `json:"notify_offset_minutes"`
	LastNotifiedRemID   *string    `json:"last_notified_reminder_id"`
	LastNotifiedDate    *time.Time `json:"last_notified_date"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// --- Request DTOs ---

type CreateMedicationRequest struct {
	Name         string              `json:"name" binding:"required"`
	Dosage       string              `json:"dosage" binding:"required"`
	Frequency    json.RawMessage     `json:"frequency" binding:"required"`
	StartDate    string              `json:"start_date" binding:"required"`
	Instructions string              `json:"instructions"`
	PrescribedBy string              `json:"prescribed_by"`
	IsActive     *bool               `json:"is_active"`
	Reminders    []CreateReminderInput `json:"reminders"`
	Shares       []CreateShareInput    `json:"shares"`
}

type CreateShareInput struct {
	GroupID             string `json:"group_id" binding:"required"`
	SharedWithUserID    string `json:"shared_with_user_id" binding:"required"`
	NotifyOffsetMinutes int    `json:"notify_offset_minutes"`
}

type CreateReminderInput struct {
	Time      string `json:"time"`
	IsEnabled *bool  `json:"is_enabled"`
}

// --- Interfaces ---

type Repository interface {
	Create(ctx context.Context, med *Medication) error
	ListByUserID(ctx context.Context, userID string) ([]Medication, error)
	Delete(ctx context.Context, medicationID string, userID string) error
	ToggleTake(ctx context.Context, medicationID string, reminderID string, userID string) error
	GetRemindersToTrigger(ctx context.Context) ([]MedicationReminder, error)

	// Sharing
	CreateShare(ctx context.Context, share *MedicationShare) error
	ListShares(ctx context.Context, medicationID string) ([]MedicationShare, error)
	DeleteShare(ctx context.Context, shareID string, medicationID string, userID string) error
	GetMissedRemindersToNotify(ctx context.Context, now time.Time) ([]MissedReminderNotification, error)
	UpdateShareNotificationStatus(ctx context.Context, shareID string, reminderID string, notifiedDate time.Time) error
}

type MissedReminderNotification struct {
	ShareID            string
	MedicationID       string
	MedicationName     string
	ReminderID         string
	ReminderTime       string
	OwnerUserID        string
	SharedWithUserID   string
	NotifyOffset       int
	Timezone           string
}

type Service interface {
	CreateMedication(ctx context.Context, userID string, req CreateMedicationRequest) (*Medication, error)
	ListMedications(ctx context.Context, userID string) ([]Medication, error)
	DeleteMedication(ctx context.Context, medicationID string, userID string) error
	TakeMedication(ctx context.Context, medicationID string, reminderID string, userID string) ([]Medication, error)
	CheckAndTriggerReminders(ctx context.Context) error
	RegisterDeviceToken(ctx context.Context, userID, token, platform string) error

	// Sharing
	AddShare(ctx context.Context, userID string, medicationID string, req CreateShareInput) error
	RemoveShare(ctx context.Context, userID string, medicationID string, shareID string) error
	ListShares(ctx context.Context, userID string, medicationID string) ([]MedicationShare, error)
}

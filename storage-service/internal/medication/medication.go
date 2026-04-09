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
	Timezone     string                `json:"timezone"`
	Reminders    []MedicationReminder  `json:"reminders"`
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

// --- Request DTOs ---

type CreateMedicationRequest struct {
	Name         string              `json:"name" binding:"required"`
	Dosage       string              `json:"dosage" binding:"required"`
	Frequency    json.RawMessage     `json:"frequency" binding:"required"`
	StartDate    string              `json:"start_date" binding:"required"`
	Timezone     string              `json:"timezone"`
	Instructions string              `json:"instructions"`
	PrescribedBy string              `json:"prescribed_by"`
	IsActive     *bool               `json:"is_active"`
	Reminders    []CreateReminderInput `json:"reminders"`
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
	GetRemindersToTrigger(ctx context.Context, triggerTime string) ([]MedicationReminder, error)
}

type Service interface {
	CreateMedication(ctx context.Context, userID string, req CreateMedicationRequest) (*Medication, error)
	ListMedications(ctx context.Context, userID string) ([]Medication, error)
	DeleteMedication(ctx context.Context, medicationID string, userID string) error
	TakeMedication(ctx context.Context, medicationID string, reminderID string, userID string) ([]Medication, error)
	CheckAndTriggerReminders(ctx context.Context) error
	RegisterDeviceToken(ctx context.Context, userID, token, platform string) error
}

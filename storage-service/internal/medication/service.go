package medication

import (
	"context"
	"fmt"
	"log"
	"storage-service/internal/notification"
	"time"
)

type medicationService struct {
	repo                Repository
	notificationService notification.Service
	tokenRepo           notification.Repository
}

func NewMedicationService(repo Repository, notificationService notification.Service, tokenRepo notification.Repository) Service {
	return &medicationService{
		repo:                repo,
		notificationService: notificationService,
		tokenRepo:           tokenRepo,
	}
}

func (s *medicationService) CreateMedication(ctx context.Context, userID string, req CreateMedicationRequest) (*Medication, error) {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		startDate = time.Now() // Fallback if parse fails
	}

	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	med := &Medication{
		UserID:       userID,
		Name:         req.Name,
		Dosage:       req.Dosage,
		Instructions: req.Instructions,
		PrescribedBy: req.PrescribedBy,
		IsActive:     isActive,
		Frequency:    req.Frequency,
		StartDate:    startDate,
		Timezone:     timezone,
	}

	// Map reminder inputs → domain reminders
	for _, ri := range req.Reminders {
		enabled := true
		if ri.IsEnabled != nil {
			enabled = *ri.IsEnabled
		}
		med.Reminders = append(med.Reminders, MedicationReminder{
			Time:      ri.Time,
			IsEnabled: enabled,
		})
	}
	// If no reminders, repo.Create will default to 08:00

	if err := s.repo.Create(ctx, med); err != nil {
		return nil, err
	}

	return med, nil
}

func (s *medicationService) ListMedications(ctx context.Context, userID string) ([]Medication, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *medicationService) DeleteMedication(ctx context.Context, medicationID string, userID string) error {
	return s.repo.Delete(ctx, medicationID, userID)
}

// TakeMedication toggles the take status, then returns the full medication list (per spec).
func (s *medicationService) TakeMedication(ctx context.Context, medicationID string, reminderID string, userID string) ([]Medication, error) {
	if err := s.repo.ToggleTake(ctx, medicationID, reminderID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListByUserID(ctx, userID)
}

func (s *medicationService) CheckAndTriggerReminders(ctx context.Context) error {
	now := time.Now()
	triggerTime := now.Format("15:04")

	log.Printf("Checking reminders for time: %s", triggerTime)

	reminders, err := s.repo.GetRemindersToTrigger(ctx, triggerTime)
	if err != nil {
		return fmt.Errorf("error getting reminders: %v", err)
	}

	for _, r := range reminders {
		log.Printf("Triggering reminder %s (medication_id: %s, user_id: %s)", r.ID, r.MedicationID, r.UserID)

		payload := notification.Notification{
			Title: "Medication Reminder",
			Body:  fmt.Sprintf("It's time for your medication: %s", r.MedicationID), // Ideally fetch medication name here
			Data: map[string]string{
				"type":          "MEDICATION_REMINDER",
				"reminder_id":   r.ID,
				"medication_id": r.MedicationID,
				"time":          r.Time,
				"timestamp":     fmt.Sprintf("%d", time.Now().Unix()),
			},
		}

		err := s.notificationService.SendToUser(ctx, r.UserID, payload)
		if err != nil {
			log.Printf("Failed to send FCM notification for reminder %s: %v", r.ID, err)
		}
	}

	return nil
}

func (s *medicationService) RegisterDeviceToken(ctx context.Context, userID, token, platform string) error {
	deviceToken := notification.DeviceToken{
		UserID:   userID,
		Token:    token,
		Platform: platform,
	}
	return s.tokenRepo.SaveToken(ctx, deviceToken)
}

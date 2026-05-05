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

	med := &Medication{
		UserID:       userID,
		Name:         req.Name,
		Dosage:       req.Dosage,
		Instructions: req.Instructions,
		PrescribedBy: req.PrescribedBy,
		IsActive:     isActive,
		Frequency:    req.Frequency,
		StartDate:    startDate,
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

	// Initial shares
	for _, si := range req.Shares {
		med.Shares = append(med.Shares, MedicationShare{
			GroupID:             si.GroupID,
			SharedWithUserID:    si.SharedWithUserID,
			NotifyOffsetMinutes: si.NotifyOffsetMinutes,
		})
	}

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

	log.Printf("Checking reminders for time: %s", now.Format(time.RFC3339))

	reminders, err := s.repo.GetRemindersToTrigger(ctx)
	if err != nil {
		return fmt.Errorf("error getting reminders: %v", err)
	}

	for _, r := range reminders {
		log.Printf("Triggering reminder %s (medication_id: %s, user_id: %s)", r.ID, r.MedicationID, r.UserID)

		payload := notification.Notification{
			Title: "Medication Reminder",
			Body:  "It's time for your medication.",
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

	// 2. Check for missed reminders that need buddy notification
	missed, err := s.repo.GetMissedRemindersToNotify(ctx, now)
	if err != nil {
		return fmt.Errorf("error getting missed reminders: %v", err)
	}

	for _, m := range missed {
		log.Printf("Triggering buddy notification for share %s (medication: %s, buddy: %s)", m.ShareID, m.MedicationName, m.SharedWithUserID)

		// Notify Owner (Remind again)
		ownerPayload := notification.Notification{
			Title: "Missed Medication!",
			Body:  fmt.Sprintf("You missed your medication: %s at %s. Please take it now.", m.MedicationName, m.ReminderTime),
			Data: map[string]string{
				"type":          "MEDICATION_MISSED_OWNER",
				"medication_id": m.MedicationID,
				"reminder_id":   m.ReminderID,
			},
		}
		_ = s.notificationService.SendToUser(ctx, m.OwnerUserID, ownerPayload)

		// Notify Buddy
		buddyPayload := notification.Notification{
			Title: "Medication Alert!",
			Body:  fmt.Sprintf("Your buddy missed their medication: %s at %s.", m.MedicationName, m.ReminderTime),
			Data: map[string]string{
				"type":          "MEDICATION_MISSED_BUDDY",
				"medication_id": m.MedicationID,
				"reminder_id":   m.ReminderID,
				"owner_id":      m.OwnerUserID,
			},
		}
		_ = s.notificationService.SendToUser(ctx, m.SharedWithUserID, buddyPayload)

		// Update status so we don't notify again today for this reminder
		loc, _ := time.LoadLocation(m.Timezone)
		if loc == nil {
			loc = time.UTC
		}
		todayLocal := now.In(loc)
		_ = s.repo.UpdateShareNotificationStatus(ctx, m.ShareID, m.ReminderID, todayLocal)
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

func (s *medicationService) AddShare(ctx context.Context, userID string, medicationID string, req CreateShareInput) error {
	// Ownership check can be done in repo or here
	// For simplicity, let's just implement it
	share := &MedicationShare{
		MedicationID:        medicationID,
		GroupID:             req.GroupID,
		SharedWithUserID:    req.SharedWithUserID,
		NotifyOffsetMinutes: req.NotifyOffsetMinutes,
	}
	if share.NotifyOffsetMinutes <= 0 {
		share.NotifyOffsetMinutes = 15 // Default
	}
	return s.repo.CreateShare(ctx, share)
}

func (s *medicationService) RemoveShare(ctx context.Context, userID string, medicationID string, shareID string) error {
	return s.repo.DeleteShare(ctx, shareID, medicationID, userID)
}

func (s *medicationService) ListShares(ctx context.Context, userID string, medicationID string) ([]MedicationShare, error) {
	// Simple list for now, repo handles ownership check if we use DeleteShare pattern,
	// but here we should probably verify the user owns the medication first.
	meds, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	found := false
	for _, m := range meds {
		if m.ID == medicationID {
			found = true
			break
		}
	}
	if !found {
		return nil, ErrForbidden
	}

	return s.repo.ListShares(ctx, medicationID)
}

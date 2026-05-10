package medication_test

import (
	"context"
	"errors"
	"testing"

	"storage-service/internal/medication"
	"storage-service/internal/notification"
	"storage-service/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type medTestEnv struct {
	repo      *mocks.MedicationRepository
	notif     *mocks.NotificationService
	tokenRepo *mocks.NotificationRepository
	service   medication.Service
}

func newMedTestEnv() *medTestEnv {
	repo := new(mocks.MedicationRepository)
	notif := new(mocks.NotificationService)
	tokenRepo := new(mocks.NotificationRepository)
	return &medTestEnv{
		repo:      repo,
		notif:     notif,
		tokenRepo: tokenRepo,
		service:   medication.NewMedicationService(repo, notif, tokenRepo),
	}
}

// boolPtr returns a pointer to a bool – used in CreateMedicationRequest.
func boolPtr(b bool) *bool { return &b }

// ─── CreateMedication ─────────────────────────────────────────────────────────

func TestCreateMedication(t *testing.T) {
	t.Run("success: creates medication with valid request", func(t *testing.T) {
		// Service must build a Medication domain object and pass it to repo.Create.
		env := newMedTestEnv()
		userID := "user-1"
		req := medication.CreateMedicationRequest{
			Name:      "Aspirin",
			Dosage:    "100mg",
			Frequency: []byte(`{"type":"daily"}`),
			StartDate: "2026-01-01",
		}

		env.repo.On("Create", mock.Anything, mock.MatchedBy(func(m *medication.Medication) bool {
			return m.UserID == userID && m.Name == "Aspirin" && m.IsActive
		})).Return(nil).Once()

		got, err := env.service.CreateMedication(context.Background(), userID, req)

		assert.NoError(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, "Aspirin", got.Name)
		assert.True(t, got.IsActive)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: IsActive=false is respected", func(t *testing.T) {
		// Callers can create an inactive medication from the start.
		env := newMedTestEnv()
		req := medication.CreateMedicationRequest{
			Name:      "Paracetamol",
			Dosage:    "500mg",
			Frequency: []byte(`{}`),
			StartDate: "2026-01-01",
			IsActive:  boolPtr(false),
		}

		env.repo.On("Create", mock.Anything, mock.MatchedBy(func(m *medication.Medication) bool {
			return !m.IsActive
		})).Return(nil).Once()

		got, err := env.service.CreateMedication(context.Background(), "user-1", req)

		assert.NoError(t, err)
		assert.False(t, got.IsActive)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: invalid date string falls back to time.Now()", func(t *testing.T) {
		// A non-parseable StartDate must not return an error; the service falls back
		// to the current time and proceeds normally.
		env := newMedTestEnv()
		req := medication.CreateMedicationRequest{
			Name:      "Ibuprofen",
			Dosage:    "400mg",
			Frequency: []byte(`{}`),
			StartDate: "not-a-date",
		}

		env.repo.On("Create", mock.Anything, mock.AnythingOfType("*medication.Medication")).Return(nil).Once()

		got, err := env.service.CreateMedication(context.Background(), "user-1", req)

		assert.NoError(t, err)
		assert.NotNil(t, got)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: reminders are mapped from request", func(t *testing.T) {
		// ReminderInput structs must be converted to domain MedicationReminder values.
		env := newMedTestEnv()
		req := medication.CreateMedicationRequest{
			Name:      "Vitamin D",
			Dosage:    "1000IU",
			Frequency: []byte(`{}`),
			StartDate: "2026-01-01",
			Reminders: []medication.CreateReminderInput{
				{Time: "08:00", IsEnabled: boolPtr(true)},
				{Time: "20:00", IsEnabled: boolPtr(false)},
			},
		}

		env.repo.On("Create", mock.Anything, mock.MatchedBy(func(m *medication.Medication) bool {
			return len(m.Reminders) == 2 &&
				m.Reminders[0].Time == "08:00" && m.Reminders[0].IsEnabled &&
				m.Reminders[1].Time == "20:00" && !m.Reminders[1].IsEnabled
		})).Return(nil).Once()

		_, err := env.service.CreateMedication(context.Background(), "user-1", req)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: repository failure propagates", func(t *testing.T) {
		// When repo.Create returns an error the service must surface it unchanged.
		env := newMedTestEnv()
		dbErr := errors.New("duplicate key")
		req := medication.CreateMedicationRequest{
			Name: "X", Dosage: "1mg", Frequency: []byte(`{}`), StartDate: "2026-01-01",
		}

		env.repo.On("Create", mock.Anything, mock.AnythingOfType("*medication.Medication")).Return(dbErr).Once()

		got, err := env.service.CreateMedication(context.Background(), "user-1", req)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertExpectations(t)
	})
}

// ─── ListMedications ─────────────────────────────────────────────────────────

func TestListMedications(t *testing.T) {
	t.Run("success: returns medications for user", func(t *testing.T) {
		// Service is a thin delegation layer over repo.ListByUserID.
		env := newMedTestEnv()
		userID := "user-1"
		want := []medication.Medication{
			{ID: "med-1", UserID: userID, Name: "Aspirin"},
			{ID: "med-2", UserID: userID, Name: "Vitamin C"},
		}

		env.repo.On("ListByUserID", mock.Anything, userID).Return(want, nil).Once()

		got, err := env.service.ListMedications(context.Background(), userID)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: empty list when user has no medications", func(t *testing.T) {
		// No medications is valid; service must not return an error.
		env := newMedTestEnv()

		env.repo.On("ListByUserID", mock.Anything, "user-2").Return([]medication.Medication{}, nil).Once()

		got, err := env.service.ListMedications(context.Background(), "user-2")

		assert.NoError(t, err)
		assert.Empty(t, got)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: repository failure propagates", func(t *testing.T) {
		env := newMedTestEnv()
		dbErr := errors.New("connection lost")

		env.repo.On("ListByUserID", mock.Anything, "user-1").Return(nil, dbErr).Once()

		_, err := env.service.ListMedications(context.Background(), "user-1")

		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertExpectations(t)
	})
}

// ─── DeleteMedication ─────────────────────────────────────────────────────────

func TestDeleteMedication(t *testing.T) {
	t.Run("success: deletes medication that belongs to user", func(t *testing.T) {
		// Happy path: service delegates directly to repo.Delete.
		env := newMedTestEnv()

		env.repo.On("Delete", mock.Anything, "med-1", "user-1").Return(nil).Once()

		err := env.service.DeleteMedication(context.Background(), "med-1", "user-1")

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: medication not found or not owned by user", func(t *testing.T) {
		// Repository enforces ownership; the service must propagate the error.
		env := newMedTestEnv()

		env.repo.On("Delete", mock.Anything, "med-99", "user-1").Return(medication.ErrNotFound).Once()

		err := env.service.DeleteMedication(context.Background(), "med-99", "user-1")

		assert.ErrorIs(t, err, medication.ErrNotFound)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: unexpected database error propagates", func(t *testing.T) {
		env := newMedTestEnv()
		dbErr := errors.New("foreign key constraint")

		env.repo.On("Delete", mock.Anything, "med-1", "user-1").Return(dbErr).Once()

		err := env.service.DeleteMedication(context.Background(), "med-1", "user-1")

		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertExpectations(t)
	})
}

// ─── TakeMedication ───────────────────────────────────────────────────────────

func TestTakeMedication(t *testing.T) {
	t.Run("success: toggles reminder and returns updated medication list", func(t *testing.T) {
		// After a successful toggle, the service returns the full medication list
		// so the client can refresh in one round-trip.
		env := newMedTestEnv()
		userID := "user-1"
		want := []medication.Medication{{ID: "med-1", UserID: userID, Name: "Aspirin"}}

		env.repo.On("ToggleTake", mock.Anything, "med-1", "rem-1", userID).Return(nil).Once()
		env.repo.On("ListByUserID", mock.Anything, userID).Return(want, nil).Once()

		got, err := env.service.TakeMedication(context.Background(), "med-1", "rem-1", userID)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: ToggleTake failure propagates; list is not fetched", func(t *testing.T) {
		// If the toggle itself fails the service must return early without fetching the list.
		env := newMedTestEnv()
		dbErr := errors.New("not found")

		env.repo.On("ToggleTake", mock.Anything, "med-1", "rem-bad", "user-1").Return(dbErr).Once()

		got, err := env.service.TakeMedication(context.Background(), "med-1", "rem-bad", "user-1")

		assert.Nil(t, got)
		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertNotCalled(t, "ListByUserID", mock.Anything, mock.Anything)
		env.repo.AssertExpectations(t)
	})
}

// ─── AddShare ─────────────────────────────────────────────────────────────────

func TestAddShare(t *testing.T) {
	t.Run("success: share is created with provided offset", func(t *testing.T) {
		// Service delegates to repo.CreateShare after building the domain object.
		env := newMedTestEnv()
		req := medication.CreateShareInput{
			GroupID:             "group-1",
			SharedWithUserID:    "buddy-1",
			NotifyOffsetMinutes: 30,
		}

		env.repo.On("CreateShare", mock.Anything, mock.MatchedBy(func(s *medication.MedicationShare) bool {
			return s.MedicationID == "med-1" && s.NotifyOffsetMinutes == 30
		})).Return(nil).Once()

		err := env.service.AddShare(context.Background(), "user-1", "med-1", req)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: zero offset defaults to 15 minutes", func(t *testing.T) {
		// When the caller omits the offset the service must apply a 15-minute default.
		env := newMedTestEnv()
		req := medication.CreateShareInput{
			GroupID:             "group-1",
			SharedWithUserID:    "buddy-1",
			NotifyOffsetMinutes: 0, // should default to 15
		}

		env.repo.On("CreateShare", mock.Anything, mock.MatchedBy(func(s *medication.MedicationShare) bool {
			return s.NotifyOffsetMinutes == 15
		})).Return(nil).Once()

		err := env.service.AddShare(context.Background(), "user-1", "med-1", req)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: repository failure propagates", func(t *testing.T) {
		env := newMedTestEnv()
		req := medication.CreateShareInput{GroupID: "g-1", SharedWithUserID: "b-1", NotifyOffsetMinutes: 10}
		dbErr := errors.New("unique constraint violation")

		env.repo.On("CreateShare", mock.Anything, mock.AnythingOfType("*medication.MedicationShare")).Return(dbErr).Once()

		err := env.service.AddShare(context.Background(), "user-1", "med-1", req)

		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertExpectations(t)
	})
}

// ─── RemoveShare ─────────────────────────────────────────────────────────────

func TestRemoveShare(t *testing.T) {
	t.Run("success: removes share by IDs", func(t *testing.T) {
		// Service delegates directly to repo.DeleteShare.
		env := newMedTestEnv()

		env.repo.On("DeleteShare", mock.Anything, "share-1", "med-1", "user-1").Return(nil).Once()

		err := env.service.RemoveShare(context.Background(), "user-1", "med-1", "share-1")

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: share not found propagates", func(t *testing.T) {
		// The repository enforces ownership; the service must propagate the error.
		env := newMedTestEnv()

		env.repo.On("DeleteShare", mock.Anything, "bad-share", "med-1", "user-1").Return(medication.ErrNotFound).Once()

		err := env.service.RemoveShare(context.Background(), "user-1", "med-1", "bad-share")

		assert.ErrorIs(t, err, medication.ErrNotFound)
		env.repo.AssertExpectations(t)
	})
}

// ─── ListShares ───────────────────────────────────────────────────────────────

func TestListShares(t *testing.T) {
	t.Run("success: returns shares for owned medication", func(t *testing.T) {
		// Service must verify ownership via ListByUserID before fetching shares.
		env := newMedTestEnv()
		userID, medID := "user-1", "med-1"
		userMeds := []medication.Medication{{ID: medID, UserID: userID}}
		want := []medication.MedicationShare{{ID: "share-1", MedicationID: medID}}

		env.repo.On("ListByUserID", mock.Anything, userID).Return(userMeds, nil).Once()
		env.repo.On("ListShares", mock.Anything, medID).Return(want, nil).Once()

		got, err := env.service.ListShares(context.Background(), userID, medID)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: medication does not belong to user returns ErrForbidden", func(t *testing.T) {
		// If the medication ID is not in the user's list the service must return
		// ErrForbidden without calling ListShares.
		env := newMedTestEnv()
		userID := "user-1"

		// User owns med-2 but the caller asks for med-99
		env.repo.On("ListByUserID", mock.Anything, userID).Return([]medication.Medication{
			{ID: "med-2", UserID: userID},
		}, nil).Once()

		got, err := env.service.ListShares(context.Background(), userID, "med-99")

		assert.Nil(t, got)
		assert.ErrorIs(t, err, medication.ErrForbidden)
		env.repo.AssertNotCalled(t, "ListShares", mock.Anything, mock.Anything)
	})

	t.Run("error: ListByUserID failure propagates", func(t *testing.T) {
		env := newMedTestEnv()
		dbErr := errors.New("query failed")

		env.repo.On("ListByUserID", mock.Anything, "user-1").Return(nil, dbErr).Once()

		_, err := env.service.ListShares(context.Background(), "user-1", "med-1")

		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertExpectations(t)
	})
}

// ─── RegisterDeviceToken ──────────────────────────────────────────────────────

func TestRegisterDeviceToken(t *testing.T) {
	t.Run("success: saves token via notification token repository", func(t *testing.T) {
		// Service converts parameters into a DeviceToken and persists via tokenRepo.
		env := newMedTestEnv()
		token := notification.DeviceToken{
			UserID:   "user-1",
			Token:    "fcm-abc",
			Platform: "android",
		}

		env.tokenRepo.On("SaveToken", mock.Anything, token).Return(nil).Once()

		err := env.service.RegisterDeviceToken(context.Background(), "user-1", "fcm-abc", "android")

		assert.NoError(t, err)
		env.tokenRepo.AssertExpectations(t)
	})

	t.Run("error: repository failure propagates", func(t *testing.T) {
		env := newMedTestEnv()
		dbErr := errors.New("token store unavailable")
		token := notification.DeviceToken{UserID: "user-1", Token: "t", Platform: "ios"}

		env.tokenRepo.On("SaveToken", mock.Anything, token).Return(dbErr).Once()

		err := env.service.RegisterDeviceToken(context.Background(), "user-1", "t", "ios")

		assert.ErrorIs(t, err, dbErr)
		env.tokenRepo.AssertExpectations(t)
	})
}

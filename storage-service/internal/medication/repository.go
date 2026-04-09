package medication

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

// Create inserts a medication and its reminders in a transaction.
// Server generates all UUIDs. If no reminders provided, defaults to 08:00.
func (r *postgresRepository) Create(ctx context.Context, med *Medication) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Default is_active to true if not set
	medQuery := `
		INSERT INTO medications (user_id, name, dosage, instructions, prescribed_by, is_active, frequency, start_date, timezone)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`
	err = tx.QueryRow(ctx, medQuery,
		med.UserID, med.Name, med.Dosage, med.Instructions, med.PrescribedBy,
		med.IsActive, med.Frequency, med.StartDate, med.Timezone,
	).Scan(&med.ID, &med.CreatedAt, &med.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert medication: %w", err)
	}

	// Default reminder if none provided
	if len(med.Reminders) == 0 {
		med.Reminders = []MedicationReminder{{Time: "08:00", IsEnabled: true}}
	}

	remQuery := `
		INSERT INTO medication_reminders (medication_id, time, is_enabled)
		VALUES ($1, $2, $3)
		RETURNING id, last_taken, missed_count, created_at, updated_at
	`
	for i := range med.Reminders {
		rem := &med.Reminders[i]
		rem.MedicationID = med.ID
		if rem.Time == "" {
			rem.Time = "08:00"
		}
		err = tx.QueryRow(ctx, remQuery, med.ID, rem.Time, rem.IsEnabled).
			Scan(&rem.ID, &rem.LastTaken, &rem.MissedCount, &rem.CreatedAt, &rem.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert reminder: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// ListByUserID returns all medications with nested reminders for a user.
func (r *postgresRepository) ListByUserID(ctx context.Context, userID string) ([]Medication, error) {
	medQuery := `
		SELECT id, user_id, name, dosage, instructions, prescribed_by, is_active,
		       frequency, start_date, timezone, created_at, updated_at
		FROM medications
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	medRows, err := r.pool.Query(ctx, medQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list medications: %w", err)
	}
	defer medRows.Close()

	var meds []Medication
	for medRows.Next() {
		var m Medication
		var freqBytes []byte
		err := medRows.Scan(
			&m.ID, &m.UserID, &m.Name, &m.Dosage, &m.Instructions, &m.PrescribedBy,
			&m.IsActive, &freqBytes, &m.StartDate, &m.Timezone, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan medication: %w", err)
		}
		m.Frequency = json.RawMessage(freqBytes)
		m.Reminders = []MedicationReminder{} // initialize to empty slice (not null in JSON)
		meds = append(meds, m)
	}

	if len(meds) == 0 {
		return []Medication{}, nil
	}

	// Collect medication IDs for batch reminder query
	medIDs := make([]string, len(meds))
	medIndex := make(map[string]int, len(meds))
	for i, m := range meds {
		medIDs[i] = m.ID
		medIndex[m.ID] = i
	}

	remQuery := `
		SELECT id, medication_id, time, is_enabled, last_taken, missed_count, created_at, updated_at
		FROM medication_reminders
		WHERE medication_id = ANY($1)
		ORDER BY time ASC
	`
	remRows, err := r.pool.Query(ctx, remQuery, medIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to list reminders: %w", err)
	}
	defer remRows.Close()

	for remRows.Next() {
		var rem MedicationReminder
		err := remRows.Scan(
			&rem.ID, &rem.MedicationID, &rem.Time, &rem.IsEnabled,
			&rem.LastTaken, &rem.MissedCount, &rem.CreatedAt, &rem.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reminder: %w", err)
		}
		if idx, ok := medIndex[rem.MedicationID]; ok {
			meds[idx].Reminders = append(meds[idx].Reminders, rem)
		}
	}

	return meds, nil
}

// Delete hard-deletes a medication (reminders cascade).
// Returns error if medication not found or not owned by user.
func (r *postgresRepository) Delete(ctx context.Context, medicationID string, userID string) error {
	query := `DELETE FROM medications WHERE id = $1 AND user_id = $2`
	tag, err := r.pool.Exec(ctx, query, medicationID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete medication: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ToggleTake toggles last_taken on a reminder.
// If last_taken is same UTC day as now → set NULL; else → set now.
// Returns ErrForbidden if medication belongs to another user, ErrNotFound if not found.
func (r *postgresRepository) ToggleTake(ctx context.Context, medicationID string, reminderID string, userID string) error {
	// 1. Verify medication ownership
	var ownerID string
	err := r.pool.QueryRow(ctx,
		`SELECT user_id FROM medications WHERE id = $1`, medicationID,
	).Scan(&ownerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		return fmt.Errorf("failed to check medication ownership: %w", err)
	}
	if ownerID != userID {
		return ErrForbidden
	}

	// 2. Get current last_taken
	var lastTaken *time.Time
	err = r.pool.QueryRow(ctx,
		`SELECT last_taken FROM medication_reminders WHERE id = $1 AND medication_id = $2`,
		reminderID, medicationID,
	).Scan(&lastTaken)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		return fmt.Errorf("failed to get reminder: %w", err)
	}

	// 3. Toggle logic: same UTC calendar day → NULL, else → NOW()
	now := time.Now().UTC()
	if lastTaken != nil && sameUTCDay(*lastTaken, now) {
		_, err = r.pool.Exec(ctx,
			`UPDATE medication_reminders SET last_taken = NULL, updated_at = NOW() WHERE id = $1`,
			reminderID,
		)
	} else {
		_, err = r.pool.Exec(ctx,
			`UPDATE medication_reminders SET last_taken = $1, updated_at = NOW() WHERE id = $2`,
			now, reminderID,
		)
	}
	if err != nil {
		return fmt.Errorf("failed to toggle take: %w", err)
	}

	return nil
}

// GetRemindersToTrigger returns enabled reminders matching the given HH:mm time,
// along with medication name/dosage for notification payload.
func (r *postgresRepository) GetRemindersToTrigger(ctx context.Context, triggerTime string) ([]MedicationReminder, error) {
	query := `
		SELECT r.id, r.medication_id, m.user_id, r.time, r.is_enabled, r.last_taken, r.missed_count, r.created_at, r.updated_at
		FROM medication_reminders r
		JOIN medications m ON m.id = r.medication_id
		WHERE r.time = $1 AND r.is_enabled = true AND m.is_active = true
	`
	rows, err := r.pool.Query(ctx, query, triggerTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get reminders to trigger: %w", err)
	}
	defer rows.Close()

	var reminders []MedicationReminder
	for rows.Next() {
		var rem MedicationReminder
		err := rows.Scan(
			&rem.ID, &rem.MedicationID, &rem.UserID, &rem.Time, &rem.IsEnabled,
			&rem.LastTaken, &rem.MissedCount, &rem.CreatedAt, &rem.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reminder: %w", err)
		}
		reminders = append(reminders, rem)
	}

	return reminders, nil
}

// --- Helpers ---

func sameUTCDay(a, b time.Time) bool {
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}

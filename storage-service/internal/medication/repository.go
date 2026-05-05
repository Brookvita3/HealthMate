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
		INSERT INTO medications (user_id, name, dosage, instructions, prescribed_by, is_active, frequency, start_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`
	err = tx.QueryRow(ctx, medQuery,
		med.UserID, med.Name, med.Dosage, med.Instructions, med.PrescribedBy,
		med.IsActive, med.Frequency, med.StartDate,
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

	// Insert initial shares if any
	for i := range med.Shares {
		s := &med.Shares[i]
		s.MedicationID = med.ID
		shareQuery := `
			INSERT INTO medication_shares (medication_id, group_id, shared_with_user_id, notify_offset_minutes)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at, updated_at
		`
		err = tx.QueryRow(ctx, shareQuery, med.ID, s.GroupID, s.SharedWithUserID, s.NotifyOffsetMinutes).
			Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert share: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// ListByUserID returns all medications with nested reminders for a user.
func (r *postgresRepository) ListByUserID(ctx context.Context, userID string) ([]Medication, error) {
	medQuery := `
		SELECT m.id, m.user_id, m.name, m.dosage, m.instructions, m.prescribed_by, m.is_active,
		       m.frequency, m.start_date, u.timezone, m.created_at, m.updated_at
		FROM medications m
		JOIN users u ON u.id = m.user_id
		WHERE m.user_id = $1
		ORDER BY m.created_at DESC
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

	// Collect medication IDs for batch reminder query.
	// Dùng medication_id::text = ANY($1::text[]) vì pgx + ANY(uuid[]) với []string dễ lỗi kiểu → 500.
	medIDs := make([]string, len(meds))
	medIndex := make(map[string]int, len(meds))
	for i, m := range meds {
		medIDs[i] = m.ID
		medIndex[m.ID] = i
	}

	remQuery := `
		SELECT id, medication_id, time, is_enabled, last_taken, missed_count, created_at, updated_at
		FROM medication_reminders
		WHERE medication_id::text = ANY($1::text[])
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

	// 3. Batch fetch shares
	shareQuery := `
		SELECT id, medication_id, group_id, shared_with_user_id, notify_offset_minutes,
		       last_notified_reminder_id, last_notified_date, created_at, updated_at
		FROM medication_shares
		WHERE medication_id::text = ANY($1::text[])
	`
	shareRows, err := r.pool.Query(ctx, shareQuery, medIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to list shares: %w", err)
	}
	defer shareRows.Close()

	for shareRows.Next() {
		var s MedicationShare
		err := shareRows.Scan(
			&s.ID, &s.MedicationID, &s.GroupID, &s.SharedWithUserID, &s.NotifyOffsetMinutes,
			&s.LastNotifiedRemID, &s.LastNotifiedDate, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan share: %w", err)
		}
		if idx, ok := medIndex[s.MedicationID]; ok {
			meds[idx].Shares = append(meds[idx].Shares, s)
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

	// 2. Get current last_taken and user's timezone
	var lastTaken *time.Time
	var timezone string
	err = r.pool.QueryRow(ctx,
		`SELECT r.last_taken, u.timezone 
		 FROM medication_reminders r
		 JOIN medications m ON m.id = r.medication_id
		 JOIN users u ON u.id = m.user_id
		 WHERE r.id = $1 AND r.medication_id = $2`,
		reminderID, medicationID,
	).Scan(&lastTaken, &timezone)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		return fmt.Errorf("failed to get reminder or timezone: %w", err)
	}

	// 3. Toggle logic: same calendar day in user's timezone → NULL, else → NOW()
	loc, err := time.LoadLocation(timezone)
	if err != nil || loc == nil {
		loc = time.UTC
	}
	nowLocal := time.Now().In(loc)
	todayLocal := nowLocal.Format("2006-01-02")

	var lastTakenLocal string
	if lastTaken != nil {
		lastTakenLocal = lastTaken.In(loc).Format("2006-01-02")
	}

	nowUTC := time.Now().UTC()
	if lastTakenLocal == todayLocal {
		_, err = r.pool.Exec(ctx,
			`UPDATE medication_reminders SET last_taken = NULL, updated_at = NOW() WHERE id = $1`,
			reminderID,
		)
	} else {
		_, err = r.pool.Exec(ctx,
			`UPDATE medication_reminders SET last_taken = $1, updated_at = NOW() WHERE id = $2`,
			nowUTC, reminderID,
		)
	}
	if err != nil {
		return fmt.Errorf("failed to toggle take: %w", err)
	}

	return nil
}

// GetRemindersToTrigger returns enabled reminders where the user's local time matches the reminder time.
func (r *postgresRepository) GetRemindersToTrigger(ctx context.Context) ([]MedicationReminder, error) {
	query := `
		SELECT r.id, r.medication_id, m.user_id, r.time, r.is_enabled, r.last_taken, r.missed_count, r.created_at, r.updated_at
		FROM medication_reminders r
		JOIN medications m ON m.id = r.medication_id
		JOIN users u ON u.id = m.user_id
		WHERE m.is_active = true 
		  AND r.is_enabled = true
		  AND r.time = to_char(CURRENT_TIMESTAMP AT TIME ZONE u.timezone, 'HH24:MI')
		  AND (CURRENT_TIMESTAMP AT TIME ZONE u.timezone)::date >= m.start_date
		  AND (m.end_date IS NULL OR (CURRENT_TIMESTAMP AT TIME ZONE u.timezone)::date <= m.end_date)
	`
	rows, err := r.pool.Query(ctx, query)
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

// --- Sharing Methods ---

func (r *postgresRepository) CreateShare(ctx context.Context, share *MedicationShare) error {
	query := `
		INSERT INTO medication_shares (medication_id, group_id, shared_with_user_id, notify_offset_minutes)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query,
		share.MedicationID, share.GroupID, share.SharedWithUserID, share.NotifyOffsetMinutes,
	).Scan(&share.ID, &share.CreatedAt, &share.UpdatedAt)
}

func (r *postgresRepository) ListShares(ctx context.Context, medicationID string) ([]MedicationShare, error) {
	query := `
		SELECT id, medication_id, group_id, shared_with_user_id, notify_offset_minutes,
		       last_notified_reminder_id, last_notified_date, created_at, updated_at
		FROM medication_shares
		WHERE medication_id = $1
	`
	rows, err := r.pool.Query(ctx, query, medicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []MedicationShare
	for rows.Next() {
		var s MedicationShare
		err := rows.Scan(
			&s.ID, &s.MedicationID, &s.GroupID, &s.SharedWithUserID, &s.NotifyOffsetMinutes,
			&s.LastNotifiedRemID, &s.LastNotifiedDate, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		shares = append(shares, s)
	}
	return shares, nil
}

func (r *postgresRepository) DeleteShare(ctx context.Context, shareID string, medicationID string, userID string) error {
	// Verify medication ownership
	var ownerID string
	err := r.pool.QueryRow(ctx, `SELECT user_id FROM medications WHERE id = $1`, medicationID).Scan(&ownerID)
	if err != nil {
		return err
	}
	if ownerID != userID {
		return ErrForbidden
	}

	res, err := r.pool.Exec(ctx, `DELETE FROM medication_shares WHERE id = $1 AND medication_id = $2`, shareID, medicationID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) GetMissedRemindersToNotify(ctx context.Context, now time.Time) ([]MissedReminderNotification, error) {
	// Logic:
	// Find enabled reminders for active medications where:
	// 1. Now is past (reminder.time + share.offset)
	// 2. Not taken today (in medication's timezone)
	// 3. Not already notified for this reminder today
	// 4. Current date is within [start_date, end_date]

	query := `
		SELECT 
			s.id as share_id,
			m.id as medication_id,
			m.name as medication_name,
			r.id as reminder_id,
			r.time as reminder_time,
			m.user_id as owner_user_id,
			s.shared_with_user_id,
			s.notify_offset_minutes,
			u.timezone,
			m.start_date,
			m.end_date,
			r.last_taken
		FROM medication_shares s
		JOIN medications m ON m.id = s.medication_id
		JOIN users u ON u.id = m.user_id
		JOIN medication_reminders r ON r.medication_id = m.id
		WHERE m.is_active = true 
		  AND r.is_enabled = true
		  AND (s.last_notified_reminder_id IS NULL OR s.last_notified_reminder_id != r.id OR s.last_notified_date != (CURRENT_TIMESTAMP AT TIME ZONE u.timezone)::date)
	`
	// Note: The time arithmetic and "taken today" check will be handled in the service layer 
	// or refined here if we want more SQL-heavy logic.
	// For clarity and to handle complex timezone logic correctly, I'll fetch candidates and filter in Go
	// but I'll add basic SQL filters to reduce row count.

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MissedReminderNotification
	for rows.Next() {
		var n MissedReminderNotification
		var startDate time.Time
		var endDate *time.Time
		var lastTaken *time.Time
		var tz string
		var rTime string

		err := rows.Scan(
			&n.ShareID, &n.MedicationID, &n.MedicationName, &n.ReminderID, &rTime,
			&n.OwnerUserID, &n.SharedWithUserID, &n.NotifyOffset, &tz,
			&startDate, &endDate, &lastTaken,
		)
		if err != nil {
			return nil, err
		}
		n.Timezone = tz
		n.ReminderTime = rTime

		// Go filtering logic for timezones and active periods
		loc, err := time.LoadLocation(tz)
		if err != nil {
			loc = time.UTC
		}

		nowLocal := now.In(loc)
		todayLocal := nowLocal.Format("2006-01-02")

		// 1. Check if within [start_date, end_date]
		if nowLocal.Before(startDate) {
			continue
		}
		if endDate != nil && nowLocal.After(endDate.AddDate(0, 0, 1)) { // inclusive of end_date
			continue
		}

		// 2. Check if already taken today in local time
		if lastTaken != nil {
			lastTakenLocal := lastTaken.In(loc).Format("2006-01-02")
			if lastTakenLocal == todayLocal {
				continue
			}
		}

		// 3. Check if past (reminder.time + offset)
		// Parse HH:mm in local today
		remTimeFull, err := time.ParseInLocation("2006-01-02 15:04", todayLocal+" "+rTime, loc)
		if err != nil {
			continue
		}
		
		notifyTime := remTimeFull.Add(time.Duration(n.NotifyOffset) * time.Minute)
		if nowLocal.After(notifyTime) {
			results = append(results, n)
		}
	}

	return results, nil
}

func (r *postgresRepository) UpdateShareNotificationStatus(ctx context.Context, shareID string, reminderID string, notifiedDate time.Time) error {
	query := `
		UPDATE medication_shares 
		SET last_notified_reminder_id = $1, last_notified_date = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.pool.Exec(ctx, query, reminderID, notifiedDate, shareID)
	return err
}

// --- Helpers ---

func sameUTCDay(a, b time.Time) bool {
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}

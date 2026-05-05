-- medication_shares: allows sharing a medication schedule with specific group members
-- and setting a notification offset for missed pills.
CREATE TABLE medication_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    medication_id UUID NOT NULL REFERENCES medications(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    shared_with_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notify_offset_minutes INT NOT NULL DEFAULT 15,
    
    -- Tracking for missed notifications to avoid duplicates
    last_notified_reminder_id UUID REFERENCES medication_reminders(id) ON DELETE SET NULL,
    last_notified_date DATE, -- The date (in medication's timezone) we last notified for this share

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Ensure a medication is only shared once with a specific user
    UNIQUE(medication_id, shared_with_user_id)
);

CREATE INDEX idx_medication_shares_medication_id ON medication_shares(medication_id);
CREATE INDEX idx_medication_shares_shared_with ON medication_shares(shared_with_user_id);
CREATE INDEX idx_medication_shares_group ON medication_shares(group_id);

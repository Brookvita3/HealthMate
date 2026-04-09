-- medications: parent table
CREATE TABLE medications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    dosage VARCHAR(100) NOT NULL,
    instructions TEXT DEFAULT '',
    prescribed_by VARCHAR(255) DEFAULT '',
    is_active BOOLEAN DEFAULT true,
    frequency JSONB NOT NULL DEFAULT '{}',
    start_date DATE NOT NULL,
    timezone VARCHAR(50) DEFAULT 'UTC',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_medications_user_id ON medications (user_id);
CREATE INDEX idx_medications_active ON medications (user_id) WHERE is_active = true;

-- medication_reminders: child table, CASCADE on medication delete
CREATE TABLE medication_reminders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    medication_id UUID NOT NULL REFERENCES medications(id) ON DELETE CASCADE,
    time VARCHAR(8) NOT NULL,
    is_enabled BOOLEAN DEFAULT true,
    last_taken TIMESTAMPTZ,
    missed_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_reminders_medication_id ON medication_reminders (medication_id);
CREATE INDEX idx_reminders_enabled ON medication_reminders (medication_id) WHERE is_enabled = true;

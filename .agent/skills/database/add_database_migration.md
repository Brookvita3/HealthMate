---
skill: add_database_migration
description: >
  Add a new SQL migration following the project's sequential numbering
  convention used by golang-migrate.
when_to_use:
  - Adding a new table for a new domain entity
  - Adding or removing columns from an existing table
  - Creating indexes, constraints, or enum types
---

## Instructions

### 1. DETERMINE THE NEXT NUMBER
List `migration/` and increment the highest `NNNNNN` prefix found.

### 2. CREATE THE FILE PAIR
```
migration/<NNNNNN>_<snake_case_description>.up.sql
migration/<NNNNNN>_<snake_case_description>.down.sql
```

### 3. WRITE THE UP MIGRATION
Make it idempotent where possible:
- `CREATE TABLE IF NOT EXISTS`
- `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`
- Use named constraints so they can be targeted in `.down.sql`

### 4. WRITE THE DOWN MIGRATION
Fully reverse the `.up.sql`:
- Drop tables, columns, or constraints in reverse order.
- Restore any previous state (e.g. re-add a column that was removed).

### 5. FOLLOW SCHEMA CONVENTIONS
For new tables:
```sql
id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()  -- if needed
```
Foreign keys to `users(id)` or `groups(id)` use `ON DELETE CASCADE` when the child record has no meaning without its parent.

### 6. APPLY AND VERIFY
```bash
# Apply
migrate -path migration -database "$DATABASE_URL" up

# Verify down reverses cleanly
migrate -path migration -database "$DATABASE_URL" down 1
```

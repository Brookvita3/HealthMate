package group

import (
	"auth-service/internal/domain"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// postgresGroup is a helper struct for scanning database rows
type postgresGroup struct {
	ID          uuid.UUID   `db:"id"`
	Name        string      `db:"name"`
	Description pgtype.Text `db:"description"`
	OwnerID     uuid.UUID   `db:"owner_id"`
	CreatedAt   time.Time   `db:"created_at"`
	UpdatedAt   time.Time   `db:"updated_at"`
}

// postgresRepository implements the GroupRepository interface
type postgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository instance
func NewPostgresRepository(pool *pgxpool.Pool) GroupRepository {
	return &postgresRepository{pool: pool}
}

// Create implements GroupRepository.Create
func (r *postgresRepository) Create(ctx context.Context, params CreateGroupParams) (*domain.Group, error) {

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO groups (id, name, description, owner_id, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, NOW(), NOW())
		RETURNING id, name, description, owner_id, created_at, updated_at`

	pGroup := postgresGroup{
		Name:        params.Name,
		Description: pgtype.Text{},
		OwnerID:     params.OwnerID,
	}

	if params.Description != nil {
		pGroup.Description = pgtype.Text{String: *params.Description, Valid: true}
	}

	row := tx.QueryRow(ctx, query,
		pGroup.Name,
		pGroup.Description,
		pGroup.OwnerID,
	)

	var group domain.Group
	err = row.Scan(
		&group.ID,
		&group.Name,
		&pGroup.Description,
		&group.OwnerID,
		&group.CreatedAt,
		&group.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// Handle unique constraint violations
			if pgErr.Code == "23505" {
				return nil, ErrGroupAlreadyExists
			}
		}
		return nil, err
	}

	memberQuery := `
		INSERT INTO group_members (group_id, user_id, invited_by, role, status, joined_at, created_at, updated_at)
		VALUES ($1, $2, $2, 'owner', 'accepted', NOW(), NOW(), NOW())`

	_, err = tx.Exec(ctx, memberQuery, group.ID, group.OwnerID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Convert description if valid
	if pGroup.Description.Valid {
		group.Description = &pGroup.Description.String
	}

	return &group, nil
}

// FindByID implements GroupRepository.FindByID
func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Group, error) {
	query := `
		SELECT id, name, description, owner_id, created_at, updated_at
		FROM groups
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)

	pGroup, err := r.scanGroup(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}

	return r.toDomainGroup(*pGroup), nil
}

// Update implements GroupRepository.Update
func (r *postgresRepository) Update(ctx context.Context, id uuid.UUID, params UpdateGroupParams) error {
	var setClauses []string
	var args []interface{}
	argID := 1

	if params.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argID))
		args = append(args, *params.Name)
		argID++
	}

	if params.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argID))
		args = append(args, *params.Description)
		argID++
	}

	if len(setClauses) == 0 {
		return nil // Nothing to update
	}

	// Add updated_at timestamp
	setClauses = append(setClauses, "updated_at = NOW()")

	// Add WHERE clause
	setClauses = append(setClauses, fmt.Sprintf("WHERE id = $%d", argID))
	args = append(args, id)

	query := fmt.Sprintf("UPDATE groups SET %s", strings.Join(setClauses, ", "))

	cmdTag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return ErrGroupAlreadyExists
			}
		}
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrGroupNotFound
	}

	return nil
}

// Delete implements GroupRepository.Delete
func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM groups WHERE id = $1`

	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrGroupNotFound
	}

	return nil
}

// FindByOwner implements GroupRepository.FindByOwner
func (r *postgresRepository) FindByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]domain.Group, error) {
	query := `
		SELECT id, name, description, owner_id, created_at, updated_at
		FROM groups
		WHERE owner_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, ownerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []domain.Group
	for rows.Next() {
		pGroup, err := r.scanGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, *r.toDomainGroup(*pGroup))
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return groups, nil
}

// FindByUser implements GroupRepository.FindByUser
func (r *postgresRepository) FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Group, error) {
	query := `
		SELECT g.id, g.name, g.description, g.owner_id, g.created_at, g.updated_at
		FROM groups g
		JOIN group_members gm ON g.id = gm.group_id
		WHERE gm.user_id = $1 AND gm.status = 'accepted'
		ORDER BY g.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []domain.Group
	for rows.Next() {
		pGroup, err := r.scanGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, *r.toDomainGroup(*pGroup))
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return groups, nil
}

// List implements GroupRepository.List
func (r *postgresRepository) List(ctx context.Context, params ListGroupsParams) ([]domain.Group, error) {
	query := `
		SELECT id, name, description, owner_id, created_at, updated_at
		FROM groups
		WHERE 1=1`

	var conditions []string
	var args []interface{}
	argID := 1

	// Filter by owner if provided
	if params.OwnerID != nil {
		conditions = append(conditions, fmt.Sprintf("owner_id = $%d", argID))
		args = append(args, *params.OwnerID)
		argID++
	}

	// Search by name if provided
	if params.Search != "" {
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argID))
		args = append(args, "%"+params.Search+"%")
		argID++
	}

	// Add conditions to query
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argID, argID+1)
	args = append(args, params.Limit, params.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []domain.Group
	for rows.Next() {
		pGroup, err := r.scanGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, *r.toDomainGroup(*pGroup))
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return groups, nil
}

// Exists implements GroupRepository.Exists
func (r *postgresRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM groups 
			WHERE id = $1
		)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// FindByName implements GroupRepository.FindByName
func (r *postgresRepository) FindByName(ctx context.Context, name string) (*domain.Group, error) {
	query := `
		SELECT id, name, description, owner_id, created_at, updated_at
		FROM groups
		WHERE name = $1`

	row := r.pool.QueryRow(ctx, query, name)

	pGroup, err := r.scanGroup(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}

	return r.toDomainGroup(*pGroup), nil
}

// TransferOwnership implements GroupRepository.TransferOwnership
func (r *postgresRepository) TransferOwnership(ctx context.Context, groupID, newOwnerID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Get current owner and lock the row
	var oldOwnerID uuid.UUID
	err = tx.QueryRow(ctx, "SELECT owner_id FROM groups WHERE id = $1 FOR UPDATE", groupID).Scan(&oldOwnerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrGroupNotFound
		}
		return err
	}

	// 2. Update groups table
	_, err = tx.Exec(ctx, "UPDATE groups SET owner_id = $1, updated_at = NOW() WHERE id = $2", newOwnerID, groupID)
	if err != nil {
		return err
	}

	// 3. Update old owner's role in group_members to 'member'
	_, err = tx.Exec(ctx, "UPDATE group_members SET role = 'member', updated_at = NOW() WHERE group_id = $1 AND user_id = $2", groupID, oldOwnerID)
	if err != nil {
		return err
	}

	// 4. Update new owner's role in group_members to 'owner'
	cmdTag, err := tx.Exec(ctx, "UPDATE group_members SET role = 'owner', updated_at = NOW() WHERE group_id = $1 AND user_id = $2 AND status = 'accepted'", groupID, newOwnerID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrNotMember
	}

	return tx.Commit(ctx)
}

// =============================================================================
// HELPER METHODS
// =============================================================================

// scannable defines a contract for types that can be scanned by pgx
type scannable interface {
	Scan(dest ...interface{}) error
}

// scanGroup scans a database row into a postgresGroup struct
func (r *postgresRepository) scanGroup(row scannable) (*postgresGroup, error) {
	var g postgresGroup
	err := row.Scan(
		&g.ID,
		&g.Name,
		&g.Description,
		&g.OwnerID,
		&g.CreatedAt,
		&g.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// toDomainGroup converts a postgresGroup to a domain.Group
func (r *postgresRepository) toDomainGroup(pg postgresGroup) *domain.Group {
	group := &domain.Group{
		ID:        pg.ID,
		Name:      pg.Name,
		OwnerID:   pg.OwnerID,
		CreatedAt: pg.CreatedAt,
		UpdatedAt: pg.UpdatedAt,
	}

	if pg.Description.Valid {
		group.Description = &pg.Description.String
	}

	return group
}

package user

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{
		pool: pool,
	}
}

func (r *postgresRepository) CreateUser(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, email, name, picture, role, status, provider, google_id, password)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.Picture,
		user.Role,
		user.Status,
		user.Provider,
		user.GoogleID,
		user.Password,
	)

	return err
}

func (r *postgresRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, name, picture, role, status, provider, google_id, password, created_at, updated_at
		FROM users
		WHERE email = $1`

	row := r.pool.QueryRow(ctx, query, email)

	user, err := r.scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r *postgresRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, email, name, picture, role, status, provider, google_id, password, created_at, updated_at
		FROM users
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)

	user, err := r.scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r *postgresRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `
		UPDATE users
		SET password = $1, updated_at = NOW()
		WHERE id = $2`

	cmdTag, err := r.pool.Exec(ctx, query, passwordHash, id)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *postgresRepository) ListUsers(ctx context.Context, params ListUsersParams) ([]User, error) {
	query := `SELECT id, email, name, picture, role, status, provider, google_id, password, created_at, updated_at
			  FROM users`

	var conditions []string
	var args []any
	argID := 1 // Argument placeholder counter ($1, $2, etc.)

	// Condition 1: Add a status filter IF it's provided in the params.
	if params.Status != "" {
		conditions = append(conditions, "status = $"+strconv.Itoa(argID))
		args = append(args, params.Status)
		argID++
	}

	// Condition 2: Add a search filter if a search term is provided.
	if params.Search != "" {
		condition := "(name ILIKE $" + strconv.Itoa(argID) + " OR email ILIKE $" + strconv.Itoa(argID) + ")"
		conditions = append(conditions, condition)
		args = append(args, "%"+params.Search+"%")
		argID++
	}

	// If there are any conditions, join them with "AND" and append to the main query.
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ordering and pagination.
	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argID) + ` OFFSET $` + strconv.Itoa(argID+1)
	args = append(args, params.Limit)
	args = append(args, params.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		user, err := r.scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *postgresRepository) UpdateUser(ctx context.Context, id uuid.UUID, params UpdateUserParams) error {
	query := `UPDATE users SET name = $1, updated_at = NOW() WHERE id = $2`

	cmdTag, err := r.pool.Exec(ctx, query, params.Name, id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *postgresRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE users SET status = $1, updated_at = NOW() WHERE id = $2`

	cmdTag, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// =============================================================================
// HELPER METHODS
// =============================================================================

// scannable defines a contract for types that can be scanned by pgx,
// such as *pgx.Row and *pgx.Rows. This allows for a reusable scanUser function.
type scannable interface {
	Scan(dest ...any) error
}

// scanUser is a helper function that scans a database row into a User struct.
// It accepts any type that satisfies the scannable interface.
func (r *postgresRepository) scanUser(row scannable) (*User, error) {
	var u User
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.Name,
		&u.Picture,
		&u.Role,
		&u.Status,
		&u.Provider,
		&u.GoogleID,
		&u.Password,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &u, nil
}

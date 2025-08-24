package user

import (
	"context"
	"errors"
	"strconv"

	"github.com/google/uuid"
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

// CreateUser inserts a new user record into the database.
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

// GetUserByEmail searches for a user by email.
// Returns (nil, nil) if not found.
func (r *postgresRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, name, picture, role, status, provider, google_id, password, created_at, updated_at
		FROM users
		WHERE email = $1`

	row := r.pool.QueryRow(ctx, query, email)
	return r.scanUser(row)
}

// GetUserByID searches for a user by ID (UUID).
// Returns (nil, nil) if not found.
func (r *postgresRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, email, name, picture, role, status, provider, google_id, password, created_at, updated_at
		FROM users
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)
	return r.scanUser(row)
}

// UpdatePassword updates the hashed password for the user with the given email.
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
		return errors.New("user not found or password unchanged")
	}

	return nil
}

// ListUsers retrieves a list of users based on filter and pagination parameters.
// It dynamically constructs the query to handle optional searching.
func (r *postgresRepository) ListUsers(ctx context.Context, params ListUsersParams) ([]User, error) {
	query := `SELECT id, email, name, picture, role, status, provider, google_id, password, created_at, updated_at
			  FROM users
			  WHERE status = "active"`
	args := []any{}
	argID := 1

	// Append a WHERE clause if a search term is provided.
	if params.Search != "" {
		query += ` WHERE name ILIKE $1 OR email ILIKE $1`
		args = append(args, "%"+params.Search+"%")
		argID++
	}

	// Add ordering and pagination using the current argument ID.
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
		return errors.New("user not found")
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

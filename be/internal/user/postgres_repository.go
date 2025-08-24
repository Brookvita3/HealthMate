package user

import (
	"context"
	"errors"

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

// Create inserts a new user record into the database.
func (r *postgresRepository) Create(ctx context.Context, user *User) error {
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

// FindByEmail searches for a user by email.
// Returns (nil, nil) if not found.
func (r *postgresRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, name, picture, role, status, provider, google_id, password, created_at, updated_at
		FROM users
		WHERE email = $1`

	row := r.pool.QueryRow(ctx, query, email)
	return r.scanUser(row)
}

// FindByID searches for a user by ID (UUID).
// Returns (nil, nil) if not found.
func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
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

// =====HELPER FUNCTIONS=====

// scanUser is a helper function to scan a result row into a User struct.
func (r *postgresRepository) scanUser(row pgx.Row) (*User, error) {
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &u, nil
}

package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	Create(ctx context.Context, user *User) error
	SetPassword(ctx context.Context, email string, passwordHash string) error
}

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{
		pool: pool,
	}
}

func (r *postgresRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}

	query := `SELECT id, email, name, google_id, password, picture, provider FROM users WHERE email = $1`

	row := r.pool.QueryRow(ctx, query, email)

	err := row.Scan(&user.ID, &user.Email, &user.Name, &user.GoogleID, &user.Password, &user.Picture, &user.Provider)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return user, nil
}

func (r *postgresRepository) FindByID(ctx context.Context, id string) (*User, error) {
	user := &User{}

	query := `SELECT id, email, name, google_id, password, picture, provider FROM users WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)

	err := row.Scan(&user.ID, &user.Email, &user.Name, &user.GoogleID, &user.Password, &user.Picture, &user.Provider)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return user, nil
}

func (r *postgresRepository) Create(ctx context.Context, user *User) error {
	query := `INSERT INTO users (id, email, name, google_id, password, picture, provider) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query, user.ID, user.Email, user.Name, user.GoogleID, user.Password, user.Picture, user.Provider)

	if err != nil {
		return err
	}

	return nil
}

func (r *postgresRepository) SetPassword(ctx context.Context, email string, passwordHash string) error {
	query := `UPDATE users SET password = $1 WHERE email = $2`

	commandTag, err := r.pool.Exec(ctx, query, passwordHash, email)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return errors.New("user not found")
	}

	return nil
}

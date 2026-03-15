package user

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"auth-service/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresUser struct {
	Id      uuid.UUID   `db:"id"`
	Email   string      `db:"email"`
	Name    string      `db:"name"`
	Picture pgtype.Text `db:"picture"`
	Role    string      `db:"role"`
	Status  string      `db:"status"`

	Phone      pgtype.Text    `db:"phone"`
	Address    pgtype.Text    `db:"address"`
	Gender     pgtype.Text    `db:"gender"`
	Birthday   pgtype.Date    `db:"birthday"`
	Weight     pgtype.Float8  `db:"weight"`
	Height     pgtype.Float8  `db:"height"`
	BloodGroup pgtype.Text    `db:"blood_group"`

	Provider string      `db:"provider"`
	GoogleID pgtype.Text `db:"google_id"`
	Password pgtype.Text `db:"password"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) UserRepository {
	return &postgresRepository{
		pool: pool,
	}
}

func (r *postgresRepository) CreateUser(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, name, picture, role, status, provider, google_id, password, phone, address, gender, birthday, weight, height, blood_group)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`
	pUser := fromDomainUser(*user)
	_, err := r.pool.Exec(ctx, query,
		pUser.Id,
		pUser.Email,
		pUser.Name,
		pUser.Picture,
		pUser.Role,
		pUser.Status,
		pUser.Provider,
		pUser.GoogleID,
		pUser.Password,
		pUser.Phone,
		pUser.Address,
		pUser.Gender,
		pUser.Birthday,
		pUser.Weight,
		pUser.Height,
		pUser.BloodGroup,
	)

	return err
}

func (r *postgresRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, name, picture, role, status, provider, google_id, password, phone, address, gender, birthday, weight, height, blood_group, created_at, updated_at
		FROM users
		WHERE email = $1`

	row := r.pool.QueryRow(ctx, query, email)

	pUser, err := r.scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toDomainUser(*pUser), nil
}

func (r *postgresRepository) GetUserById(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, name, picture, role, status, provider, google_id, password, phone, address, gender, birthday, weight, height, blood_group, created_at, updated_at
		FROM users
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)

	pUser, err := r.scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toDomainUser(*pUser), nil
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

func (r *postgresRepository) ListUsers(ctx context.Context, params ListUsersParams) ([]domain.User, error) {
	query := `SELECT id, email, name, picture, role, status, provider, google_id, password, phone, address, gender, birthday, weight, height, blood_group, created_at, updated_at
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

	var users []domain.User
	for rows.Next() {
		pUser, err := r.scanUser(rows)
		if err != nil {
			return nil, err
		}
		user := toDomainUser(*pUser)
		users = append(users, *user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *postgresRepository) UpdateUser(ctx context.Context, id uuid.UUID, params UpdateUserParams) error {
	var setClauses []string
	var args []any
	argId := 1

	if params.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argId))
		args = append(args, *params.Name)
		argId++
	}
	if params.Picture != nil {
		setClauses = append(setClauses, fmt.Sprintf("picture = $%d", argId))
		args = append(args, *params.Picture)
		argId++
	}
	if params.Role != nil {
		setClauses = append(setClauses, fmt.Sprintf("role = $%d", argId))
		args = append(args, *params.Role)
		argId++
	}
	if params.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argId))
		args = append(args, *params.Status)
		argId++
	}
	if params.Phone != nil {
		setClauses = append(setClauses, fmt.Sprintf("phone = $%d", argId))
		args = append(args, *params.Phone)
		argId++
	}
	if params.Address != nil {
		setClauses = append(setClauses, fmt.Sprintf("address = $%d", argId))
		args = append(args, *params.Address)
		argId++
	}
	if params.Gender != nil {
		setClauses = append(setClauses, fmt.Sprintf("gender = $%d", argId))
		args = append(args, *params.Gender)
		argId++
	}
	if params.Birthday != nil {
		if *params.Birthday == "" {
			setClauses = append(setClauses, "birthday = NULL")
		} else {
			setClauses = append(setClauses, fmt.Sprintf("birthday = $%d", argId))
			args = append(args, *params.Birthday)
			argId++
		}
	}
	if params.Weight != nil {
		setClauses = append(setClauses, fmt.Sprintf("weight = $%d", argId))
		args = append(args, *params.Weight)
		argId++
	}
	if params.Height != nil {
		setClauses = append(setClauses, fmt.Sprintf("height = $%d", argId))
		args = append(args, *params.Height)
		argId++
	}
	if params.BloodGroup != nil {
		setClauses = append(setClauses, fmt.Sprintf("blood_group = $%d", argId))
		args = append(args, *params.BloodGroup)
		argId++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf(`UPDATE users SET %s, updated_at = NOW() WHERE id = $%d`,
		strings.Join(setClauses, ", "), argId)

	args = append(args, id)

	cmdTag, err := r.pool.Exec(ctx, query, args...)
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

func (r *postgresRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`

	row := r.pool.QueryRow(ctx, query, id)

	var exists bool
	err := row.Scan(&exists)
	if err != nil {
		return false, ErrUserNotFound
	}
	return exists, nil
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
func (r *postgresRepository) scanUser(row scannable) (*postgresUser, error) {
	var u postgresUser
	err := row.Scan(
		&u.Id,
		&u.Email,
		&u.Name,
		&u.Picture,
		&u.Role,
		&u.Status,
		&u.Provider,
		&u.GoogleID,
		&u.Password,
		&u.Phone,
		&u.Address,
		&u.Gender,
		&u.Birthday,
		&u.Weight,
		&u.Height,
		&u.BloodGroup,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &u, nil
}

// toDomainUser is a helper function that converts a postgresUser struct to a domain User struct.
func toDomainUser(p postgresUser) *domain.User {
	u := &domain.User{
		Id:        p.Id,
		Email:     p.Email,
		Name:      p.Name,
		Role:      p.Role,
		Status:    p.Status,
		Provider:  p.Provider,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}

	if p.Picture.Valid {
		u.Picture = p.Picture.String
	}
	if p.GoogleID.Valid {
		u.GoogleID = p.GoogleID.String
	}
	if p.Password.Valid {
		u.Password = p.Password.String
	}
	if p.Phone.Valid {
		u.Phone = p.Phone.String
	}
	if p.Address.Valid {
		u.Address = p.Address.String
	}
	if p.Gender.Valid {
		u.Gender = p.Gender.String
	}
	if p.Birthday.Valid {
		u.Birthday = p.Birthday.Time.Format(time.DateOnly)
	}
	if p.Weight.Valid {
		u.Weight = p.Weight.Float64
	}
	if p.Height.Valid {
		u.Height = p.Height.Float64
	}
	if p.BloodGroup.Valid {
		u.BloodGroup = p.BloodGroup.String
	}

	return u
}

// fromDomainUser is a helper function that converts a domain User struct to a postgresUser struct.
func fromDomainUser(u domain.User) *postgresUser {
	p := &postgresUser{
		Id:        u.Id,
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		Status:    u.Status,
		Provider:  u.Provider,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}

	if u.Picture != "" {
		p.Picture = pgtype.Text{String: u.Picture, Valid: true}
	}

	if u.GoogleID != "" {
		p.GoogleID = pgtype.Text{String: u.GoogleID, Valid: true}
	}

	if u.Password != "" {
		p.Password = pgtype.Text{String: u.Password, Valid: true}
	}

	if u.Phone != "" {
		p.Phone = pgtype.Text{String: u.Phone, Valid: true}
	}

	if u.Address != "" {
		p.Address = pgtype.Text{String: u.Address, Valid: true}
	}

	if u.Gender != "" {
		p.Gender = pgtype.Text{String: u.Gender, Valid: true}
	}

	if u.Birthday != "" {
		// Try to parse as RFC3339 first, then DateOnly
		if t, err := time.Parse(time.RFC3339, u.Birthday); err == nil {
			p.Birthday = pgtype.Date{Time: t, Valid: true}
		} else if t, err := time.Parse(time.DateOnly, u.Birthday); err == nil {
			p.Birthday = pgtype.Date{Time: t, Valid: true}
		}
	}

	if u.Weight != 0 {
		p.Weight = pgtype.Float8{Float64: u.Weight, Valid: true}
	}

	if u.Height != 0 {
		p.Height = pgtype.Float8{Float64: u.Height, Valid: true}
	}

	if u.BloodGroup != "" {
		p.BloodGroup = pgtype.Text{String: u.BloodGroup, Valid: true}
	}

	return p
}

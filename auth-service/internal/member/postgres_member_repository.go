package member

import (
	"auth-service/internal/domain"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) MemberRepository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) AddMember(ctx context.Context, groupID, userID, invitedBy uuid.UUID, role string) error {
	query := `
		INSERT INTO group_members (group_id, user_id, invited_by, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'pending', NOW(), NOW())
		ON CONFLICT (group_id, user_id) DO NOTHING`
	_, err := r.pool.Exec(ctx, query, groupID, userID, invitedBy, role)

	return err
}

func (r *postgresRepository) UpdateMemberStatus(ctx context.Context, groupID, userID uuid.UUID, status string) error {
	var joinedAt interface{}
	if status == "accepted" {
		joinedAt = time.Now()
	} else {
		joinedAt = nil
	}

	query := `
		UPDATE group_members 
		SET status = $1, joined_at = $2, updated_at = NOW()
		WHERE group_id = $3 AND user_id = $4`
	_, err := r.pool.Exec(ctx, query, status, joinedAt, groupID, userID)
	return err
}

func (r *postgresRepository) RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error {
	query := `DELETE FROM group_members WHERE group_id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, query, groupID, userID)
	return err
}

func (r *postgresRepository) GetMember(ctx context.Context, groupID, userID uuid.UUID) (*domain.GroupMember, error) {
	query := `
		SELECT group_id, user_id, role, status, invited_by, joined_at, created_at, updated_at
		FROM group_members
		WHERE group_id = $1 AND user_id = $2`

	var m domain.GroupMember
	err := r.pool.QueryRow(ctx, query, groupID, userID).Scan(
		&m.GroupID, &m.UserID, &m.Role, &m.Status, &m.InvitedBy, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (r *postgresRepository) ListGroupMembers(ctx context.Context, groupID uuid.UUID) ([]domain.GroupMember, error) {
	query := `
		SELECT group_id, user_id, role, status, invited_by, joined_at, created_at, updated_at
		FROM group_members
		WHERE group_id = $1`

	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []domain.GroupMember
	for rows.Next() {
		var m domain.GroupMember
		err := rows.Scan(
			&m.GroupID, &m.UserID, &m.Role, &m.Status, &m.InvitedBy, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

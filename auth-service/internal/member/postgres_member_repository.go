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
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Delete shared permissions first to avoid foreign key violation
	_, err = tx.Exec(ctx, `DELETE FROM sharing_permissions WHERE group_id = $1 AND user_id = $2`, groupID, userID)
	if err != nil {
		return err
	}

	// 2. Delete group member
	query := `DELETE FROM group_members WHERE group_id = $1 AND user_id = $2`
	cmdTag, err := tx.Exec(ctx, query, groupID, userID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrMemberNotFound
	}

	return tx.Commit(ctx)
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
		WHERE group_id = $1 AND status = 'accepted'`

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

func (r *postgresRepository) GroupExists(ctx context.Context, groupID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, groupID).Scan(&exists)
	return exists, err
}

func (r *postgresRepository) IsOwner(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1 AND owner_id = $2)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, groupID, userID).Scan(&exists)
	return exists, err
}

func (r *postgresRepository) GetUserInvitations(ctx context.Context, userID uuid.UUID) ([]domain.InvitationResponse, error) {
	query := `
		SELECT 
			gm.group_id, 
			gm.group_id as id,
			gm.created_at as sent_at,
			g.id as group_id_info, 
			g.name as group_name, 
			(SELECT COUNT(*) FROM group_members WHERE group_id = g.id AND status = 'accepted') as member_count,
			u.id as inviter_id, 
			u.name as inviter_name, 
			u.email as inviter_email
		FROM group_members gm
		JOIN groups g ON gm.group_id = g.id
		JOIN users u ON gm.invited_by = u.id
		WHERE gm.user_id = $1 AND gm.status = 'pending'`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []domain.InvitationResponse
	for rows.Next() {
		var inv domain.InvitationResponse
		var groupInfo domain.GroupInfo
		var inviterInfo domain.UserInfo

		err := rows.Scan(
			&inv.GroupID,
			&inv.ID,
			&inv.SentAt,
			&groupInfo.ID,
			&groupInfo.Name,
			&groupInfo.MemberCount,
			&inviterInfo.ID,
			&inviterInfo.Name,
			&inviterInfo.Email,
		)
		if err != nil {
			return nil, err
		}
		inv.Group = groupInfo
		inv.Inviter = inviterInfo
		inv.MemberCount = groupInfo.MemberCount
		invitations = append(invitations, inv)
	}

	if invitations == nil {
		invitations = []domain.InvitationResponse{}
	}

	return invitations, nil
}

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
		SELECT gm.user_id,
		       COALESCE(
		         NULLIF(TRIM(u.name), ''),
		         NULLIF(SPLIT_PART(COALESCE(u.email, ''), '@', 1), ''),
		         ''
		       ),
		       COALESCE(u.email, ''),
		       gm.role, gm.status, gm.invited_by, gm.joined_at, gm.created_at, gm.updated_at
		FROM group_members gm
		INNER JOIN users u ON u.id = gm.user_id
		WHERE gm.group_id = $1 AND gm.user_id = $2`

	var m domain.GroupMember
	err := r.pool.QueryRow(ctx, query, groupID, userID).Scan(
		&m.UserID, &m.Name, &m.Email, &m.Role, &m.Status, &m.InvitedBy, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt,
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
		SELECT gm.user_id,
		       COALESCE(
		         NULLIF(TRIM(u.name), ''),
		         NULLIF(SPLIT_PART(COALESCE(u.email, ''), '@', 1), ''),
		         ''
		       ),
		       COALESCE(u.email, ''),
		       gm.role, gm.status, gm.invited_by, gm.joined_at, gm.created_at, gm.updated_at
		FROM group_members gm
		INNER JOIN users u ON u.id = gm.user_id
		WHERE gm.group_id = $1 AND gm.status = 'accepted'`

	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []domain.GroupMember
	for rows.Next() {
		var m domain.GroupMember
		err := rows.Scan(
			&m.UserID, &m.Name, &m.Email, &m.Role, &m.Status, &m.InvitedBy, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt,
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
			gm.created_at as sent_at,
			g.id as group_id_info, 
			g.name as group_name, 
			(SELECT COUNT(*) FROM group_members WHERE group_id = g.id AND status = 'accepted') as member_count,
			u.id as inviter_id, 
			u.name as inviter_name, 
			u.email as inviter_email,
			uto.id as invite_to_id,
			uto.name as invite_to_name,
			uto.email as invite_to_email
		FROM group_members gm
		JOIN groups g ON gm.group_id = g.id
		JOIN users u ON gm.invited_by = u.id
		JOIN users uto ON gm.user_id = uto.id
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
		var inviteTo domain.UserInfo

		err := rows.Scan(
			&inv.ID,
			&inv.SentAt,
			&groupInfo.ID,
			&groupInfo.Name,
			&groupInfo.MemberCount,
			&inviterInfo.ID,
			&inviterInfo.Name,
			&inviterInfo.Email,
			&inviteTo.ID,
			&inviteTo.Name,
			&inviteTo.Email,
		)
		if err != nil {
			return nil, err
		}
		inv.Group = groupInfo
		inv.Inviter = inviterInfo
		inv.InviteTo = inviteTo
		inv.MemberCount = groupInfo.MemberCount
		invitations = append(invitations, inv)
	}

	if invitations == nil {
		invitations = []domain.InvitationResponse{}
	}

	return invitations, nil
}

func (r *postgresRepository) GetGroupInvitations(ctx context.Context, groupID uuid.UUID) ([]domain.SentInvitationResponse, error) {
	query := `
		SELECT 
			gm.group_id, 
			gm.user_id, 
			u.email as user_email, 
			u.name as user_name, 
			gm.invited_by, 
			inv.name as inviter_name, 
			gm.status, 
			gm.created_at
		FROM group_members gm
		JOIN users u ON gm.user_id = u.id
		JOIN users inv ON gm.invited_by = inv.id
		WHERE gm.group_id = $1 AND gm.status IN ('pending', 'pending_owner_approval')
		ORDER BY gm.created_at DESC`

	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []domain.SentInvitationResponse
	for rows.Next() {
		var inv domain.SentInvitationResponse
		err := rows.Scan(
			&inv.GroupID,
			&inv.UserID,
			&inv.UserEmail,
			&inv.UserName,
			&inv.InvitedBy,
			&inv.InviterName,
			&inv.Status,
			&inv.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		invitations = append(invitations, inv)
	}

	if invitations == nil {
		invitations = []domain.SentInvitationResponse{}
	}

	return invitations, nil
}

func (r *postgresRepository) GetPendingApprovals(ctx context.Context, groupID uuid.UUID) ([]domain.SentInvitationResponse, error) {
	query := `
		SELECT
			gm.group_id,
			gm.user_id,
			u.email as user_email,
			u.name as user_name,
			gm.invited_by,
			inv.name as inviter_name,
			gm.status,
			gm.created_at
		FROM group_members gm
		JOIN users u ON gm.user_id = u.id
		JOIN users inv ON gm.invited_by = inv.id
		WHERE gm.group_id = $1 AND gm.status = 'pending_owner_approval'
		ORDER BY gm.created_at DESC`

	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var approvals []domain.SentInvitationResponse
	for rows.Next() {
		var inv domain.SentInvitationResponse
		err := rows.Scan(
			&inv.GroupID,
			&inv.UserID,
			&inv.UserEmail,
			&inv.UserName,
			&inv.InvitedBy,
			&inv.InviterName,
			&inv.Status,
			&inv.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, inv)
	}

	if approvals == nil {
		approvals = []domain.SentInvitationResponse{}
	}

	return approvals, nil
}

func (r *postgresRepository) CountMembers(ctx context.Context, groupID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM group_members WHERE group_id = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, groupID).Scan(&count)
	return count, err
}

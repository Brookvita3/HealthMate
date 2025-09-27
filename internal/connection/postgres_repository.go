package connection

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
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) Create(ctx context.Context, conn *Connection) error {
	u1, u2 := orderUserIDs(conn.UserOneId, conn.UserTwoId)

	query := `INSERT INTO connections (user_one_id, user_two_id, requester_id, status)
			  VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(ctx, query, u1, u2, conn.RequesterId, conn.Status)

	return err
}

func (r *postgresRepository) UpdateStatus(ctx context.Context, userOneID, userTwoID uuid.UUID, status Status) error {
	u1, u2 := orderUserIDs(userOneID, userTwoID)

	query := `UPDATE connections SET status = $3, updated_at = NOW()
			  WHERE user_one_id = $1 AND user_two_id = $2`
	commandTag, err := r.pool.Exec(ctx, query, u1, u2, status)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrConnectionNotFound
	}

	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, userOneID, userTwoID uuid.UUID) error {
	u1, u2 := orderUserIDs(userOneID, userTwoID)

	query := `DELETE FROM connections WHERE user_one_id = $1 AND user_two_id = $2`
	_, err := r.pool.Exec(ctx, query, u1, u2)

	return err
}

func (r *postgresRepository) GetConnectionByPair(ctx context.Context, userID1, userID2 uuid.UUID) (*Connection, error) {
	u1, u2 := orderUserIDs(userID1, userID2)
	query := `SELECT user_one_id, user_two_id, requester_id, status, created_at, updated_at
			  FROM connections
			  WHERE user_one_id = $1 AND user_two_id = $2`

	row := r.pool.QueryRow(ctx, query, u1, u2)

	conn, err := r.scanConnection(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrConnectionNotFound
		}
		return nil, err
	}

	return conn, nil
}

func (r *postgresRepository) ListPendingConnectionsForUser(ctx context.Context, userID uuid.UUID) ([]Connection, error) {
	var connections []Connection
	query := `SELECT user_one_id, user_two_id, requester_id, status, created_at, updated_at
			  FROM connections
			  WHERE (user_one_id = $1 OR user_two_id = $1)
			  AND status = 'pending'
			  AND requester_id != $1`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		conn, err := r.scanConnection(rows)
		if err != nil {
			return nil, err
		}
		connections = append(connections, *conn)
	}

	return connections, nil
}

func (r *postgresRepository) ListAcceptedConnectionsForUser(ctx context.Context, userID uuid.UUID) ([]Connection, error) {
	var connections []Connection
	query := `SELECT user_one_id, user_two_id, requester_id, status, created_at, updated_at
			  FROM connections
			  WHERE (user_one_id = $1 OR user_two_id = $1) AND status = 'accepted'`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		conn, err := r.scanConnection(rows)
		if err != nil {
			return nil, err
		}
		connections = append(connections, *conn)
	}

	return connections, nil
}

// =============================================================================
// HELPER METHODS
// =============================================================================

// orderUserIDs is a helper function to ensure that userOneID is always smaller than userTwoID.
func orderUserIDs(id1, id2 uuid.UUID) (uuid.UUID, uuid.UUID) {
	if id1.String() < id2.String() {
		return id1, id2
	}
	return id2, id1
}

// scannable defines a contract for types that can be scanned by pgx,
// such as *pgx.Row and *pgx.Rows. This allows for a reusable scanUser function.
type scannable interface {
	Scan(dest ...any) error
}

// scanConnection is a helper function that scans a database row into a Connection struct.
// It accepts any type that satisfies the scannable interface.
func (r *postgresRepository) scanConnection(row scannable) (*Connection, error) {
	var conn Connection
	err := row.Scan(
		&conn.UserOneId,
		&conn.UserTwoId,
		&conn.RequesterId,
		&conn.Status,
		&conn.CreatedAt,
		&conn.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &conn, nil
}

package connection

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	// Create creates a new connection record (usually with status 'pending').
	Create(ctx context.Context, conn *Connection) error

	// UpdateStatus updates the status of a connection (e.g., from 'pending' to 'accepted').
	// Return ErrConnectionNotFound if no connection is found.
	UpdateStatus(ctx context.Context, userOneID, userTwoID uuid.UUID, status Status) error

	// Delete removes a connection from the database.
	Delete(ctx context.Context, userOneID, userTwoID uuid.UUID) error

	// FindByPair finds a unique connection between two users.
	// Return ErrConnectionNotFound if no connection is found.
	GetConnectionByPair(ctx context.Context, userID1, userID2 uuid.UUID) (*Connection, error)

	// FindPendingForUser finds all connections that are waiting for this user to accept.
	ListPendingConnectionsForUser(ctx context.Context, userID uuid.UUID) ([]Connection, error)

	// FindAcceptedForUser finds all accepted connections for a user.
	ListAcceptedConnectionsForUser(ctx context.Context, userID uuid.UUID) ([]Connection, error)
}

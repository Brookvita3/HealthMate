package group

import (
	"auth-service/internal/domain"
	"context"
	"github.com/google/uuid"
)

// CreateGroupParams contains parameters for creating a new group
type CreateGroupParams struct {
	Name        string
	Description *string
	OwnerID     uuid.UUID
}

// UpdateGroupParams contains parameters for updating a group
type UpdateGroupParams struct {
	Name        *string
	Description *string
}

// ListGroupsParams contains parameters for filtering and paginating groups
type ListGroupsParams struct {
	OwnerID *uuid.UUID // Filter by owner
	Limit   int        // Pagination limit
	Offset  int        // Pagination offset
	Search  string     // Search by name
}

// GroupRepository defines the interface for group data operations
type GroupRepository interface {
	// Create creates a new group in the database
	Create(ctx context.Context, params CreateGroupParams) (*domain.Group, error)

	// FindByID retrieves a group by its ID
	// Returns ErrGroupNotFound if no group is found
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Group, error)

	// Update updates an existing group
	// Returns ErrGroupNotFound if no group is found
	Update(ctx context.Context, id uuid.UUID, params UpdateGroupParams) error

	// Delete removes a group by its ID
	// Returns ErrGroupNotFound if no group is found
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByOwner retrieves all groups owned by a user
	FindByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]domain.Group, error)

	// List retrieves groups with filtering and pagination
	List(ctx context.Context, params ListGroupsParams) ([]domain.Group, error)

	// Exists checks if a group exists by ID
	Exists(ctx context.Context, id uuid.UUID) (bool, error)

	// TransferOwnership transfers group ownership to another user
	// Returns ErrGroupNotFound if no group is found
	TransferOwnership(ctx context.Context, groupID, newOwnerID uuid.UUID) error
}

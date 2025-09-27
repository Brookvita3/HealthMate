package connection

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"healthmate/internal/user"
)

type Service interface {
	// CreateRequest handles the logic for sending a connection request from requesterID to receiverID.
	// If the request is valid, this function creates a connection request between two users.
	CreateRequest(ctx context.Context, requesterID, receiverID uuid.UUID) error

	// UpdateConnectionStatus updates the connection status between two users.
	// `actorID` is the user performing the action (e.g., the one accepting the connection request).
	// `status` is the new status of the connection, e.g., "accepted" or "rejected".
	UpdateConnectionStatus(ctx context.Context, actorID, targetUserID uuid.UUID, status Status) error

	// GetConnectionByPair retrieves the connection information between two users.
	// This function returns the connection info between userID1 and userID2 if it exists.
	GetConnectionByPair(ctx context.Context, userID1, userID2 uuid.UUID) (*Connection, error)

	// DeleteConnection handles the removal of a connection between actorID and targetUserID.
	// This function deletes the connection between two users if they have an active connection.
	DeleteConnection(ctx context.Context, actorID, targetUserID uuid.UUID) error
}

type service struct {
	connRepo Repository
	userRepo user.Repository
}

func NewService(connRepo Repository, userRepo user.Repository) Service {
	return &service{
		connRepo: connRepo,
		userRepo: userRepo,
	}
}

func (s *service) CreateRequest(ctx context.Context, requesterId, receiverId uuid.UUID) error {
	if requesterId == receiverId {
		return ErrInvalidRequest
	}

	_, err := s.userRepo.GetUserById(ctx, receiverId)
	if err != nil {
		return err
	}

	existing, err := s.connRepo.GetConnectionByPair(ctx, requesterId, receiverId)
	if err != nil && !errors.Is(err, ErrConnectionNotFound) {
		return err
	}
	if existing != nil {
		return ErrConnectionExists
	}

	u1, u2 := orderUserIDs(requesterId, receiverId)
	newConn := &Connection{
		UserOneId:   u1,
		UserTwoId:   u2,
		RequesterId: requesterId,
		Status:      StatusPending,
	}

	return s.connRepo.Create(ctx, newConn)
}

func (s *service) UpdateConnectionStatus(ctx context.Context, acceptorId, requesterId uuid.UUID, status Status) error {
	if status != StatusAccepted {
		return ErrInvalidStatus
	}

	conn, err := s.connRepo.GetConnectionByPair(ctx, acceptorId, requesterId)
	if err != nil {
		return err
	}

	if conn.RequesterId == acceptorId {
		return ErrInvalidRequest
	}

	if conn.Status != StatusPending {
		return ErrInvalidStatus
	}

	u1, u2 := orderUserIDs(acceptorId, requesterId)
	return s.connRepo.UpdateStatus(ctx, u1, u2, status)
}

func (s *service) GetConnectionByPair(ctx context.Context, userID1, userID2 uuid.UUID) (*Connection, error) {
	return s.connRepo.GetConnectionByPair(ctx, userID1, userID2)
}

func (s *service) DeleteConnection(ctx context.Context, actorID, targetUserID uuid.UUID) error {
	_, err := s.connRepo.GetConnectionByPair(ctx, actorID, targetUserID)
	if err != nil {
		return err
	}

	u1, u2 := orderUserIDs(actorID, targetUserID)
	return s.connRepo.Delete(ctx, u1, u2)
}

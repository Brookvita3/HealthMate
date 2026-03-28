package group_test

import (
	"context"
	"errors"
	"testing"

	"auth-service/internal/domain"
	"auth-service/internal/group"
	"auth-service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testEnv struct {
	GroupRepository  *mocks.GroupRepository
	UserRepository   *mocks.UserRepository
	MemberRepository *mocks.MemberRepository
	Service          group.Service
}

func setupTest() *testEnv {
	mockGroupRepo := new(mocks.GroupRepository)
	mockUserRepo := new(mocks.UserRepository)
	mockMemberRepo := new(mocks.MemberRepository)

	service := group.NewService(mockGroupRepo, mockUserRepo, mockMemberRepo)

	return &testEnv{
		GroupRepository:  mockGroupRepo,
		UserRepository:   mockUserRepo,
		MemberRepository: mockMemberRepo,
		Service:          service,
	}
}

func TestCreateGroup(t *testing.T) {
	t.Run("Create success", func(t *testing.T) {
		env := setupTest()

		ownerID := uuid.New()
		name := "Test Group"
		description := "Test Description"
		mockGroup := &domain.Group{
			ID:          uuid.New(),
			Name:        name,
			Description: &description,
			OwnerID:     ownerID,
		}

		env.UserRepository.On("Exists", mock.Anything, ownerID).Return(true, nil).Once()
		env.GroupRepository.On("FindByNameAndOwner", mock.Anything, name, ownerID).Return(nil, group.ErrGroupNotFound).Once()
		env.GroupRepository.On("Create", mock.Anything, group.CreateGroupParams{
			Name:        name,
			Description: &description,
			OwnerID:     ownerID,
		}).Return(mockGroup, nil).Once()

		result, err := env.Service.CreateGroup(context.Background(), ownerID, name, &description)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, name, result.Name)
		assert.Equal(t, ownerID, result.OwnerID)

		env.UserRepository.AssertExpectations(t)
		env.GroupRepository.AssertExpectations(t)
	})

	t.Run("Create fail: Invalid group name (too short)", func(t *testing.T) {
		env := setupTest()

		ownerID := uuid.New()
		name := "Te"

		result, err := env.Service.CreateGroup(context.Background(), ownerID, name, nil)

		assert.Error(t, err)
		assert.Equal(t, group.ErrInvalidGroupName, err)
		assert.Nil(t, result)
	})

	t.Run("Create fail: Owner does not exist", func(t *testing.T) {
		env := setupTest()

		ownerID := uuid.New()
		name := "Test Group"

		env.UserRepository.On("Exists", mock.Anything, ownerID).Return(false, nil).Once()

		result, err := env.Service.CreateGroup(context.Background(), ownerID, name, nil)

		assert.Error(t, err)
		assert.Equal(t, group.ErrNotGroupOwner, err)
		assert.Nil(t, result)

		env.UserRepository.AssertExpectations(t)
	})

	t.Run("Create fail: Repository error", func(t *testing.T) {
		env := setupTest()

		ownerID := uuid.New()
		name := "Test Group"
		dbError := errors.New("database error")

		env.UserRepository.On("Exists", mock.Anything, ownerID).Return(true, nil).Once()
		env.GroupRepository.On("FindByNameAndOwner", mock.Anything, name, ownerID).Return(nil, group.ErrGroupNotFound).Once()
		env.GroupRepository.On("Create", mock.Anything, mock.Anything).Return(nil, dbError).Once()

		result, err := env.Service.CreateGroup(context.Background(), ownerID, name, nil)

		assert.Error(t, err)
		assert.Equal(t, dbError, err)
		assert.Nil(t, result)

		env.UserRepository.AssertExpectations(t)
		env.GroupRepository.AssertExpectations(t)
	})
}

func TestGetGroup(t *testing.T) {
	t.Run("Get success", func(t *testing.T) {
		env := setupTest()

		groupID := uuid.New()
		mockGroup := &domain.Group{ID: groupID, Name: "Test Group"}

		env.GroupRepository.On("FindByID", mock.Anything, groupID).Return(mockGroup, nil).Once()

		result, err := env.Service.GetGroup(context.Background(), groupID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, groupID, result.ID)

		env.GroupRepository.AssertExpectations(t)
	})

	t.Run("Get fail: Group not found", func(t *testing.T) {
		env := setupTest()

		groupID := uuid.New()

		env.GroupRepository.On("FindByID", mock.Anything, groupID).Return(nil, group.ErrGroupNotFound).Once()

		result, err := env.Service.GetGroup(context.Background(), groupID)

		assert.Error(t, err)
		assert.Equal(t, group.ErrGroupNotFound, err)
		assert.Nil(t, result)

		env.GroupRepository.AssertExpectations(t)
	})
}

func TestUpdateGroup(t *testing.T) {
	t.Run("Update success", func(t *testing.T) {
		env := setupTest()

		groupID := uuid.New()
		requesterID := uuid.New()
		newName := "Updated Name"
		newDescription := "Updated Description"

		env.GroupRepository.On("FindByID", mock.Anything, groupID).Return(&domain.Group{ID: groupID, OwnerID: requesterID}, nil).Once()
		env.GroupRepository.On("FindByNameAndOwner", mock.Anything, newName, requesterID).Return(nil, group.ErrGroupNotFound).Once()
		env.GroupRepository.On("Update", mock.Anything, groupID, group.UpdateGroupParams{
			Name:        &newName,
			Description: &newDescription,
		}).Return(nil).Once()

		err := env.Service.UpdateGroup(context.Background(), groupID, &newName, &newDescription, requesterID)

		assert.NoError(t, err)

		env.GroupRepository.AssertExpectations(t)
	})

	t.Run("Update fail: Not group owner", func(t *testing.T) {
		env := setupTest()

		groupID := uuid.New()
		requesterID := uuid.New()
		otherUserID := uuid.New()
		newName := "Updated Name"

		env.GroupRepository.On("FindByID", mock.Anything, groupID).Return(&domain.Group{ID: groupID, OwnerID: otherUserID}, nil).Once()

		err := env.Service.UpdateGroup(context.Background(), groupID, &newName, nil, requesterID)

		assert.Error(t, err)
		assert.Equal(t, group.ErrNotGroupOwner, err)

		env.GroupRepository.AssertExpectations(t)
	})

	t.Run("Update fail: Invalid group name", func(t *testing.T) {
		env := setupTest()

		groupID := uuid.New()
		requesterID := uuid.New()
		invalidName := "Ab"

		env.GroupRepository.On("FindByID", mock.Anything, groupID).Return(&domain.Group{ID: groupID, OwnerID: requesterID}, nil).Once()

		err := env.Service.UpdateGroup(context.Background(), groupID, &invalidName, nil, requesterID)

		assert.Error(t, err)
		assert.Equal(t, group.ErrInvalidGroupName, err)

		env.GroupRepository.AssertExpectations(t)
	})
}

func TestDeleteGroup(t *testing.T) {
	t.Run("Delete success", func(t *testing.T) {
		env := setupTest()

		groupID := uuid.New()
		requesterID := uuid.New()

		env.GroupRepository.On("FindByID", mock.Anything, groupID).Return(&domain.Group{ID: groupID, OwnerID: requesterID}, nil).Once()
		env.MemberRepository.On("CountMembers", mock.Anything, groupID).Return(1, nil).Once()
		env.GroupRepository.On("Delete", mock.Anything, groupID).Return(nil).Once()

		err := env.Service.DeleteGroup(context.Background(), groupID, requesterID)

		assert.NoError(t, err)

		env.GroupRepository.AssertExpectations(t)
	})

	t.Run("Delete fail: Not group owner", func(t *testing.T) {
		env := setupTest()

		groupID := uuid.New()
		requesterID := uuid.New()
		otherUserID := uuid.New()

		env.GroupRepository.On("FindByID", mock.Anything, groupID).Return(&domain.Group{ID: groupID, OwnerID: otherUserID}, nil).Once()

		err := env.Service.DeleteGroup(context.Background(), groupID, requesterID)

		assert.Error(t, err)
		assert.Equal(t, group.ErrNotGroupOwner, err)

		env.GroupRepository.AssertExpectations(t)
	})
}

func TestTransferOwnership(t *testing.T) {
	t.Run("Transfer success", func(t *testing.T) {
		env := setupTest()

		groupID := uuid.New()
		currentOwnerID := uuid.New()
		newOwnerID := uuid.New()

		env.GroupRepository.On("FindByID", mock.Anything, groupID).Return(&domain.Group{ID: groupID, OwnerID: currentOwnerID}, nil).Once()
		env.UserRepository.On("Exists", mock.Anything, newOwnerID).Return(true, nil).Once()
		env.GroupRepository.On("TransferOwnership", mock.Anything, groupID, newOwnerID).Return(nil).Once()

		err := env.Service.TransferOwnership(context.Background(), groupID, currentOwnerID, newOwnerID)

		assert.NoError(t, err)

		env.GroupRepository.AssertExpectations(t)
		env.UserRepository.AssertExpectations(t)
	})

	t.Run("Transfer fail: Current user is not owner", func(t *testing.T) {
		env := setupTest()

		groupID := uuid.New()
		currentOwnerID := uuid.New()
		otherUserID := uuid.New()
		newOwnerID := uuid.New()

		env.GroupRepository.On("FindByID", mock.Anything, groupID).Return(&domain.Group{ID: groupID, OwnerID: currentOwnerID}, nil).Once()

		err := env.Service.TransferOwnership(context.Background(), groupID, otherUserID, newOwnerID)

		assert.Error(t, err)
		assert.Equal(t, group.ErrNotGroupOwner, err)

		env.GroupRepository.AssertExpectations(t)
	})

	t.Run("Transfer fail: New owner does not exist", func(t *testing.T) {
		env := setupTest()

		groupID := uuid.New()
		currentOwnerID := uuid.New()
		newOwnerID := uuid.New()

		env.GroupRepository.On("FindByID", mock.Anything, groupID).Return(&domain.Group{ID: groupID, OwnerID: currentOwnerID}, nil).Once()
		env.UserRepository.On("Exists", mock.Anything, newOwnerID).Return(false, nil).Once()

		err := env.Service.TransferOwnership(context.Background(), groupID, currentOwnerID, newOwnerID)

		assert.Error(t, err)
		assert.Equal(t, group.ErrNotGroupOwner, err)

		env.GroupRepository.AssertExpectations(t)
		env.UserRepository.AssertExpectations(t)
	})
}

package repository

import (
	"encoding/json"
	"errors"
	"os"
	"sync"

	"heathhub/internal/model"
)

type UserRepository struct {
	filePath string
	mu       sync.Mutex
	users    map[string]*model.User // key: email
}

func NewUserRepository(filePath string) *UserRepository {
	repo := &UserRepository{
		filePath: filePath,
		users:    make(map[string]*model.User),
	}
	repo.loadFromFile()
	return repo
}

func (r *UserRepository) loadFromFile() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	bytes, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(bytes) == 0 {
		return nil
	}

	return json.Unmarshal(bytes, &r.users)
}

func (r *UserRepository) saveToFile() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	bytes, err := json.MarshalIndent(r.users, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, bytes, 0644)
}

func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[email]
	if !exists {
		return nil, nil
	}
	return user, nil
}

func (r *UserRepository) Create(user *model.User) error {
	r.mu.Lock()

	if _, exists := r.users[user.Email]; exists {
		return errors.New("user already exists")
	}
	r.users[user.Email] = user

	r.mu.Unlock()
	return r.saveToFile()
}

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
)

type Repository interface {
	FindByEmail(email string) (*User, error)
	FindByID(id string) (*User, error)
	Create(user *User) error
	SetPassword(ctx context.Context, email string, passwordHash string) error
}

type userRepository struct {
	filePath string
	mu       sync.Mutex
	users    map[string]*User
}

func NewRepository(filePath string) Repository {
	repo := &userRepository{
		filePath: filePath,
		users:    make(map[string]*User),
	}
	repo.loadFromFile()
	return repo
}

func (r *userRepository) loadFromFile() error {
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

func (r *userRepository) saveToFile() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	bytes, err := json.MarshalIndent(r.users, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, bytes, 0644)
}

func (r *userRepository) FindByEmail(email string) (*User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[email]
	if !exists {
		return nil, nil
	}
	return user, nil
}

func (r *userRepository) FindByID(id string) (*User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, user := range r.users {
		if user.ID.String() == id {
			return user, nil
		}
	}

	return nil, nil
}

func (r *userRepository) SetPassword(ctx context.Context, email string, passwordHash string) error {
	r.mu.Lock()

	user, exists := r.users[email]
	if !exists {
		r.mu.Unlock()
		return errors.New("user not found")
	}

	user.Password = passwordHash
	r.users[email] = user

	r.mu.Unlock()
	return r.saveToFile()
}

func (r *userRepository) Create(user *User) error {
	r.mu.Lock()

	if _, exists := r.users[user.Email]; exists {
		r.mu.Unlock()
		return errors.New("user already exists")
	}
	r.users[user.Email] = user

	r.mu.Unlock()
	return r.saveToFile()
}

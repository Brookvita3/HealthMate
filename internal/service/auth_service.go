package service

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"heathhub/internal/model"
	"heathhub/internal/repository"
)

type AuthService struct {
	UserRepo *repository.UserRepository
}

func NewAuthService(repo *repository.UserRepository) *AuthService {
	return &AuthService{UserRepo: repo}
}

func (s *AuthService) LoginWithGoogleIDToken(idToken string) (*model.User, error) {

	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil || resp.StatusCode != 200 {
		log.Println("error: ", resp.Body)
		return nil, errors.New("invalid Google token")
	}
	defer resp.Body.Close()

	var gUser struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return nil, errors.New("failed to parse Google user info")
	}

	// Check user in Db
	existing, err := s.UserRepo.FindByEmail(gUser.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	newUser := &model.User{
		Email:    gUser.Email,
		Name:     gUser.Name,
		Picture:  gUser.Picture,
		Provider: "google",
	}

	if err := s.UserRepo.Create(newUser); err != nil {
		return nil, err
	}
	return newUser, nil
}

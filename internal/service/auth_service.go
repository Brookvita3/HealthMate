package service

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"heathhub/internal/model"
	"heathhub/internal/repository"
	"heathhub/pkg/auth"
)

type AuthResult struct {
	User         *model.User
	AccessToken  string
	RefreshToken string
	AccessTTL    int64
	RefreshTTL   int64
}

type gUser struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

type AuthService struct {
	UserRepo     *repository.UserRepository
	TokenService *auth.TokenService
}

func NewAuthService(repo *repository.UserRepository, tokenService *auth.TokenService) *AuthService {
	return &AuthService{UserRepo: repo, TokenService: tokenService}
}

func (s *AuthService) LoginWithGoogleIDToken(idToken string) (*AuthResult, error) {

	gUser, err := verifyGoogleIDToken(idToken)
	if err != nil {
		return nil, err
	}

	existing, err := s.UserRepo.FindByEmail(gUser.Email)
	if err != nil {
		return nil, err
	}

	var user *model.User
	if existing != nil {
		user = existing
	} else {
		newUser := &model.User{
			Email:    gUser.Email,
			Name:     gUser.Name,
			Picture:  gUser.Picture,
			Provider: "google",
		}
		if err := s.UserRepo.Create(newUser); err != nil {
			return nil, err
		}
		user = newUser
	}

	accessToken, err := s.TokenService.GenerateAccessJWT(user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.TokenService.GenerateRefreshJWT(user.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AccessTTL:    300,
		RefreshTTL:   3600,
	}, nil

}

func verifyGoogleIDToken(idToken string) (*gUser, error) {

	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil || resp.StatusCode != 200 {
		log.Println("error: ", resp.Body)
		return nil, errors.New("invalid Google token")
	}
	defer resp.Body.Close()

	var gUser gUser

	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return nil, errors.New("failed to parse Google user info")
	}

	return &gUser, nil
}

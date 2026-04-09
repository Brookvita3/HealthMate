package notification

import (
	"context"
	"fmt"
	"log"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/cenkalti/backoff/v4"
	"google.golang.org/api/option"
)

type fcmService struct {
	client *messaging.Client
	repo   Repository
}

func NewFCMService(ctx context.Context, repo Repository, serviceAccountPath string) (Service, error) {
	var app *firebase.App
	var err error

	if serviceAccountPath != "" {
		opt := option.WithCredentialsFile(serviceAccountPath)
		app, err = firebase.NewApp(ctx, nil, opt)
	} else {
		// Fallback to default credentials (e.g. from environment variable GOOGLE_APPLICATION_CREDENTIALS)
		app, err = firebase.NewApp(ctx, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("error initializing firebase app: %v", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting messaging client: %v", err)
	}

	return &fcmService{
		client: client,
		repo:   repo,
	}, nil
}

func (s *fcmService) SendToUser(ctx context.Context, userID string, n Notification) error {
	tokens, err := s.repo.GetTokensByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("error getting user tokens: %v", err)
	}

	if len(tokens) == 0 {
		log.Printf("No device tokens found for user %s", userID)
		return nil
	}

	for _, token := range tokens {
		go func(t string) {
			err := s.SendToToken(context.Background(), t, n)
			if err != nil {
				log.Printf("Failed to send notification to token %s for user %s: %v", t, userID, err)
			}
		}(token)
	}

	return nil
}

func (s *fcmService) SendToToken(ctx context.Context, token string, n Notification) error {
	message := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: n.Title,
			Body:  n.Body,
		},
		Data: n.Data,
	}

	operation := func() error {
		resp, err := s.client.Send(ctx, message)
		if err != nil {
			return err
		}

		log.Println("FCM sent successfully, messageID: %s, message: %v", resp, message)
		if err != nil {
			// Check if the token is invalid and should be deleted
			if messaging.IsUnregistered(err) {
				log.Printf("Token %s is not registered, deleting...", token)
				// We don't have userID here easily unless we change the interface,
				// but we can delete by token if our repo supports it.
				// For now, just logging.
				return nil // Don't retry for invalid tokens
			}
			return err
		}
		return nil
	}

	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.MaxElapsedTime = 30 * time.Second

	err := backoff.Retry(operation, expBackoff)
	if err != nil {
		return fmt.Errorf("failed to send FCM after retries: %v", err)
	}

	return nil
}

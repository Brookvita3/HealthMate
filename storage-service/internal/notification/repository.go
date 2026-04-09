package notification

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) SaveToken(ctx context.Context, token DeviceToken) error {
	query := `
		INSERT INTO user_device_tokens (user_id, token, platform, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, token) 
		DO UPDATE SET platform = EXCLUDED.platform, updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, token.UserID, token.Token, token.Platform)
	if err != nil {
		return fmt.Errorf("failed to save device token: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetTokensByUserID(ctx context.Context, userID string) ([]string, error) {
	query := `SELECT token FROM user_device_tokens WHERE user_id = $1`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device tokens: %w", err)
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, fmt.Errorf("failed to scan token: %w", err)
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func (r *postgresRepository) DeleteToken(ctx context.Context, userID string, token string) error {
	query := `DELETE FROM user_device_tokens WHERE user_id = $1 AND token = $2`
	_, err := r.pool.Exec(ctx, query, userID, token)
	if err != nil {
		return fmt.Errorf("failed to delete device token: %w", err)
	}
	return nil
}

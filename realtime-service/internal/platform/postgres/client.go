package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgreSQLConnFromURL(postgreDBURL string) (*pgxpool.Pool, error) {
	conn, err := pgxpool.New(context.Background(), postgreDBURL)
	if err != nil {
		return nil, fmt.Errorf("can't connect to database: %w", err)
	}

	err = conn.Ping(context.Background())
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("can't ping database: %w", err)
	}

	return conn, nil
}

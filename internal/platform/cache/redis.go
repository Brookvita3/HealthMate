package cache

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(addr, username, password string, db int, useTLS bool) (*redis.Client, error) {

	opts := &redis.Options{
		Addr:     addr,
		Username: username,
		Password: password,
		DB:       db,
	}

	if useTLS {
		opts.TLSConfig = &tls.Config{}
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("could not connect to redis at %s: %w", addr, err)
	}

	return client, nil
}

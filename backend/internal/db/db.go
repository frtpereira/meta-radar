package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// New creates a pgx connection pool and verifies connectivity with a ping,
// retrying briefly since docker-compose can start the API before Postgres
// is fully ready to accept connections even after its healthcheck passes.
func New(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}

	var pingErr error
	for i := 0; i < 10; i++ {
		pingErr = pool.Ping(ctx)
		if pingErr == nil {
			return pool, nil
		}
		time.Sleep(1 * time.Second)
	}

	pool.Close()
	return nil, fmt.Errorf("pinging database after retries: %w", pingErr)
}

package db

import (
	"context"
	"errors"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoDatabase means DATABASE_URL isn't set in config/.env.
var ErrNoDatabase = errors.New("DATABASE_URL isn't set in config/.env")

var (
	poolOnce sync.Once
	pool     *pgxpool.Pool
	poolErr  error
)

// connectDB returns the shared connection pool to DATABASE_URL, opening it
// the first time.
func connectDB() (*pgxpool.Pool, error) {
	poolOnce.Do(func() {
		url := os.Getenv("DATABASE_URL")
		if url == "" {
			poolErr = ErrNoDatabase
			return
		}
		pool, poolErr = pgxpool.New(context.Background(), url)
	})
	return pool, poolErr
}

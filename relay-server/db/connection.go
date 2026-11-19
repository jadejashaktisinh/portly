package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var pool *pgxpool.Pool

func Connect() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	fmt.Println("Connecting to database...")
	var err error
	pool, err = pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}

	fmt.Println("Successfully connected to PostgreSQL!")
}

func Close() {
	if pool != nil {
		pool.Close()
	}
}

func ValidateAPIKey(ctx context.Context, key string) (bool, error) {
	if pool == nil || key == "" {
		return false, nil
	}

	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM "User" WHERE "apiKey" = $1)`, key).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var pool *pgxpool.Pool

type User struct {
	ID string
}

type WebhookEvent struct {
	RequestID  string
	UserID     string
	Method     string
	Host       string
	Slug       string
	Path       string
	Headers    http.Header
	Body       string
	StatusCode int
	DurationMs int
	Error      string
}

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

func FindUserByAPIKey(ctx context.Context, key string) (User, bool, error) {
	if pool == nil || key == "" {
		return User{}, false, nil
	}

	var id string
	err := pool.QueryRow(ctx, `SELECT "id" FROM "User" WHERE "apiKey" = $1`, key).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}
	return User{ID: id}, true, nil
}

func ValidateAPIKey(ctx context.Context, key string) (bool, error) {
	_, found, err := FindUserByAPIKey(ctx, key)
	return found, err
}

func RecordWebhookEvent(ctx context.Context, event WebhookEvent) error {
	if pool == nil {
		return nil
	}

	headers, err := json.Marshal(event.Headers)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = pool.Exec(ctx, `
		INSERT INTO "WebhookEvent"
			("id", "requestId", "userId", "method", "host", "slug", "path", "headers", "body", "statusCode", "durationMs", "error", "createdAt")
		VALUES (gen_random_uuid()::text, $1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11, NOW())`,
		event.RequestID,
		event.UserID,
		event.Method,
		event.Host,
		nullableString(event.Slug),
		event.Path,
		string(headers),
		nullableString(event.Body),
		nullableInt(event.StatusCode),
		nullableInt(event.DurationMs),
		nullableString(event.Error),
	)
	return err
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

package db

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func Connect() {

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	// 1. Define your connection string (ideally loaded from an environment variable)
	// Format: postgres://username:password@localhost:5432/database_name

	// 2. Create the connection pool
	ctx := context.Background()
	fmt.Printf("Attempting to connect to database: %v\n", os.Getenv("DATABASE_URL"))
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	// 3. Verify the connection works
	err = conn.Ping(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully connected to PostgreSQL!")

	// Execute operatio
}

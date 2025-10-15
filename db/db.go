package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
)

// global variable
var Pool *pgxpool.Pool

// initalixe tje connection
func InitializeDB() {
	err := godotenv.Load()
	dbURL := os.Getenv("DATABASE_URL")

	//fmt.Println("Database URL:", dbURL)
	//os.Exit(0)

	Pool, err = pgxpool.Connect(context.Background(), dbURL)
	if err != nil {
		fmt.Println(" Unable to connect to database: %v", err)
	}

	fmt.Println("Database pool connected successfully")
}

// closeDB
func CloseDB() {
	if Pool != nil {
		Pool.Close()
		fmt.Println("Database pool closed.")
	}
}

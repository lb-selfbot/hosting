package main

import (
	"fmt"
	"log"
	"os"

	"github.com/getsentry/sentry-go"
	_ "github.com/go-sql-driver/mysql"

	"github.com/joho/godotenv"
)

// ID --> User
var Users = make(map[int]*User)

// ID --> HostInstance
var Instances = make(map[int]*HostInstance)

func main() {
	// Setup environment variables
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file:", err)
		os.Exit(1)
	}

	// Setup sentry
	err := sentry.Init(sentry.ClientOptions{
		Dsn: os.Getenv("SENTRY_DSN"),
		TracesSampleRate: 1.0,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}

	// Setup database
	db := NewDatabase()
	if err := db.Connect(); err != nil {
		fmt.Println("Error connecting to database:", err)
		os.Exit(1)
	}

	worker(db)
}

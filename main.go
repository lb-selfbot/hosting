package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/LightningDev1/LB-Hosting/api"
	"github.com/LightningDev1/LB-Hosting/database"
	"github.com/LightningDev1/LB-Hosting/hosting"
	"github.com/getsentry/sentry-go"
	_ "github.com/go-sql-driver/mysql"

	"github.com/joho/godotenv"
)

func main() {
	// Setup environment variables
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file:", err)
		os.Exit(1)
	}

	// Setup sentry
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              os.Getenv("SENTRY_DSN"),
		TracesSampleRate: 1.0,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}

	// Setup database
	db := database.NewDatabase()
	if err := db.Connect(); err != nil {
		fmt.Println("Error connecting to database:", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup

	go func() {
		wg.Add(1)
		defer wg.Done()
		hosting.Worker(db)
	}()

	go func() {
		wg.Add(1)
		defer wg.Done()
		api.StartAPI(db)
	}()

	wg.Wait()
}

package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

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

	wg.Add(2)

	go func() {
		defer wg.Done()
		for {
			hosting.Worker(db)

			fmt.Println("[WARNING] Worker stopped, restarting in 5 seconds...")
			time.Sleep(5 * time.Second)
		}
	}()

	go func() {
		defer wg.Done()
		for {
			api.StartAPI(db)

			fmt.Println("[WARNING] API stopped, restarting in 5 seconds...")
			time.Sleep(5 * time.Second)
		}
	}()

	wg.Wait()
}

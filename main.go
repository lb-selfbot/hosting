package main

import (
	"fmt"
	"os"

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

	// Setup database
	db := NewDatabase()
	if err := db.Connect(); err != nil {
		fmt.Println("Error connecting to database:", err)
		os.Exit(1)
	}

	worker(db)
}

package main

import (
	"fmt"
	"time"
)

func worker(db *Database) {
	for {
		fmt.Println("Getting users")

		users, err := db.GetUsers()
		if err != nil {
			fmt.Println("Error getting hosted users:", err)
		}

		for _, user := range users {
			instance := GetHostInstance(user)

			if instance.IsRunning() {
				handleRunningInstance(instance, user, db)
			} else {
				handleStoppedInstance(instance, user, db)
			}
		}

		time.Sleep(time.Second * 5)
	}
}

func handleRunningInstance(instance *HostInstance, user *User, db *Database) {
	if user.IsHosting {
		fmt.Println("User", user.Username, "is hosting, credit:", user.HostingCredit)

		// Remove 1 credit (1 minute)
		user.HostingCredit--

		// Update the user in the database
		if err := db.SaveUser(user); err != nil {
			fmt.Println("Error updating user:", err)
		}

		// Check if the user has enough credit
		if user.HostingCredit <= 0 {
			// User doesn't have enough credit ==> Stop the instance
			instance.Stop()

			fmt.Println("Stopping instance for user:", user.Username)
		}
	}

	if !user.IsHosting {
		// User isn't hosting but the instance is running ==> Stop the instance
		instance.Stop()

		fmt.Println("Stopping instance for user:", user.Username)
	}
}

func handleStoppedInstance(instance *HostInstance, user *User, db *Database) {
	// If the instance is not running and the user is hosting, start the instance
	if user.IsHosting && user.HostingCredit > 0 {
		token, err := db.GetUserToken(user)
		if err != nil {
			fmt.Println("Error getting user token:", err)
		}

		instance.Run(token)

		fmt.Println("Starting instance for user:", user.Username)
	}
}

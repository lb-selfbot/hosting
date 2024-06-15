package hosting

import (
	"context"
	"fmt"
	"time"

	"sync/atomic"

	"github.com/LightningDev1/LB-Hosting/database"
	"github.com/LightningDev1/LB-Hosting/log"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

const (
	WorkerStatusStopped = 0
	WorkerStatusRunning = 1
)

var WorkerStatus = atomic.Int32{}

func Worker(db *database.Database) {
	WorkerStatus.Store(WorkerStatusRunning)
	defer WorkerStatus.Store(WorkerStatusStopped)

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Println("Error creating docker client:", err)
		return
	}

	dockerClient.NegotiateAPIVersion(context.Background())

	for {
		fmt.Println("Getting users")

		users, err := db.GetUsers()
		if err != nil {
			fmt.Println("Error getting hosted users:", err)
		}

		containers, err := dockerClient.ContainerList(context.Background(), container.ListOptions{All: true})
		if err != nil {
			fmt.Println("Error getting containers:", err)
		}

		for _, user := range users {
			instance := GetHostInstance(user, dockerClient)

			if instance.IsRunning(containers) {
				handleRunningInstance(instance, user, db)
			} else {
				handleStoppedInstance(instance, user, db)
			}

			handleErrors(instance, user, db)
		}

		time.Sleep(time.Second * 60)
	}
}

func handleErrors(instance *HostInstance, user *database.User, db *database.Database) {
	if instance.Errors > instance.MaxErrors && user.IsHosting {
		// Too many errors ==> Stop the instance
		instance.Stop()

		// Reset the error counter
		instance.Errors = 0
		instance.ErrorStart = time.Now()

		// Set the user to not hosting
		user.IsHosting = false

		// Update the user in the database
		if err := db.SaveUser(user); err != nil {
			fmt.Println("Error updating user:", err)
		}

		log.LogTooManyErrors(user)

		fmt.Println("Stopping instance for user (too many errors):", user.Username)
	}
}

func handleRunningInstance(instance *HostInstance, user *database.User, db *database.Database) {
	if user.IsHosting {
		fmt.Println("User", user.Username, "is hosting, credit:", user.HostingCredit, "lifetime:", user.HostingLifetime)

		if !user.HostingLifetime {
			// Remove 1 credit (1 minute)
			user.HostingCredit--

			// Check if the user has enough credit
			if user.HostingCredit <= 0 {
				// User doesn't have enough credit ==> Stop the instance
				instance.Stop()

				user.IsHosting = false

				fmt.Println("Stopping instance for user:", user.Username)
			}

			// Update the user in the database
			if err := db.SaveUser(user); err != nil {
				fmt.Println("Error updating user:", err)
			}
		}
	}

	if !user.IsHosting {
		// User isn't hosting but the instance is running ==> Stop the instance
		instance.Stop()

		fmt.Println("Stopping instance for user:", user.Username)
	}
}

func handleStoppedInstance(instance *HostInstance, user *database.User, db *database.Database) {
	// If the instance is not running and the user is hosting, start the instance
	if user.IsHosting && (user.HostingCredit > 0 || user.HostingLifetime) {
		token, err := db.GetUserToken(user)
		if err != nil {
			fmt.Println("Error getting user token:", err)
		}

		instance.Run(token)

		fmt.Println("Starting instance for user:", user.Username)
	}
}

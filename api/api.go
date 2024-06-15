package api

import (
	"crypto/subtle"
	"os"

	"github.com/LightningDev1/LB-Hosting/database"
	"github.com/gofiber/fiber/v2"
)

func StartAPI(db *database.Database) {
	app := fiber.New(fiber.Config{
		AppName: "LB-Hosting API",
	})

	apiKey := []byte(os.Getenv("API_KEY"))
	if apiKey == nil {
		panic("API_KEY not set")
	}

	middleware :=
		func(c *fiber.Ctx) error {
			requestKey := []byte(c.Get("Authorization"))

			if subtle.ConstantTimeCompare(requestKey, apiKey) != 1 {
				return c.SendStatus(fiber.StatusUnauthorized)
			}

			c.Locals("db", db)

			return c.Next()
		}

	app.Get("/api/v1/status", middleware, getStatus)
	app.Get("/api/v1/list-files", middleware, getListFiles)
	app.Post("/api/v1/delete-folder", middleware, postDeleteFolder)
	app.Post("/api/v1/delete-file", middleware, postDeleteFile)
	app.Get("/api/v1/download-file", middleware, getDownloadFile)

	host := ":8085"
	if len(os.Args) > 1 && os.Args[1] != "" {
		host = os.Args[1]
	}

	app.Listen(host)
}

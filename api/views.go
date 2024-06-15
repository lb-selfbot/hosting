package api

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/LightningDev1/LB-Hosting/database"
	"github.com/LightningDev1/LB-Hosting/hosting"
	"github.com/LightningDev1/LB-Hosting/utils"
	"github.com/gofiber/fiber/v2"
)

type statusResponse struct {
	Error        bool  `json:"error"`
	WorkerStatus int32 `json:"worker_status"`
}

func getStatus(c *fiber.Ctx) error {
	workerStatus := hosting.WorkerStatus.Load()

	return c.JSON(statusResponse{
		Error:        false,
		WorkerStatus: workerStatus,
	})
}

func getListFiles(c *fiber.Ctx) error {
	userIDStr := c.Get("User-Id")
	if userIDStr == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	db := c.Locals("db").(*database.Database)

	user, err := db.GetUserByID(userID)
	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	_, _, mainFolder := utils.GetUserFolders(user.Username, user.ID)

	if _, err := os.Stat(mainFolder); os.IsNotExist(err) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":  true,
			"reason": "You must enable hosting at least once before you can list files.",
		})
	}

	content := make(map[string]map[string][]string)
	err = filepath.Walk(mainFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath := strings.Replace(strings.TrimPrefix(path, mainFolder), "\\", "/", -1)
		if relPath == "" {
			relPath = "/"
		}

		if _, exists := content[relPath]; !exists {
			content[relPath] = make(map[string][]string)
		}

		if info.IsDir() {
			content[relPath]["folders"] = append(content[relPath]["folders"], info.Name())
		} else {
			content[relPath]["files"] = append(content[relPath]["files"], info.Name())
		}
		return nil
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  true,
			"reason": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"error":   false,
		"content": content,
	})
}

func postDeleteFolder(c *fiber.Ctx) error {
	userIDStr := c.Get("User-Id")
	if userIDStr == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	db := c.Locals("db").(*database.Database)

	user, err := db.GetUserByID(userID)
	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	_, _, mainFolder := utils.GetUserFolders(user.Username, user.ID)

	if _, err := os.Stat(mainFolder); os.IsNotExist(err) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  true,
			"reason": "You must enable hosting at least once before you can delete files.",
		})
	}

	folderPathParam := c.FormValue("folder_path")
	cleanedFolderPath := utils.CleanFilename(folderPathParam, true)

	cleanedFolderPath = strings.TrimPrefix(cleanedFolderPath, "/") // Remove leading slashes
	requestedPath := filepath.Join(mainFolder, cleanedFolderPath)

	// Security check: Ensure the requested path is within the user's hosting directory
	if !strings.HasPrefix(filepath.Clean(requestedPath), mainFolder) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  true,
			"reason": "Invalid folder path.",
		})
	}

	if _, err := os.Stat(requestedPath); os.IsNotExist(err) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  true,
			"reason": "Folder does not exist.",
		})
	}

	isEmpty, _ := utils.IsDirEmpty(requestedPath)
	if !isEmpty {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  true,
			"reason": "Folder is not empty.",
		})
	}

	err = os.Remove(requestedPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  true,
			"reason": "Failed to delete folder.",
		})
	}

	return c.JSON(fiber.Map{"error": false})
}

func postDeleteFile(c *fiber.Ctx) error {
	userIDStr := c.Get("User-Id")
	if userIDStr == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	db := c.Locals("db").(*database.Database)

	user, err := db.GetUserByID(userID)
	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	_, _, mainFolder := utils.GetUserFolders(user.Username, user.ID)

	if _, err := os.Stat(mainFolder); os.IsNotExist(err) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  true,
			"reason": "You must enable hosting at least once before you can delete files.",
		})
	}

	filePathParam := c.FormValue("file_path")
	cleanedFilePath := utils.CleanFilename(filePathParam, false)
	cleanedFilePath = strings.TrimPrefix(cleanedFilePath, "/")
	requestedPath := filepath.Join(mainFolder, cleanedFilePath)

	// Security check: Ensure the requested path is within the user's hosting directory
	if !strings.HasPrefix(filepath.Clean(requestedPath), mainFolder) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  true,
			"reason": "Invalid file path.",
		})
	}

	if _, err := os.Stat(requestedPath); os.IsNotExist(err) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  true,
			"reason": "File does not exist.",
		})
	}

	err = os.Remove(requestedPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  true,
			"reason": "Failed to delete file.",
		})
	}

	return c.JSON(fiber.Map{"error": false})
}

func getDownloadFile(c *fiber.Ctx) error {
	userIDStr := c.Get("User-Id")
	if userIDStr == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	db := c.Locals("db").(*database.Database)

	user, err := db.GetUserByID(userID)
	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	_, _, mainFolder := utils.GetUserFolders(user.Username, user.ID)

	if _, err := os.Stat(mainFolder); os.IsNotExist(err) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  true,
			"reason": "You must enable hosting at least once before you can download files.",
		})
	}

	filePathParam := c.Query("file_path")
	if filePathParam == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  true,
			"reason": "Missing file path.",
		})
	}

	cleanedFilePath := utils.CleanFilename(filePathParam, false)
	cleanedFilePath = strings.TrimPrefix(cleanedFilePath, "/")
	requestedPath := filepath.Join(mainFolder, cleanedFilePath)

	// Security check: Ensure the requested path is within the user's hosting directory
	if !strings.HasPrefix(filepath.Clean(requestedPath), mainFolder) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  true,
			"reason": "Invalid file path.",
		})
	}

	if _, err := os.Stat(requestedPath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  true,
			"reason": "File does not exist.",
		})
	}

	// Set headers for file download
	c.Set("Content-Disposition", "attachment; filename="+filepath.Base(requestedPath))
	c.Set("Content-Type", "application/octet-stream")
	return c.SendFile(requestedPath)
}

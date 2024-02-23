package main

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func cleanFilename(filename string) string {
	// Replace invalid characters with underscores
	re := regexp.MustCompile(`[^a-zA-Z0-9\._-]`)
	cleaned := re.ReplaceAllString(filename, "_")

	// Remove leading and trailing underscores
	cleaned = strings.Trim(cleaned, "_")

	// Remove leading dot (if any)
	cleaned = strings.TrimPrefix(cleaned, ".")

	// Make sure the result is not "." or ".."
	if cleaned == "" || cleaned == "." || cleaned == ".." {
		cleaned = "_"
	}

	return cleaned
}

func migrateInstance(client *client.Client, cID, username, homeFolder string) error {
	// Check if the user has already been migrated
	migratedPath := filepath.Join(homeFolder, ".migrated")
	_, err := os.Stat(migratedPath)
	if !os.IsNotExist(err) {
		return nil
	}

	userCfgPath := filepath.Join(homeFolder, "LightningBot", "Config", "config.json")

	fileStat, err := os.Stat(userCfgPath)
	if os.IsNotExist(err) {
		fmt.Printf("[MIGRATION: %s] User config not found\n", username)
		return nil
	} else if err != nil {
		fmt.Printf("[MIGRATION: %s] User checking user config: %v", username, err)
		return err
	}

	// Compress config file using TAR
	archive := new(bytes.Buffer)

	tw := tar.NewWriter(archive)
	defer tw.Close()

	header := &tar.Header{
		Name: "config.json",
		Mode: 0644,
		Size: fileStat.Size(),
	}

	if err := tw.WriteHeader(header); err != nil {
		fmt.Printf("[MIGRATION: %s] Error writing TAR header: %v\n", username, err)
		return err
	}

	file, err := os.Open(userCfgPath)
	if err != nil {
		fmt.Printf("[MIGRATION: %s] Error opening user config file: %v\n", username, err)
		return err
	}

	if _, err := io.Copy(tw, file); err != nil {
		fmt.Printf("[MIGRATION: %s] Error writing TAR body: %v\n", username, err)
		return err
	}

	// Create folder in the container
	_, err = client.ContainerExecCreate(
		context.Background(),
		cID,
		types.ExecConfig{
			Cmd:          []string{"sudo", "-u", "lightningbot", "mkdir", "-p", "/lightningbot/Config"},
			AttachStdout: true,
			AttachStderr: true,
		},
	)
	if err != nil {
		fmt.Printf("[MIGRATION: %s] Error creating folder in container: %v\n", username, err)
		return err
	}

	// Copy files from the user's folder to the docker volume
	err = client.CopyToContainer(
		context.Background(),
		cID,
		"/lightningbot/Config",
		archive,
		types.CopyToContainerOptions{
			AllowOverwriteDirWithFile: true,
		},
	)
	if err != nil {
		fmt.Printf("[MIGRATION: %s] Error copying files to container: %v\n", username, err)
		return err
	}

	// Mark the user as migrated
	f, err := os.Create(migratedPath)
	if err != nil {
		fmt.Printf("[MIGRATION: %s] Error creating migration file: %v\n", username, err)
		return err
	}
	_ = f.Close()

	fmt.Printf("[MIGRATION: %s] User config migrated\n", username)

	return nil

}

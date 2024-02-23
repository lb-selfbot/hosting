package main

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/LightningDev1/LB-Hosting/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type HostInstance struct {
	DockerClient *client.Client
	User         *User
	Token        string

	// Sliding window for errors
	Errors      int
	MaxErrors   int
	ErrorStart  time.Time
	ErrorWindow time.Duration
}

func (h *HostInstance) containerName() string {
	return fmt.Sprintf("lightningbot-hosting-%d", h.User.ID)
}

func (h *HostInstance) getContainer() (*types.Container, error) {
	containers, err := h.DockerClient.ContainerList(context.Background(), container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.KeyValuePair{Key: "name", Value: h.containerName()},
		),
	})
	if err != nil {
		return nil, err
	}

	if len(containers) == 0 {
		return nil, fmt.Errorf("container not found")
	}

	c := containers[0]

	if c.State == "exited" && c.Status != "Exited (0)" {
		// Exited with error
		h.Errors++

		if time.Since(h.ErrorStart) > h.ErrorWindow {
			h.Errors = 0
			h.ErrorStart = time.Now()
		}

		logs, err := h.DockerClient.ContainerLogs(
			context.Background(),
			c.ID,
			container.LogsOptions{ShowStdout: true, ShowStderr: true},
		)
		if err != nil {
			return nil, err
		}

		logBytes := new(bytes.Buffer)
		_, err = logBytes.ReadFrom(logs)
		if err != nil {
			return nil, err
		}

		logString := logBytes.String()

		fmt.Println("Bot for user", h.User.Username, "has exited:", logString, "with status code", c.Status)

		reportError(h.User, logBytes.String(), c.Status)
	}

	return &c, nil
}

func (h *HostInstance) getFolders() (string, string, string) {
	username := fmt.Sprintf("%s-%d", utils.CleanFilename(h.User.Username), h.User.ID)
	homeFolder := filepath.Join("/hosting", username)
	mainFolder := filepath.Join(homeFolder, "LightningBot")

	return username, homeFolder, mainFolder
}

func (h *HostInstance) createContainer() (*types.Container, error) {
	if c, err := h.getContainer(); err == nil {
		// Stop and delete the container if it exists
		err = h.DockerClient.ContainerRemove(context.Background(), c.ID, container.RemoveOptions{})
		if err != nil {
			fmt.Println("Error removing container:", err)
		}
	}

	username, homeFolder, mainFolder := h.getFolders()

	uid, gid, err := utils.CreateUser(username, homeFolder)
	if err != nil {
		fmt.Println("Error creating user:", err)
		return nil, err
	}

	err = utils.CreateFolderIfNotExists(mainFolder, 0755, int(uid), int(gid))
	if err != nil {
		fmt.Println("Error creating home folder:", err)
		return nil, err
	}

	c, err := h.DockerClient.ContainerCreate(
		context.Background(),
		&container.Config{
			Image: "lightningbot-hosting:latest",
			Env: []string{
				"USERNAME=" + h.User.Username,
				"PASSWORD=s]+zs,p7+X2eWY$gJ90£DjZ}RlWjk21*sbY<lE)VTWy5b!N'",
				"TOKEN=" + h.Token,
				"MAIN_FOLDER=/lightningbot",
				"UID=" + fmt.Sprint(uid),
				"GID=" + fmt.Sprint(gid),
			},
		},
		&container.HostConfig{
			CapAdd:     []string{"NET_ADMIN"},
			DNS:        []string{"8.8.8.8"},
			Privileged: true,
			Resources: container.Resources{
				// 0.1 CPU
				NanoCPUs: 0.1 * 1e9,
				// 200MB of memory
				Memory: 200 * 1024 * 1024,
			},
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: mainFolder,
					Target: "/lightningbot",
				},
			},
		},
		&network.NetworkingConfig{},
		&v1.Platform{},
		h.containerName(),
	)
	if err != nil {
		fmt.Println("Error creating container:", err)
		return nil, err
	}

	return &types.Container{ID: c.ID}, nil
}

func (h *HostInstance) Run(token string) {
	if token != "" {
		var err error
		h.Token, err = decrypt(token)
		if err != nil {
			fmt.Println("Error decrypting token:", err)
			return
		}
		fmt.Println(h.Token)
	}

	// Start docker container for user
	_, err := h.createContainer()
	if err != nil {
		fmt.Println("Error creating container:", err)
		return
	}

	err = h.DockerClient.ContainerStart(context.Background(), h.containerName(), container.StartOptions{})
	if err != nil {
		fmt.Println("Error starting container:", err)
		return
	}

	// _, homeFolder, _ := h.getFolders()

	// err = migrateInstance(h.DockerClient, c.ID, h.User.Username, homeFolder)
	// if err != nil {
	// 	fmt.Println("Error migrating instance:", err)
	// 	return
	// }
}

func (h *HostInstance) Stop() {
	c, err := h.getContainer()
	if err != nil {
		fmt.Println("Error getting container:", err)
		return
	}

	timeout := 0
	err = h.DockerClient.ContainerStop(
		context.Background(),
		c.ID,
		container.StopOptions{Timeout: &timeout},
	)
	if err != nil {
		fmt.Println("Error stopping container:", err)
		return
	}
}

func (h *HostInstance) IsRunning() bool {
	container, _ := h.getContainer()

	return container != nil && container.State == "running"
}

func GetHostInstance(user *User, dockerClient *client.Client) *HostInstance {
	return &HostInstance{DockerClient: dockerClient, User: user}
}

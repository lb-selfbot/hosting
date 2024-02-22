package main

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
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

func (h *HostInstance) createContainer() error {
	if _, err := h.getContainer(); err == nil {
		return nil
	}

	_, err := h.DockerClient.ContainerCreate(
		context.Background(),
		&container.Config{
			Image: "lightningbot-hosting:latest",
			Env: []string{
				"USERNAME=" + h.User.Username,
				"PASSWORD=s]+zs,p7+X2eWY$gJ90£DjZ}RlWjk21*sbY<lE)VTWy5b!N'",
				"TOKEN=" + h.Token,
				"MAIN_FOLDER=/home/lightningbot",
			},
		},
		&container.HostConfig{
			CapAdd:     []string{"NET_ADMIN"},
			DNS:        []string{"8.8.8.8"},
			Privileged: true,
		},
		&network.NetworkingConfig{},
		&v1.Platform{},
		h.containerName(),
	)
	return err
}

func (h *HostInstance) Run(token string) {
	if token != "" {
		var err error
		h.Token, err = decrypt(token)
		if err != nil {
			fmt.Println("Error decrypting token:", err)
			return
		}
		h.Token += "aa"
	}

	// Start docker container for user
	err := h.createContainer()
	if err != nil {
		fmt.Println("Error creating container:", err)
		return
	}

	err = h.DockerClient.ContainerStart(context.Background(), h.containerName(), container.StartOptions{})
	if err != nil {
		fmt.Println("Error starting container:", err)
		return
	}
}

func (h *HostInstance) Stop() {
	c, err := h.getContainer()
	if err != nil {
		fmt.Println("Error getting container:", err)
		return
	}

	timeout := 5
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

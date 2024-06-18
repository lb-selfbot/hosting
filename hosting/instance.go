package hosting

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"time"

	"github.com/LightningDev1/LB-Hosting/database"
	"github.com/LightningDev1/LB-Hosting/encryption"
	"github.com/LightningDev1/LB-Hosting/log"
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
	User         *database.User
	Token        string

	// Sliding window for errors
	Errors      int
	MaxErrors   int
	ErrorStart  time.Time
	ErrorWindow time.Duration
}

// ID --> HostInstance
var Instances = make(map[int]*HostInstance)

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

	containers = slices.DeleteFunc(containers, func(c types.Container) bool {
		return c.Names[0] != "/"+h.containerName()
	})

	if len(containers) == 0 {
		return nil, fmt.Errorf("container not found")
	}

	c := containers[0]

	// Exited with non-zero exit code
	if c.State == "exited" && !strings.Contains(c.Status, "Exited (0)") {
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

		log.LogError(h.User, logBytes.String(), c.Status)
	}

	return &c, nil
}

func (h *HostInstance) ifaceToMacvlan(iface string) string {
	return "macvlan-" + strings.ReplaceAll(iface, ":", "-")
}

func (h *HostInstance) createContainer() (*types.Container, error) {
	if c, err := h.getContainer(); err == nil {
		// Stop and delete the container if it exists
		err = h.DockerClient.ContainerRemove(context.Background(), c.ID, container.RemoveOptions{})
		if err != nil {
			fmt.Println("Error removing container:", err)
		}
	}

	username, homeFolder, mainFolder := utils.GetUserFolders(h.User.Username, h.User.ID)

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

	// Choose random IP
	// ips := []string{"134.255.220.154", "5.249.162.249"}
	// ip := ips[rand.Intn(len(ips))]

	interfaces := []string{"eth0", "eth0.1"}
	parentInterface := interfaces[rand.Intn(len(interfaces))]

	// Check if macvlan network exists, if not, create it.
	networkList, err := h.DockerClient.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		fmt.Println("Error creating container: listing networks:", err)
		return nil, err
	}

	macvlanName := h.ifaceToMacvlan(parentInterface)

	fmt.Println("Using", parentInterface, macvlanName, "for user", h.User.Username)

	macvlanNetworkExists := false
	for _, network := range networkList {
		if network.Name == macvlanName && network.Driver == "macvlan" {
			macvlanNetworkExists = true
			break
		}
	}

	if !macvlanNetworkExists {
		var subnet, gateway string
		if parentInterface == "eth0" {
			subnet = "134.255.220.0/24" // From eth0 configuration
			gateway = "134.255.220.1"   // From eth0 configuration
		} else if parentInterface == "eth0.1" {
			subnet = "5.249.162.0/24" // From eth0:1 configuration
			gateway = "5.249.162.1"   // From eth0:1 configuration
		}

		_, err = h.DockerClient.NetworkCreate(context.Background(), macvlanName, types.NetworkCreate{
			Driver: "macvlan",
			IPAM: &network.IPAM{
				Config: []network.IPAMConfig{
					{Subnet: subnet, Gateway: gateway},
				},
			},
			Options: map[string]string{
				"parent": parentInterface,
			},
		})
		if err != nil {
			fmt.Println("Error creating container: creating network", err)
			return nil, err
		}
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
			// Start without network
			// NetworkMode: "defaukt",
			NetworkMode: container.NetworkMode(macvlanName),
		},
		&network.NetworkingConfig{
			// EndpointsConfig: map[string]*network.EndpointSettings{
			// 	"hosting-macvlan": {
			// 		IPAMConfig: &network.EndpointIPAMConfig{
			// 			IPv4Address: ip,
			// 		},
			// 	},
			// },
		},
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
		h.Token, err = encryption.Decrypt(token)
		if err != nil {
			fmt.Println("Error decrypting token:", err)
			return
		}
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

func (h *HostInstance) IsRunning(containers []types.Container) bool {
	for _, c := range containers {
		if c.Names[0] == "/"+h.containerName() {
			return c.State == "running"
		}
	}

	return false
}

func GetHostInstance(user *database.User, dockerClient *client.Client) *HostInstance {
	if instance, ok := Instances[user.ID]; ok {
		// Update user
		instance.User = user

		return instance
	}

	Instances[user.ID] = &HostInstance{
		DockerClient: dockerClient,
		User:         user,
		MaxErrors:    5,
		ErrorStart:   time.Now(),
		ErrorWindow:  time.Minute * 10,
	}

	return Instances[user.ID]
}

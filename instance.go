package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync/atomic"
)

type HostInstance struct {
	User        *User
	Locker      atomic.Uint32
	StopChannel chan struct{}
	Cmd         *exec.Cmd

	Token string
}

func (h *HostInstance) handler() {
	if !h.Locker.CompareAndSwap(0, 1) {
		return
	}
	defer h.Locker.Store(0)

	fmt.Println("Running bot for user:", h.User.Username)

	// Set main folder
	mainFolder := filepath.Join("/hosting", h.User.Username, "LightningBot")

	// RUn the executable
	h.Cmd = exec.Command("./LB-Selfbot")
	h.Cmd.Env = append(h.Cmd.Env, fmt.Sprintf("USERNAME=%s", h.User.Username))
	h.Cmd.Env = append(h.Cmd.Env, fmt.Sprintf("PASSWORD=%s", "s]+zs,p7+X2eWY$gJ90£DjZ}RlWjk21*sbY<lE)VTWy5b!N'"))
	h.Cmd.Env = append(h.Cmd.Env, fmt.Sprintf("TOKEN=%s", h.Token))
	h.Cmd.Env = append(h.Cmd.Env, fmt.Sprintf("MAIN_FOLDER=%s", mainFolder))
	// h.Cmd.Stdout

	stdoutPipe, err := h.Cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error getting stdout:", err)
		return
	}

	stdoutReader := bufio.NewReader(stdoutPipe)

	err = h.Cmd.Start()
	if err != nil {
		fmt.Println("Error starting bot:", err)
		return
	}

	processKilled := make(chan struct{})
	stdout := ""

	go func() {
		for {
			stdoutData, err := stdoutReader.ReadString('\n')
			if err != nil {
				break
			}

			stdout += stdoutData
		}
		h.Cmd.Wait()
		processKilled <- struct{}{}
	}()

	select {
	case <-h.StopChannel:
		fmt.Println("Stopping bot for user:", h.User.Username)
		h.Cmd.Process.Kill()

		return
	case <-processKilled:
		fmt.Println("Bot for user", h.User.Username, "has exited:", stdout, h.Cmd.ProcessState.ExitCode())

		return
	}
}

func (h *HostInstance) Run(token string) {
	h.Token = token

	go h.handler()
}

func (h *HostInstance) Stop() {
	h.StopChannel <- struct{}{}
}

func (h *HostInstance) IsRunning() bool {
	return h.Locker.Load() == 1
}

func GetHostInstance(user *User) *HostInstance {
	if instance, ok := Instances[user.ID]; ok {
		// Update user
		instance.User = user

		return instance
	}

	Instances[user.ID] = &HostInstance{
		User:        user,
		Locker:      atomic.Uint32{},
		StopChannel: make(chan struct{}),
	}

	return Instances[user.ID]
}

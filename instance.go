package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type HostInstance struct {
	User        *User
	Locker      atomic.Uint32
	StopChannel chan struct{}
	Cmd         *exec.Cmd

	Token string

	Stdout string

	// Sliding window for errors
	Errors      int
	MaxErrors   int
	ErrorStart  time.Time
	ErrorWindow time.Duration
}

func (h *HostInstance) cleanFilename(filename string) string {
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

func (h *HostInstance) userExists(username string) bool {
	cmd := exec.Command("id", "-u", username)
	err := cmd.Run()
	return err == nil
}

func (h *HostInstance) getUIDGID(username string) (int64, int64, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return 0, 0, err
	}

	uid, err := strconv.ParseInt(u.Uid, 10, 32)
	if err != nil {
		return 0, 0, err
	}
	gid, err := strconv.ParseInt(u.Gid, 10, 32)
	if err != nil {
		return 0, 0, err
	}

	return uid, gid, nil
}

func (h *HostInstance) createUser(username, home string) (int64, int64, error) {
	if h.userExists(username) {
		return h.getUIDGID(username)
	}

	cmd := exec.Command("useradd", "-m", "-d", home, username)
	if err := cmd.Run(); err != nil {
		return 0, 0, err
	}

	return h.getUIDGID(username)
}

func (h *HostInstance) handler() {
	if !h.Locker.CompareAndSwap(0, 1) {
		return
	}
	defer h.Locker.Store(0)

	fmt.Println("Running bot for user:", h.User.Username)

	// Setup user
	username := fmt.Sprintf("%s-%d", h.cleanFilename(h.User.Username), h.User.ID)
	uid, gid, err := h.createUser(username, filepath.Join("/hosting", username))
	if err != nil {
		fmt.Println("Error creating user:", err)
		return
	}
	mainFolder := filepath.Join("/hosting", username, "LightningBot")

	// Run the executable
	h.Cmd = exec.Command("./LB-Selfbot")
	setupCmd(h.Cmd, uid, gid)
	h.Cmd.Env = append(h.Cmd.Env, fmt.Sprintf("USERNAME=%s", h.User.Username))
	h.Cmd.Env = append(h.Cmd.Env, fmt.Sprintf("PASSWORD=%s", "s]+zs,p7+X2eWY$gJ90£DjZ}RlWjk21*sbY<lE)VTWy5b!N'"))
	h.Cmd.Env = append(h.Cmd.Env, fmt.Sprintf("TOKEN=%s", h.Token))
	h.Cmd.Env = append(h.Cmd.Env, fmt.Sprintf("MAIN_FOLDER=%s", mainFolder))

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

	processKilled := make(chan int)

	h.Stdout = ""

	go func() {
		for {
			stdoutData, err := stdoutReader.ReadString('\n')
			if err != nil {
				break
			}

			h.Stdout += stdoutData
		}
		h.Cmd.Wait()
		processKilled <- h.Cmd.ProcessState.ExitCode()
	}()

	select {
	case <-h.StopChannel:
		fmt.Println("Gracefully stopping bot for user:", h.User.Username)
		h.Cmd.Process.Kill()

		return
	case exitCode := <-processKilled:
		fmt.Println("Bot for user", h.User.Username, "has exited:", h.Stdout, "with status code", exitCode)

		if exitCode != 0 {
			h.Errors++

			if time.Since(h.ErrorStart) > h.ErrorWindow {
				h.ErrorStart = time.Now()
				h.Errors = 1
			}

			reportError(h.User, h.Stdout, exitCode)
		}

		return
	}
}

func (h *HostInstance) Run(token string) {
	if token != "" {
		var err error
		h.Token, err = decrypt(token)
		if err != nil {
			fmt.Println("Error decrypting token:", err)
			return
		}
	}

	go h.handler()
}

func (h *HostInstance) Stop() {
	if h.IsRunning() {
		h.StopChannel <- struct{}{}
	}
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
		MaxErrors:   5,
		ErrorStart:  time.Now(),
		ErrorWindow: time.Minute * 10,
	}

	return Instances[user.ID]
}

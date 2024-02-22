package main

import (
	"strings"

	"github.com/getsentry/sentry-go"
)

func reportError(user *User, stdout string, containerState string) {
	// Truncate stdout to last 100 lines
	lines := strings.Split(stdout, "\n")
	if len(lines) > 100 {
		stdout = strings.Join(lines[len(lines)-100:], "\n")
	}

	sentry.CaptureEvent(&sentry.Event{
		Level:   sentry.LevelError,
		Message: "Bot exited with non-zero exit code",
		User:    sentry.User{ID: user.Username, Name: user.Username},
		Extra: map[string]any{
			"user_id":   user.ID,
			"username":  user.Username,
			"container_state": containerState,
			"stdout":    stdout,
		},
	})
}

func reportTooManyErrors(user *User) {
	sentry.CaptureEvent(&sentry.Event{
		Level:   sentry.LevelError,
		Message: "Too many errors",
		User:    sentry.User{ID: user.Username, Name: user.Username},
		Extra: map[string]any{
			"user_id":  user.ID,
			"username": user.Username,
		},
	})
}

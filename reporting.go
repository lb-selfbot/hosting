package main

import "github.com/getsentry/sentry-go"

func reportError(user *User, stdout string, exitCode int) {
	sentry.CaptureEvent(&sentry.Event{
		Level:   sentry.LevelError,
		Message: "Bot exited with non-zero exit code",
		User:    sentry.User{ID: user.Username, Name: user.Username},
		Extra: map[string]any{
			"user_id":   user.ID,
			"username":  user.Username,
			"exit_code": exitCode,
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

//go:build windows

package main

import "os/exec"

func setupCmd(cmd *exec.Cmd, uid, gid int64) {
	// No-op
}

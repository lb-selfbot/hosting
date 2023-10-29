//go:build linux

package main

import (
	"os/exec"
	"syscall"
)

func setupCmd(cmd *exec.Cmd, uid, gid int64)  {
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
}

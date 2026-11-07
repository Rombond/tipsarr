package main

import "syscall"

// childAttrs: a sandbox dies with the gateway (no orphan keeps running after a crash).
func childAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL, Setpgid: true}
}

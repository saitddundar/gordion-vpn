//go:build !windows

package cmd

import (
	"os"
	"syscall"
)

func sendInterrupt(proc *os.Process) error {
	return proc.Signal(syscall.SIGTERM)
}

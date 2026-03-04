//go:build !windows

package state

import (
	"os"
	"syscall"
)

func processExists(proc *os.Process) bool {
	err := proc.Signal(syscall.Signal(0))
	return err == nil
}

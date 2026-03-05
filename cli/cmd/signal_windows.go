//go:build windows

package cmd

import (
	"os"
)

func sendInterrupt(proc *os.Process) error {
	// Windows doesn't support SIGTERM; interrupt is the closest equivalent.
	return proc.Signal(os.Interrupt)
}

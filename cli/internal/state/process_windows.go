//go:build windows

package state

import (
	"os"

	"golang.org/x/sys/windows"
)

func processExists(proc *os.Process) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(proc.Pid))
	if err != nil {
		return false
	}
	windows.CloseHandle(h)
	return true
}

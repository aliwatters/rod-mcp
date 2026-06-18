//go:build !windows

package types

import (
	"errors"
	"syscall"
)

// processExists reports whether a process with the given pid is alive.
// On Unix, signal 0 performs existence/permission checking without delivering
// a signal: nil means the process exists and we may signal it; EPERM means it
// exists but is owned by another user.
func processExists(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

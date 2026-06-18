//go:build windows

package types

import "os"

// processExists reports whether a process with the given pid is alive.
// Windows has no signal-0 existence check; unlike Unix, os.FindProcess opens a
// real process handle via OpenProcess and returns an error when the pid does
// not refer to a live process. A successful open means the process exists; we
// release the handle immediately.
func processExists(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	_ = p.Release()
	return true
}

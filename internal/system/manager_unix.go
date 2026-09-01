//go:build unix

package system

import (
	"errors"
	"fmt"
	"syscall"
)

// UnixManager terminates processes via signals on macOS and Linux.
type UnixManager struct{}

// NewManager returns the platform Manager. On Windows it returns a
// manager that fails with a clear error (Windows is not in the MVP).
func NewManager() Manager {
	return UnixManager{}
}

// Terminate sends SIGTERM (graceful termination).
func (UnixManager) Terminate(pid int) error {
	if err := ValidatePID(pid); err != nil {
		return err
	}
	return mapSignalError(syscall.Kill(pid, syscall.SIGTERM), pid)
}

// Kill sends SIGKILL (immediate, cannot be caught).
func (UnixManager) Kill(pid int) error {
	if err := ValidatePID(pid); err != nil {
		return err
	}
	return mapSignalError(syscall.Kill(pid, syscall.SIGKILL), pid)
}

// Alive reports whether the process exists. Zombies count as alive=false
// once reaped by someone else; signal 0 only tests existence.
func (UnixManager) Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

// mapSignalError converts raw signal errors into portctl error types.
func mapSignalError(err error, pid int) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, syscall.ESRCH):
		return fmt.Errorf("process %d no longer exists: %w", pid, ErrProcessGone)
	case errors.Is(err, syscall.EPERM):
		return fmt.Errorf("%w: cannot signal process %d", ErrPermissionDenied, pid)
	case errors.Is(err, syscall.EINVAL):
		return fmt.Errorf("invalid signal for PID %d: %w", pid, err)
	default:
		return fmt.Errorf("failed to signal process %d: %w", pid, err)
	}
}

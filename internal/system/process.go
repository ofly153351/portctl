package system

import (
	"errors"
	"fmt"
)

// Sentinel errors mapped to user-facing messages and exit codes.
var (
	// ErrPermissionDenied means the OS refused the operation (EPERM/EACCES).
	ErrPermissionDenied = errors.New("permission denied")
	// ErrProcessGone means the process disappeared before we acted on it.
	ErrProcessGone = errors.New("process no longer exists")
	// ErrProtectedProcess means the PID is refused by safety policy.
	ErrProtectedProcess = errors.New("protected process")
	// ErrInvalidPID means the PID is not a positive integer.
	ErrInvalidPID = errors.New("invalid PID")
)

// ProtectedProcessError is returned when safety policy refuses to kill a
// PID. Its text matches the TASK.md wording, and errors.Is matches
// ErrProtectedProcess.
type ProtectedProcessError struct {
	PID int
}

func (e *ProtectedProcessError) Error() string {
	return fmt.Sprintf("refusing to kill protected process (PID %d)", e.PID)
}

// Unwrap makes errors.Is(err, ErrProtectedProcess) work.
func (e *ProtectedProcessError) Unwrap() error { return ErrProtectedProcess }

// Manager terminates processes. Terminate is graceful (SIGTERM), Kill is
// immediate (SIGKILL).
type Manager interface {
	Terminate(pid int) error
	Kill(pid int) error
	Alive(pid int) bool
}

// ValidatePID returns an error for PIDs that must never be signalled.
func ValidatePID(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID %d: %w", pid, ErrInvalidPID)
	}
	if IsProtected(pid) {
		return &ProtectedProcessError{PID: pid}
	}
	return nil
}

// IsProtected reports whether pid is protected by safety policy.
// PID 1 (launchd/init/systemd) is always protected.
func IsProtected(pid int) bool { return pid == 1 }

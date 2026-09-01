//go:build windows

package system

import "fmt"

// WindowsManager is a placeholder: Windows support is not in the MVP.
type WindowsManager struct{}

// NewManager returns the platform Manager.
func NewManager() Manager { return WindowsManager{} }

func (WindowsManager) Terminate(pid int) error {
	return fmt.Errorf("Windows is not supported yet (PID %d)", pid)
}

func (WindowsManager) Kill(pid int) error {
	return fmt.Errorf("Windows is not supported yet (PID %d)", pid)
}

func (WindowsManager) Alive(pid int) bool { return false }

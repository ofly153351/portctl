package system

import "errors"

// FakeManager records calls instead of signalling real processes. It is
// the mock process manager used across unit tests (TASK.md §27).
type FakeManager struct {
	Terminated []int
	Killed     []int
	AliveSet   map[int]bool
	FailTerm   map[int]error
	FailKill   map[int]error
}

// NewFakeManager returns a ready-to-use FakeManager.
func NewFakeManager() *FakeManager {
	return &FakeManager{
		AliveSet: map[int]bool{},
		FailTerm: map[int]error{},
		FailKill: map[int]error{},
	}
}

// Terminate records a graceful-termination request. A faked "process
// gone" failure also marks the PID dead, mirroring real OS semantics:
// ESRCH means the process already exited, so its port is free.
func (f *FakeManager) Terminate(pid int) error {
	if err, ok := f.FailTerm[pid]; ok {
		if errors.Is(err, ErrProcessGone) {
			f.AliveSet[pid] = false
		}
		return err
	}
	f.Terminated = append(f.Terminated, pid)
	f.AliveSet[pid] = false
	return nil
}

// Kill records a force-kill request. As with Terminate, a faked
// "process gone" failure marks the PID dead.
func (f *FakeManager) Kill(pid int) error {
	if err, ok := f.FailKill[pid]; ok {
		if errors.Is(err, ErrProcessGone) {
			f.AliveSet[pid] = false
		}
		return err
	}
	f.Killed = append(f.Killed, pid)
	f.AliveSet[pid] = false
	return nil
}

// Alive reports the faked liveness state.
func (f *FakeManager) Alive(pid int) bool {
	alive, ok := f.AliveSet[pid]
	return ok && alive
}

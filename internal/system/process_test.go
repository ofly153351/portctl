package system

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

func TestValidatePID(t *testing.T) {
	if err := ValidatePID(1); err == nil || err.Error() != "refusing to kill protected process (PID 1)" {
		t.Fatalf("PID 1: err = %v", err)
	}
	if err := ValidatePID(0); err == nil {
		t.Fatal("PID 0 must be invalid")
	}
	if err := ValidatePID(-5); err == nil {
		t.Fatal("negative PID must be invalid")
	}
	if err := ValidatePID(42); err != nil {
		t.Fatalf("PID 42 should be fine: %v", err)
	}
}

func TestIsProtected(t *testing.T) {
	if !IsProtected(1) {
		t.Fatal("PID 1 must be protected")
	}
	if IsProtected(2) {
		t.Fatal("PID 2 must not be protected")
	}
}

// --- real manager tests (signals against a throwaway child) ---

func TestRealManagerTerminate(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns a real child process")
	}
	m := NewManager()
	child := spawnChild(t, "term")
	defer child.Cleanup()
	pid := child.PID()

	if !m.Alive(pid) {
		t.Fatalf("child %d should be alive", pid)
	}
	if err := m.Terminate(pid); err != nil {
		t.Fatalf("Terminate: %v", err)
	}
	if err := child.WaitTimeout(2 * time.Second); err != nil {
		t.Fatalf("child should have exited after SIGTERM: %v", err)
	}
}

func TestRealManagerKillIgnoresTERM(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns a real child process")
	}
	m := NewManager()
	child := spawnChild(t, "ignore-term")
	defer child.Cleanup()
	pid := child.PID()

	if err := m.Terminate(pid); err != nil {
		t.Fatalf("Terminate: %v", err)
	}
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && !m.Alive(pid) {
		time.Sleep(20 * time.Millisecond)
	}
	if !m.Alive(pid) {
		t.Fatal("child ignores SIGTERM; should still be alive")
	}
	if err := m.Kill(pid); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	if err := child.WaitTimeout(2 * time.Second); err != nil {
		t.Fatalf("SIGKILL must not be ignorable: %v", err)
	}
}

func TestRealManagerRefusesPID1(t *testing.T) {
	// Safety layer must reject PID 1 before any syscall happens.
	m := NewManager()
	err := m.Terminate(1)
	if !errors.Is(err, ErrProtectedProcess) {
		t.Fatalf("err = %v, want protected-process refusal", err)
	}
	err = m.Kill(1)
	if !errors.Is(err, ErrProtectedProcess) {
		t.Fatalf("Kill(1) err = %v, want protected-process refusal", err)
	}
}

func TestRealManagerGone(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns a real child process")
	}
	m := NewManager()
	child := spawnChild(t, "term")
	pid := child.PID()
	_ = child.Cmd().Process.Kill()
	_, _ = child.Cmd().Process.Wait()
	err := m.Terminate(pid)
	if !errors.Is(err, ErrProcessGone) {
		t.Fatalf("err = %v, want ErrProcessGone", err)
	}
}

// --- helpers ---

type childProc struct {
	cmd *exec.Cmd
}

func (c *childProc) PID() int       { return c.cmd.Process.Pid }
func (c *childProc) Cmd() *exec.Cmd { return c.cmd }
func (c *childProc) Cleanup() {
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		_, _ = c.cmd.Process.Wait()
	}
}

// WaitTimeout waits for the child to exit within d, returning an error
// when it is still running afterwards.
func (c *childProc) WaitTimeout(d time.Duration) error {
	done := make(chan error, 1)
	go func() { done <- c.cmd.Wait() }()
	select {
	case <-done:
		return nil
	case <-time.After(d):
		return errors.New("child still running after deadline")
	}
}

func spawnChild(t *testing.T, mode string) *childProc {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", mode)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("spawn helper: %v", err)
	}
	return &childProc{cmd: cmd}
}

// TestHelperProcess is not a real test; it is the child body used by
// spawnChild. It never runs unless GO_WANT_HELPER_PROCESS=1.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	mode := args[len(args)-1]
	switch mode {
	case "ignore-term":
		signal.Ignore(syscall.SIGTERM)
		select {}
	case "term":
		// default: die on SIGTERM (Go default behavior)
		select {}
	}
	os.Exit(0)
}

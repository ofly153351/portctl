package app

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ofly153351/portctl/internal/ports"
	"github.com/ofly153351/portctl/internal/system"
	"github.com/ofly153351/portctl/internal/ui"
)

// fakeFinder is the mock port detector for unit tests. It simulates the
// real world: once the manager marks a PID dead, its port is free.
type fakeFinder struct {
	list  []ports.Process
	mgr   *system.FakeManager
	calls int
}

func (f *fakeFinder) ListPorts() ([]ports.Process, error) { return f.live(), nil }

func (f *fakeFinder) live() []ports.Process {
	if f.mgr == nil {
		return f.list
	}
	var out []ports.Process
	for _, p := range f.list {
		if f.mgr.Alive(p.PID) {
			out = append(out, p)
		}
	}
	return out
}

func (f *fakeFinder) FindByPort(port int) ([]ports.Process, error) {
	f.calls++
	var out []ports.Process
	for _, p := range f.live() {
		if p.Port == port {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("port %d: %w", port, ports.ErrPortNotInUse)
	}
	return out, nil
}

// scriptPrompter returns canned answers, matching a keyword in the prompt.
type scriptPrompter struct {
	answers map[string]bool
}

func (s *scriptPrompter) Confirm(msg string) bool {
	if v, ok := s.answers["*"]; ok {
		return v
	}
	for key, v := range s.answers {
		if strings.Contains(msg, key) {
			return v
		}
	}
	return false
}

func newTestEnv(list []ports.Process, answers map[string]bool) (*appImpl, *system.FakeManager, *strings.Builder) {
	var buf strings.Builder
	printer := ui.NewPrinter(&buf, false)
	mgr := system.NewFakeManager()
	for _, p := range list {
		mgr.AliveSet[p.PID] = true
	}
	finder := &fakeFinder{list: list, mgr: mgr}
	app := New(finder, mgr, printer).(*appImpl)
	app.SetPrompter(&scriptPrompter{answers: answers})
	return app, mgr, &buf
}

func sampleList() []ports.Process {
	return []ports.Process{
		{PID: 18231, Name: "node", User: "obx", Port: 3000, Protocol: "TCP", Address: "127.0.0.1"},
		{PID: 19342, Name: "pos-backend", User: "obx", Port: 8080, Protocol: "TCP", Address: "0.0.0.0"},
	}
}

func TestKillSinglePort(t *testing.T) {
	a, mgr, buf := newTestEnv(sampleList(), map[string]bool{"*": true})
	if err := a.KillPorts([]int{3000}, Opts{}); err != nil {
		t.Fatalf("KillPorts: %v", err)
	}
	if len(mgr.Terminated) != 1 || mgr.Terminated[0] != 18231 {
		t.Fatalf("terminate calls = %v, want [18231]", mgr.Terminated)
	}
	if len(mgr.Killed) != 0 {
		t.Fatalf("kill calls = %v, want none (graceful default)", mgr.Killed)
	}
	if !strings.Contains(buf.String(), "Kill process 18231?") {
		t.Fatalf("missing confirmation prompt: %q", buf.String())
	}
}

func TestKillDeclined(t *testing.T) {
	a, mgr, _ := newTestEnv(sampleList(), map[string]bool{"*": false})
	if err := a.KillPorts([]int{3000}, Opts{}); err != nil {
		t.Fatalf("declining is not an error: %v", err)
	}
	if len(mgr.Terminated) != 0 {
		t.Fatal("nothing should be terminated when declined")
	}
}

func TestKillForceSkipsConfirmAndUsesSIGKILL(t *testing.T) {
	a, mgr, buf := newTestEnv(sampleList(), map[string]bool{})
	if err := a.KillPorts([]int{3000}, Opts{Force: true}); err != nil {
		t.Fatalf("KillPorts: %v", err)
	}
	if len(mgr.Killed) != 1 || mgr.Killed[0] != 18231 {
		t.Fatalf("kill calls = %v, want [18231]", mgr.Killed)
	}
	if len(mgr.Terminated) != 0 {
		t.Fatalf("terminate calls = %v, want none", mgr.Terminated)
	}
	if strings.Contains(buf.String(), "Kill process") {
		t.Fatalf("force must not ask: %q", buf.String())
	}
}

func TestKillMultiplePorts(t *testing.T) {
	a, mgr, buf := newTestEnv(sampleList(), map[string]bool{"*": true})
	if err := a.KillPorts([]int{3000, 8080}, Opts{}); err != nil {
		t.Fatalf("KillPorts: %v", err)
	}
	if len(mgr.Terminated) != 2 {
		t.Fatalf("want 2 terminations, got %v", mgr.Terminated)
	}
	if !strings.Contains(buf.String(), "Port 3000") || !strings.Contains(buf.String(), "Port 8080") {
		t.Fatalf("both ports should be listed: %q", buf.String())
	}
}

func TestKillSkipsPortNotInUseAndContinues(t *testing.T) {
	a, mgr, buf := newTestEnv(sampleList(), map[string]bool{"*": true})
	if err := a.KillPorts([]int{3000, 9999, 8080}, Opts{}); err != nil {
		t.Fatalf("a free port must not abort the run: %v", err)
	}
	if len(mgr.Terminated) != 2 {
		t.Fatalf("want 2 terminations, got %v", mgr.Terminated)
	}
	if !strings.Contains(buf.String(), "Port 9999 is not in use.") {
		t.Fatalf("must report the unused port: %q", buf.String())
	}
}

func TestKillProtectedPID(t *testing.T) {
	list := []ports.Process{{PID: 1, Name: "launchd", Port: 3000, Protocol: "TCP"}}
	a, mgr, _ := newTestEnv(list, map[string]bool{"*": true})
	err := a.KillPorts([]int{3000}, Opts{Force: true})
	if err == nil {
		t.Fatal("must refuse killing PID 1")
	}
	if !errors.Is(err, system.ErrProtectedProcess) {
		t.Fatalf("err = %v, want ErrProtectedProcess", err)
	}
	if len(mgr.Killed) != 0 {
		t.Fatal("no signal must be sent")
	}
}

func TestKillGoneBetweenFindAndKill(t *testing.T) {
	a, mgr, buf := newTestEnv(sampleList(), map[string]bool{"*": true})
	mgr.FailTerm[18231] = system.ErrProcessGone
	if err := a.KillPorts([]int{3000}, Opts{}); err != nil {
		t.Fatalf("gone process should not fail the run: %v", err)
	}
	if !strings.Contains(buf.String(), "18231 no longer exists") {
		t.Fatalf("must explain the gone process: %q", buf.String())
	}
}

func TestFreeFlow(t *testing.T) {
	a, mgr, buf := newTestEnv(sampleList(), map[string]bool{"*": true})
	if err := a.FreePort(3000, Opts{}); err != nil {
		t.Fatalf("FreePort: %v", err)
	}
	if len(mgr.Terminated) != 1 || mgr.Terminated[0] != 18231 {
		t.Fatalf("terminated = %v", mgr.Terminated)
	}
	out := buf.String()
	for _, want := range []string{"Port 3000 is used by", "18231", "node"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in free output: %q", want, out)
		}
	}
}

func TestFreeFlowVerify(t *testing.T) {
	a, _, buf := newTestEnv(sampleList(), map[string]bool{"*": true})
	if err := a.FreePort(3000, Opts{}); err != nil {
		t.Fatalf("FreePort: %v", err)
	}
	if !strings.Contains(buf.String(), "Port 3000 is free") {
		t.Fatalf("free must verify and report: %q", buf.String())
	}
}

func TestFreeFlowVerifyStillUsed(t *testing.T) {
	// A terminate that fails with EPERM leaves the port in use: free
	// must report the failure and return an error.
	var buf strings.Builder
	printer := ui.NewPrinter(&buf, false)
	mgr := system.NewFakeManager()
	mgr.AliveSet[18231] = true
	mgr.FailTerm[18231] = system.ErrPermissionDenied
	finder := &fakeFinder{list: sampleList(), mgr: mgr}
	a := New(finder, mgr, printer).(*appImpl)
	a.SetPrompter(&scriptPrompter{answers: map[string]bool{"*": true}})
	err := a.FreePort(3000, Opts{})
	if err == nil {
		t.Fatal("port still in use after failed kill must be an error")
	}
	if !strings.Contains(buf.String(), "still in use") {
		t.Fatalf("must report the stubborn port: %q", buf.String())
	}
}

func TestInfoSingle(t *testing.T) {
	a, _, buf := newTestEnv(sampleList(), nil)
	if err := a.Info(3000, Opts{}); err != nil {
		t.Fatalf("Info: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Port:", "3000", "Protocol:", "TCP", "PID:", "18231", "Process:", "node", "User:", "obx"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q: %q", want, out)
		}
	}
}

func TestInfoMultipleProcessesSamePort(t *testing.T) {
	list := []ports.Process{
		{PID: 18231, Name: "node", Port: 3000, Protocol: "TCP", Address: "127.0.0.1"},
		{PID: 19420, Name: "node", Port: 3000, Protocol: "TCP", Address: "127.0.0.1"},
	}
	a, _, buf := newTestEnv(list, nil)
	if err := a.Info(3000, Opts{}); err != nil {
		t.Fatalf("Info: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Port: 3000") {
		t.Fatalf("multi-process header missing: %q", out)
	}
	for _, want := range []string{"18231", "19420"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing PID %s: %q", want, out)
		}
	}
}

func TestInfoNotInUse(t *testing.T) {
	a, _, buf := newTestEnv(sampleList(), nil)
	if err := a.Info(9999, Opts{}); err != nil {
		t.Fatalf("not-in-use info is not an error: %v", err)
	}
	if !strings.Contains(buf.String(), "Port 9999 is not in use.") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestInfoJSON(t *testing.T) {
	a, _, buf := newTestEnv(sampleList(), nil)
	if err := a.Info(3000, Opts{JSON: true}); err != nil {
		t.Fatalf("Info: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "\"port\": 3000") {
		t.Fatalf("json info output = %q", out)
	}
	if strings.Contains(out, "Port:") {
		t.Fatalf("json mode must not print the text view: %q", out)
	}
}

func TestListPortsTable(t *testing.T) {
	a, _, buf := newTestEnv(sampleList(), nil)
	if err := a.ListPorts(Opts{}); err != nil {
		t.Fatalf("ListPorts: %v", err)
	}
	if !strings.Contains(buf.String(), "PORT") || !strings.Contains(buf.String(), "node") {
		t.Fatalf("table output = %q", buf.String())
	}
}

func TestListPortsJSON(t *testing.T) {
	a, _, buf := newTestEnv(sampleList(), nil)
	if err := a.ListPorts(Opts{JSON: true}); err != nil {
		t.Fatalf("ListPorts: %v", err)
	}
	if !strings.Contains(buf.String(), "\"port\": 3000") {
		t.Fatalf("json = %q", buf.String())
	}
}

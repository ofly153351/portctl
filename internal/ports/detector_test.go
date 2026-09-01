package ports

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// fakeRunner lets detector tests inject canned lsof output.
type fakeRunner struct {
	stdout string
	err    error
	calls  []string
}

func (f *fakeRunner) run(name string, args ...string) (string, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	return f.stdout, f.err
}

func TestDetectorListPortsUsesLsofOnce(t *testing.T) {
	f := &fakeRunner{stdout: fixtureMacOSField}
	d := newLsofDetectorWithRunner(f)
	ps, err := d.ListPorts()
	if err != nil {
		t.Fatalf("ListPorts: %v", err)
	}
	if len(f.calls) != 1 {
		t.Fatalf("lsof called %d times, want exactly 1 (performance)", len(f.calls))
	}
	if !strings.HasPrefix(f.calls[0], "lsof -nP -i ") {
		t.Fatalf("unexpected lsof invocation: %q", f.calls[0])
	}
	// fixture has: UDP(61367) + TCP LISTEN(52309) + ESTABLISHED(53953,
	// must be dropped) + LISTEN(53977) + LISTEN 8081 x2 (v4+v6 kept).
	var gotPorts []int
	for _, p := range ps {
		gotPorts = append(gotPorts, p.Port)
	}
	want := []int{8081, 8081, 52309, 53977, 61367}
	if len(ps) != len(want) {
		t.Fatalf("got %d entries (%v), want %d", len(ps), gotPorts, len(want))
	}
	for i := range want {
		if gotPorts[i] != want[i] {
			t.Fatalf("entry %d port = %d, want %d (sorted asc)", i, gotPorts[i], want[i])
		}
	}
}

func TestDetectorListPortsDropsEstablished(t *testing.T) {
	f := &fakeRunner{stdout: fixtureMacOSField}
	d := newLsofDetectorWithRunner(f)
	ps, _ := d.ListPorts()
	for _, p := range ps {
		if p.State == "ESTABLISHED" {
			t.Fatalf("established connection leaked into listing: %+v", p)
		}
		if p.Protocol == "UDP" && strings.Contains(p.RawName(), "->") {
			t.Fatalf("connected UDP socket leaked into listing: %+v", p)
		}
	}
}

func TestDetectorFindByPortFound(t *testing.T) {
	f := &fakeRunner{stdout: fixtureMacOSField}
	d := newLsofDetectorWithRunner(f)
	ps, err := d.FindByPort(8081)
	if err != nil {
		t.Fatalf("FindByPort(8081): %v", err)
	}
	if len(ps) != 2 {
		t.Fatalf("len = %d, want 2 (v4+v6)", len(ps))
	}
}

func TestDetectorFindByPortNotInUse(t *testing.T) {
	f := &fakeRunner{stdout: fixtureMacOSField}
	d := newLsofDetectorWithRunner(f)
	_, err := d.FindByPort(9999)
	if !errors.Is(err, ErrPortNotInUse) {
		t.Fatalf("err = %v, want ErrPortNotInUse", err)
	}
	if !strings.Contains(err.Error(), "9999") {
		t.Fatalf("error should mention the port: %v", err)
	}
}

func TestDetectorLsofExit1NoStderrMeansEmpty(t *testing.T) {
	// lsof exits 1 when nothing matched; that is not an error for us.
	f := &fakeRunner{err: &exec.ExitError{Stderr: nil}}
	// NOTE: ExitError without a ProcessState is unusual; craft via exec?
	// Simpler: use a real exit error produced by /usr/bin/false.
	f.err = func() error {
		out, err := exec.Command("/usr/bin/false").Output()
		_ = out
		return err
	}()
	d := newLsofDetectorWithRunner(f)
	ps, err := d.ListPorts()
	if err != nil {
		t.Fatalf("exit-1-no-stderr should yield empty list, got err %v", err)
	}
	if len(ps) != 0 {
		t.Fatalf("want empty list, got %v", ps)
	}
}

func TestDetectorLsofMissing(t *testing.T) {
	f := &fakeRunner{err: exec.ErrNotFound}
	d := newLsofDetectorWithRunner(f)
	_, err := d.ListPorts()
	if !errors.Is(err, ErrLsofUnavailable) {
		t.Fatalf("err = %v, want ErrLsofUnavailable", err)
	}
}

func TestDetectorLsofRealStderrFails(t *testing.T) {
	f := &fakeRunner{err: errors.New("boom")}
	d := newLsofDetectorWithRunner(f)
	_, err := d.ListPorts()
	if err == nil {
		t.Fatal("want error")
	}
	if !strings.Contains(err.Error(), "lsof") {
		t.Fatalf("error should mention lsof: %v", err)
	}
}

func TestFilterEndpointsRules(t *testing.T) {
	ps := []Process{
		{PID: 1, Port: 100, Protocol: "TCP", State: "LISTEN"},
		{PID: 1, Port: 101, Protocol: "TCP", State: "ESTABLISHED"},
		{PID: 1, Port: 102, Protocol: "TCP", State: ""},
		{PID: 1, Port: 103, Protocol: "UDP", rawName: "127.0.0.1:103"},
		{PID: 1, Port: 104, Protocol: "UDP", rawName: "127.0.0.1:104->8.8.8.8:53"},
		{PID: 1, Port: 105, Protocol: "TCP", State: "CLOSE_WAIT"},
	}
	got := filterEndpoints(ps)
	wantPorts := []int{100, 102, 103}
	if len(got) != len(wantPorts) {
		t.Fatalf("got %d, want %d: %+v", len(got), len(wantPorts), got)
	}
	for i, p := range got {
		if p.Port != wantPorts[i] {
			t.Fatalf("entry %d = port %d, want %d", i, p.Port, wantPorts[i])
		}
	}
}

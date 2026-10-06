package cmd

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ofly153351/portctl/internal/app"
	"github.com/ofly153351/portctl/internal/dnsflush"
	"github.com/ofly153351/portctl/internal/ports"
	"github.com/ofly153351/portctl/internal/system"
	"github.com/ofly153351/portctl/internal/ui"
)

// stubApp records calls made by the CLI layer.
type stubApp struct {
	listCalls int
	infoPort  int
	infoCalls int
	killPorts []int
	freePort  int
	freeCalls int
	lastOpts  app.Opts
	err       error
}

func (s *stubApp) ListPorts(opts app.Opts) error {
	s.listCalls++
	s.lastOpts = opts
	return s.err
}

func (s *stubApp) Info(port int, opts app.Opts) error {
	s.infoCalls++
	s.infoPort = port
	s.lastOpts = opts
	return s.err
}

func (s *stubApp) KillPorts(pl []int, opts app.Opts) error {
	s.killPorts = append(s.killPorts, pl...)
	s.lastOpts = opts
	return s.err
}

func (s *stubApp) FreePort(port int, opts app.Opts) error {
	s.freeCalls++
	s.freePort = port
	s.lastOpts = opts
	return s.err
}

// ExitError pairs an error with its process exit code.
type ExitError struct {
	Code int
	Msg  string
}

func (e *ExitError) Error() string { return e.Msg }

func runForTest(t *testing.T, args ...string) (*stubApp, string, int) {
	t.Helper()
	var buf strings.Builder
	stub := &stubApp{}
	printer := ui.NewPrinter(&buf, false)
	code := Execute(args, stub, printer)
	return stub, buf.String(), code
}

func TestNoArgsShowsHelp(t *testing.T) {
	_, out, code := runForTest(t)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(out, "portctl - Port & Process Manager") {
		t.Fatalf("help missing: %q", out)
	}
	if !strings.Contains(out, "Usage:") || !strings.Contains(out, "Examples:") {
		t.Fatalf("help sections missing: %q", out)
	}
}

func TestHelpFlag(t *testing.T) {
	for _, arg := range []string{"--help", "-h", "help"} {
		_, out, code := runForTest(t, arg)
		if code != 0 {
			t.Fatalf("%s: code = %d", arg, code)
		}
		if !strings.Contains(out, "Usage:") {
			t.Fatalf("%s: no usage: %q", arg, out)
		}
	}
}

func TestVersionFlag(t *testing.T) {
	_, out, code := runForTest(t, "--version")
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(out, "portctl version ") {
		t.Fatalf("version output: %q", out)
	}
}

func TestUnknownCommand(t *testing.T) {
	_, out, code := runForTest(t, "foo")
	if code != ExitUsage {
		t.Fatalf("code = %d, want ExitUsage", code)
	}
	if !strings.Contains(out, `unknown command "foo"`) {
		t.Fatalf("output: %q", out)
	}
	if !strings.Contains(out, "portctl --help") {
		t.Fatalf("must point to help: %q", out)
	}
}

func TestLsCommand(t *testing.T) {
	stub, out, code := runForTest(t, "ls")
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if stub.listCalls != 1 {
		t.Fatalf("ListPorts calls = %d", stub.listCalls)
	}
	if strings.Contains(out, "Error") {
		t.Fatalf("unexpected error text: %q", out)
	}
}

func TestLsJSONFlag(t *testing.T) {
	stub, _, code := runForTest(t, "ls", "--json")
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if !stub.lastOpts.JSON {
		t.Fatal("--json must set Opts.JSON")
	}
}

func TestInfoRequiresPort(t *testing.T) {
	_, out, code := runForTest(t, "info")
	if code != ExitUsage {
		t.Fatalf("code = %d, want ExitUsage", code)
	}
	if !strings.Contains(out, "Error:") {
		t.Fatalf("output: %q", out)
	}
}

func TestInfoInvalidPort(t *testing.T) {
	_, out, code := runForTest(t, "info", "abc")
	if code != ExitUsage {
		t.Fatalf("code = %d, want ExitUsage", code)
	}
	if !strings.Contains(out, `invalid port "abc"`) {
		t.Fatalf("output: %q", out)
	}
}

func TestInfoValidPort(t *testing.T) {
	stub, _, code := runForTest(t, "info", "3000")
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if stub.infoPort != 3000 {
		t.Fatalf("infoPort = %d", stub.infoPort)
	}
}

func TestInfoOutOfRangePort(t *testing.T) {
	for _, raw := range []string{"0", "-1", "65536", "99999"} {
		_, _, code := runForTest(t, "info", raw)
		if code != ExitUsage {
			t.Fatalf("info %s: code = %d, want ExitUsage", raw, code)
		}
	}
}

func TestKillRequiresPort(t *testing.T) {
	_, _, code := runForTest(t, "kill")
	if code != ExitUsage {
		t.Fatalf("code = %d", code)
	}
}

func TestKillInvalidPort(t *testing.T) {
	_, out, code := runForTest(t, "kill", "abc")
	if code != ExitUsage {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(out, `invalid port "abc"`) {
		t.Fatalf("output: %q", out)
	}
}

func TestKillMultiple(t *testing.T) {
	stub, _, code := runForTest(t, "kill", "3000", "8080", "4000")
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	want := []int{3000, 8080, 4000}
	if len(stub.killPorts) != 3 {
		t.Fatalf("killPorts = %v", stub.killPorts)
	}
	for i, p := range want {
		if stub.killPorts[i] != p {
			t.Fatalf("killPorts[%d] = %d, want %d", i, stub.killPorts[i], p)
		}
	}
}

func TestKillFlags(t *testing.T) {
	stub, _, code := runForTest(t, "kill", "3000", "--force", "--quiet")
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if !stub.lastOpts.Force || !stub.lastOpts.Quiet {
		t.Fatalf("opts = %+v", stub.lastOpts)
	}
	// short forms
	stub2, _, code2 := runForTest(t, "kill", "3000", "-f", "-q")
	if code2 != 0 {
		t.Fatalf("code = %d", code2)
	}
	if !stub2.lastOpts.Force || !stub2.lastOpts.Quiet {
		t.Fatalf("opts = %+v", stub2.lastOpts)
	}
}

func TestFlagsBeforePort(t *testing.T) {
	// portctl kill -f 3000 must also work
	stub, _, code := runForTest(t, "kill", "-f", "3000")
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if !stub.lastOpts.Force || len(stub.killPorts) != 1 || stub.killPorts[0] != 3000 {
		t.Fatalf("opts=%+v ports=%v", stub.lastOpts, stub.killPorts)
	}
}

func TestAliases(t *testing.T) {
	stub, _, code := runForTest(t, "l")
	if code != 0 || stub.listCalls != 1 {
		t.Fatalf("alias l: code=%d calls=%d", code, stub.listCalls)
	}
	stub2, _, code2 := runForTest(t, "i", "3000")
	if code2 != 0 || stub2.infoPort != 3000 {
		t.Fatalf("alias i: code=%d port=%d", code2, stub2.infoPort)
	}
	stub3, _, code3 := runForTest(t, "k", "3000", "-f")
	if code3 != 0 || !stub3.lastOpts.Force || stub3.killPorts[0] != 3000 {
		t.Fatalf("alias k: code=%d opts=%+v", code3, stub3.lastOpts)
	}
	stub4, _, code4 := runForTest(t, "f", "3000")
	if code4 != 0 || stub4.freePort != 3000 {
		t.Fatalf("alias f: code=%d port=%d", code4, stub4.freePort)
	}
}

func TestExitCodeGeneralError(t *testing.T) {
	var buf strings.Builder
	stub := &stubApp{err: errors.New("boom")}
	printer := ui.NewPrinter(&buf, false)
	code := Execute([]string{"ls"}, stub, printer)
	if code != ExitGeneral {
		t.Fatalf("code = %d, want ExitGeneral", code)
	}
	if !strings.Contains(buf.String(), "boom") {
		t.Fatalf("error must be shown: %q", buf.String())
	}
}

func TestExitCodePermissionDenied(t *testing.T) {
	var buf strings.Builder
	stub := &stubApp{err: system.ErrPermissionDenied}
	printer := ui.NewPrinter(&buf, false)
	code := Execute([]string{"kill", "3000"}, stub, printer)
	if code != ExitPermission {
		t.Fatalf("code = %d, want ExitPermission", code)
	}
	if !strings.Contains(buf.String(), "sudo") {
		t.Fatalf("must suggest sudo: %q", buf.String())
	}
}

func TestExitCodePortNotInUseIsZero(t *testing.T) {
	// info on a free port is a successful command (TASK.md §19).
	var buf strings.Builder
	stub := &stubApp{err: fmt.Errorf("port 3000: %w", ports.ErrPortNotInUse)}
	printer := ui.NewPrinter(&buf, false)
	code := Execute([]string{"info", "3000"}, stub, printer)
	if code != 0 {
		t.Fatalf("code = %d, want 0 (port-not-in-use is not an error)", code)
	}
}

func TestDNSFlushUnsupportedOnNonMacOS(t *testing.T) {
	var buf strings.Builder
	printer := ui.NewPrinter(&buf, false)
	calls := 0
	err := dnsflush.Flush("linux", func(string, ...string) error {
		calls++
		return nil
	}, printer.Success)
	if err == nil || !strings.Contains(err.Error(), "supported only on macOS") {
		t.Fatalf("error = %v, want macOS-only message", err)
	}
	if calls != 0 {
		t.Fatalf("commands executed = %d, want 0", calls)
	}
}

func TestDNSFlushRequiresFlushSubcommand(t *testing.T) {
	_, out, code := runForTest(t, "dns")
	if code != ExitUsage {
		t.Fatalf("code = %d, want ExitUsage", code)
	}
	if !strings.Contains(out, "usage: portctl dns flush") {
		t.Fatalf("output = %q", out)
	}
}

func TestDNSFlushUsesTheExpectedCommandsOnMacOS(t *testing.T) {
	oldFlusher, oldRun := activeDNSFlusher, dnsRun
	t.Cleanup(func() {
		activeDNSFlusher, dnsRun = oldFlusher, oldRun
	})
	activeDNSFlusher = dnsFlusherFunc(func(goos string, run dnsflush.Runner, report func(string)) error {
		return dnsflush.Flush("darwin", run, report)
	})
	var calls [][]string
	dnsRun = func(name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	}
	var buf strings.Builder
	printer := ui.NewPrinter(&buf, false)
	code := Execute([]string{"dns", "flush"}, &stubApp{}, printer)
	if code != ExitSuccess {
		t.Fatalf("code = %d, want ExitSuccess; output=%q", code, buf.String())
	}
	wantCalls := [][]string{{"sudo", "dscacheutil", "-flushcache"}, {"sudo", "killall", "-HUP", "mDNSResponder"}}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("commands = %v, want %v", calls, wantCalls)
	}
	if !strings.Contains(buf.String(), "DNS cache flushed successfully") || !strings.Contains(buf.String(), "mDNSResponder restarted") {
		t.Fatalf("success output = %q", buf.String())
	}
}

type dnsFlusherFunc func(string, dnsflush.Runner, func(string)) error

func (f dnsFlusherFunc) Flush(goos string, run dnsflush.Runner, report func(string)) error {
	return f(goos, run, report)
}

func TestUnknownFlag(t *testing.T) {
	_, _, code := runForTest(t, "ls", "--bogus")
	if code != ExitUsage {
		t.Fatalf("code = %d, want ExitUsage", code)
	}
}

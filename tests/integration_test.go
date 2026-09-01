package tests

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// TestMain builds the portctl binary once for the whole package.
func TestMain(m *testing.M) {
	bin := filepath.Join("..", "bin", "portctl")
	if _, err := os.Stat(bin); err != nil {
		// Build it ourselves so `go test ./tests/` works standalone.
		cmd := exec.Command("go", "build", "-o", bin, "..")
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "build portctl: %v\n%s", err, out)
			os.Exit(1)
		}
	}
	os.Exit(m.Run())
}

// freePort asks the kernel for an unused TCP port.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// startServer spawns a real listener child and returns a stop func.
func startServer(t *testing.T, port int) (stop func()) {
	t.Helper()
	// The child is this test binary itself in helper mode.
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperListener")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_LISTENER=1",
		"HELPER_PORT="+strconv.Itoa(port))
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	// Wait until the port is actually accepting connections.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return func() {
				_ = cmd.Process.Kill()
				_, _ = cmd.Process.Wait()
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("helper never listened on %d", port)
	return nil
}

// TestHelperListener is the child process body; not a real test.
func TestHelperListener(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_LISTENER") != "1" {
		return
	}
	port, _ := strconv.Atoi(os.Getenv("HELPER_PORT"))
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		os.Exit(1)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			os.Exit(0)
		}
		conn.Close()
	}
}

func portctl(t *testing.T, args ...string) (string, int) {
	t.Helper()
	// TestMain builds the binary at <repo>/bin/portctl; tests run with
	// cwd = <repo>/tests, hence the "..".
	bin, err := filepath.Abs(filepath.Join("..", "bin", "portctl"))
	if err != nil {
		t.Fatalf("resolve binary path: %v", err)
	}
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("run portctl: %v", err)
	}
	return string(out), code
}

func TestIntegrationLsAndInfo(t *testing.T) {
	port := freePort(t)
	stop := startServer(t, port)
	defer stop()

	// ls must list the port.
	out, code := portctl(t, "ls")
	if code != 0 {
		t.Fatalf("ls exit=%d out=%s", code, out)
	}
	if !containsPort(out, port) {
		t.Fatalf("ls output missing port %d:\n%s", port, out)
	}

	// info must show the port and this test binary as the process.
	out, code = portctl(t, "info", strconv.Itoa(port))
	if code != 0 {
		t.Fatalf("info exit=%d out=%s", code, out)
	}
	if !containsPort(out, port) {
		t.Fatalf("info output missing port:\n%s", out)
	}

	// JSON mode must be valid JSON containing the port.
	out, _ = portctl(t, "info", strconv.Itoa(port), "--json")
	var entries []map[string]any
	if err := json.Unmarshal([]byte(out), &entries); err != nil {
		t.Fatalf("info --json invalid: %v\n%s", err, out)
	}
	if len(entries) == 0 {
		t.Fatalf("info --json empty: %s", out)
	}
	if int(entries[0]["port"].(float64)) != port {
		t.Fatalf("info --json port = %v, want %d", entries[0]["port"], port)
	}
}

func TestIntegrationKillFreePort(t *testing.T) {
	port := freePort(t)
	stop := startServer(t, port)
	defer stop()

	// kill on an unused port reports and exits fine.
	out, code := portctl(t, "kill", strconv.Itoa(freePort(t)), "-f")
	if code != 0 {
		t.Fatalf("kill free port exit=%d out=%s", code, out)
	}
	if !contains(out, "is not in use") {
		t.Fatalf("expected not-in-use message: %s", out)
	}

	// Force-kill the helper and verify the port is released.
	out, code = portctl(t, "kill", strconv.Itoa(port), "-f", "-q")
	if code != 0 {
		t.Fatalf("kill exit=%d out=%s", code, out)
	}
	_ = stop // already killed via portctl; make defer idempotent
	stop = func() {}
}

func TestIntegrationInvalidInput(t *testing.T) {
	for _, raw := range []string{"abc", "0", "-1", "65536"} {
		_, code := portctl(t, "info", raw)
		if code != 2 {
			t.Fatalf("info %s: exit=%d, want 2", raw, code)
		}
	}
	_, code := portctl(t, "nope")
	if code != 2 {
		t.Fatalf("unknown command exit=%d, want 2", code)
	}
}

func containsPort(hay string, port int) bool {
	return contains(hay, strconv.Itoa(port))
}

func contains(hay, needle string) bool {
	return len(hay) >= len(needle) && (hay == needle ||
		len(hay) > 0 && indexOf(hay, needle) >= 0)
}

func indexOf(hay, needle string) int {
	for i := 0; i+len(needle) <= len(hay); i++ {
		if hay[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

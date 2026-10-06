package dnsflush

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Runner executes a command directly, without invoking a shell.
type Runner func(name string, args ...string) error

// Flush clears the macOS DNS cache and reloads mDNSResponder. On success,
// report is called once after each operation completes.
func Flush(goos string, run Runner, report func(string)) error {
	if goos != "darwin" {
		return fmt.Errorf("DNS cache flush is currently supported only on macOS")
	}
	if run == nil {
		run = func(name string, args ...string) error {
			return exec.Command(name, args...).Run()
		}
	}
	if report == nil {
		report = func(string) {}
	}

	if err := run("sudo", "dscacheutil", "-flushcache"); err != nil {
		return fmt.Errorf("failed to flush DNS cache (try running with sudo): %w", err)
	}
	report("DNS cache flushed successfully")

	if err := run("sudo", "killall", "-HUP", "mDNSResponder"); err != nil {
		return fmt.Errorf("failed to restart mDNSResponder (try running with sudo): %w", err)
	}
	report("mDNSResponder restarted")
	return nil
}

// FlushCurrentOS flushes using the current platform.
func FlushCurrentOS(run Runner, report func(string)) error {
	return Flush(runtime.GOOS, run, report)
}

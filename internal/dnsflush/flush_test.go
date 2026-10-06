package dnsflush

import (
	"errors"
	"reflect"
	"testing"
)

func TestFlushRunsBothCommandsOnMacOS(t *testing.T) {
	var got [][]string
	var messages []string
	run := func(name string, args ...string) error {
		got = append(got, append([]string{name}, args...))
		return nil
	}

	if err := Flush("darwin", run, func(message string) { messages = append(messages, message) }); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}

	wantCommands := [][]string{
		{"sudo", "dscacheutil", "-flushcache"},
		{"sudo", "killall", "-HUP", "mDNSResponder"},
	}
	if !reflect.DeepEqual(got, wantCommands) {
		t.Fatalf("commands = %v, want %v", got, wantCommands)
	}
	wantMessages := []string{"DNS cache flushed successfully", "mDNSResponder restarted"}
	if !reflect.DeepEqual(messages, wantMessages) {
		t.Fatalf("messages = %v, want %v", messages, wantMessages)
	}
}

func TestFlushRejectsNonMacOSWithoutRunningCommands(t *testing.T) {
	called := false
	err := Flush("linux", func(string, ...string) error {
		called = true
		return nil
	}, func(string) {})
	if err == nil || err.Error() != "DNS cache flush is currently supported only on macOS" {
		t.Fatalf("error = %v, want macOS-only message", err)
	}
	if called {
		t.Fatal("must not run system commands on non-macOS")
	}
}

func TestFlushStopsWhenCommandFails(t *testing.T) {
	wantErr := errors.New("sudo failed")
	calls := 0
	var messages []string
	err := Flush("darwin", func(string, ...string) error {
		calls++
		if calls == 2 {
			return wantErr
		}
		return nil
	}, func(message string) { messages = append(messages, message) })
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped sudo error", err)
	}
	if calls != 2 {
		t.Fatalf("command calls = %d, want 2", calls)
	}
	if !reflect.DeepEqual(messages, []string{"DNS cache flushed successfully"}) {
		t.Fatalf("messages = %v", messages)
	}
}

package ui

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ofly153351/portctl/internal/ports"
)

func sample() []ports.Process {
	return []ports.Process{
		{PID: 18231, Name: "node", User: "obx", Port: 3000, Protocol: "TCP", Address: "127.0.0.1", State: "LISTEN"},
		{PID: 19342, Name: "pos-backend", User: "obx", Port: 8080, Protocol: "TCP", Address: "0.0.0.0", State: "LISTEN"},
		{PID: 1298, Name: "postgres", User: "postgres", Port: 5432, Protocol: "UDP", Address: "127.0.0.1"},
	}
}

func TestRenderTableNoColorWhenNotTTY(t *testing.T) {
	var buf bytes.Buffer
	// New with color explicitly disabled (as when piped).
	p := NewPrinter(&buf, false)
	p.PrintTable(sample())
	out := buf.String()
	if !strings.Contains(out, "PORT") || !strings.Contains(out, "PID") || !strings.Contains(out, "PROCESS") || !strings.Contains(out, "ADDRESS") {
		t.Fatalf("missing headers: %q", out)
	}
	if !strings.Contains(out, "3000") || !strings.Contains(out, "node") {
		t.Fatalf("missing data: %q", out)
	}
	// Rows must be sorted by port ascending even if input is not.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var dataLines []string
	for _, l := range lines {
		if strings.HasPrefix(l, "PORT") || strings.HasPrefix(l, "─") || strings.HasPrefix(l, "-") {
			continue
		}
		dataLines = append(dataLines, l)
	}
	if len(dataLines) != 3 {
		t.Fatalf("want 3 data rows, got %d in %q", len(dataLines), out)
	}
	if !strings.Contains(dataLines[0], "3000") {
		t.Fatalf("first data row should be port 3000: %q", dataLines[0])
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("no ANSI escapes when not a TTY: %q", out)
	}
}

func TestRenderTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, false)
	p.PrintTable([]ports.Process{})
	out := buf.String()
	if strings.TrimSpace(out) != "No ports in use." {
		t.Fatalf("empty table output = %q", out)
	}
}

func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, false)
	if err := p.PrintJSON(sample()); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}
	out := buf.String()
	// Must be valid JSON and decodable.
	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 entries, got %d", len(got))
	}
	if got[0]["port"].(float64) != 3000 || got[0]["pid"].(float64) != 18231 {
		t.Fatalf("entry 0 wrong: %v", got[0])
	}
	if got[0]["process"] != "node" || got[0]["protocol"] != "tcp" {
		t.Fatalf("entry 0 fields wrong: %v", got[0])
	}
	for _, banned := range []string{"✓", "─", "PORT"} {
		if strings.Contains(out, banned) {
			t.Fatalf("JSON must not contain %q: %s", banned, out)
		}
	}
}

func TestRenderJSONEmpty(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, false)
	if err := p.PrintJSON([]ports.Process{}); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "[]" {
		t.Fatalf("empty JSON = %q, want []", buf.String())
	}
}

func TestQuietMode(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, false)
	p.SetQuiet(true)
	p.PrintTable(sample())
	if buf.Len() != 0 {
		t.Fatalf("quiet mode must not print table, got %q", buf.String())
	}
	p.Success("Port 3000 is free")
	if buf.Len() != 0 {
		t.Fatalf("quiet mode must not print success text, got %q", buf.String())
	}
	p.Error("something failed")
	if !strings.Contains(buf.String(), "something failed") {
		t.Fatalf("quiet mode must still print errors: %q", buf.String())
	}
}

func TestSuccessCheckmarkOnlyOnTTY(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, true) // TTY
	p.Success("Port 3000 is free")
	if !strings.Contains(buf.String(), "✓") {
		t.Fatalf("TTY success should use ✓: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "\x1b[") {
		t.Fatalf("TTY output should be colored: %q", buf.String())
	}

	buf.Reset()
	p2 := NewPrinter(&buf, false) // piped
	p2.Success("Port 3000 is free")
	if strings.Contains(buf.String(), "✓") || strings.Contains(buf.String(), "\x1b[") {
		t.Fatalf("piped output must be plain: %q", buf.String())
	}
}

func TestTruncationLongProcessName(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, false)
	long := "very-long-process-name-that-exceeds-column"
	ps := []ports.Process{{PID: 1, Name: long, Port: 3000, Protocol: "TCP", Address: "127.0.0.1"}}
	p.PrintTable(ps)
	if !strings.Contains(buf.String(), long) {
		t.Fatalf("long process name should not be truncated away: %q", buf.String())
	}
}

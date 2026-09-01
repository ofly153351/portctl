package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/ofly153351/portctl/internal/ports"
)

// ANSI color codes, used only when writing to a terminal.
const (
	ansiReset  = "\x1b[0m"
	ansiGreen  = "\x1b[32m"
	ansiRed    = "\x1b[31m"
	ansiBold   = "\x1b[1m"
	ansiCyan   = "\x1b[36m"
	ansiYellow = "\x1b[33m"
)

// Printer renders portctl output. It owns all presentation concerns:
// tables, JSON, colors, quiet mode, and the checkmark decorations.
type Printer struct {
	w     io.Writer
	color bool
	quiet bool
}

// NewPrinter returns a Printer writing to w. color enables ANSI escapes
// and the ✓ decoration; disable it when stdout is not a TTY.
func NewPrinter(w io.Writer, color bool) *Printer {
	return &Printer{w: w, color: color}
}

// SetQuiet toggles quiet mode: everything except errors is suppressed.
func (p *Printer) SetQuiet(q bool) { p.quiet = q }

// IsQuiet reports whether quiet mode is on.
func (p *Printer) IsQuiet() bool { return p.quiet }

func (p *Printer) plain(s string) string {
	if !p.color {
		return s
	}
	return s
}

func (p *Printer) green(s string) string {
	if !p.color {
		return s
	}
	return ansiGreen + s + ansiReset
}

func (p *Printer) red(s string) string {
	if !p.color {
		return s
	}
	return ansiRed + s + ansiReset
}

func (p *Printer) bold(s string) string {
	if !p.color {
		return s
	}
	return ansiBold + s + ansiReset
}

func (p *Printer) cyan(s string) string {
	if !p.color {
		return s
	}
	return ansiCyan + s + ansiReset
}

func (p *Printer) yellow(s string) string {
	if !p.color {
		return s
	}
	return ansiYellow + s + ansiReset
}

// Success prints a success line. On a TTY it is decorated with ✓ and
// color; when piped it degrades to plain text with no decorations, per
// the "no color/emoji when piping" rule.
func (p *Printer) Success(msg string) {
	if p.quiet {
		return
	}
	switch {
	case p.color:
		fmt.Fprintf(p.w, "%s✓ %s%s\n", ansiGreen, msg, ansiReset)
	default:
		fmt.Fprintf(p.w, "OK: %s\n", msg)
	}
}

// Error prints an error line to stderr-like output (always shown, even in
// quiet mode, because errors are the contract with scripts).
func (p *Printer) Error(msg string) {
	fmt.Fprintf(p.w, "%s\n", p.red("Error: "+msg))
}

// Info prints a neutral informational line (suppressed by quiet mode).
func (p *Printer) Info(msg string) {
	if p.quiet {
		return
	}
	fmt.Fprintln(p.w, msg)
}

// PrintTable renders the PORT/PID/PROCESS/ADDRESS table with a dashed
// separator, sorted by port. Empty input prints "No ports in use."
func (p *Printer) PrintTable(ps []ports.Process) {
	if p.quiet {
		return
	}
	if len(ps) == 0 {
		fmt.Fprintln(p.w, "No ports in use.")
		return
	}
	sorted := make([]ports.Process, len(ps))
	copy(sorted, ps)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Port != sorted[j].Port {
			return sorted[i].Port < sorted[j].Port
		}
		return sorted[i].PID < sorted[j].PID
	})

	// Column widths: start from the headers, grow to fit the data.
	wPort, wPID, wProc := len("PORT"), len("PID"), len("PROCESS")
	for _, proc := range sorted {
		if n := len(strconv.Itoa(proc.Port)); n > wPort {
			wPort = n
		}
		if n := len(strconv.Itoa(proc.PID)); n > wPID {
			wPID = n
		}
		if n := len(proc.Name); n > wProc {
			wProc = n
		}
	}
	gap := "  "
	row := func(port, pid, proc, addr string) {
		fmt.Fprintf(p.w, "%s%s%s%s%s%s%s%s%s\n",
			padRight(port, wPort), gap,
			padRight(pid, wPID), gap,
			padRight(proc, wProc), gap,
			addr, "", "")
	}
	row("PORT", "PID", "PROCESS", "ADDRESS")
	// Separator spans exactly the header row width.
	headerW := wPort + wPID + wProc + 3*len(gap) + len("ADDRESS")
	sep := strings.Repeat("─", headerW)
	if !p.color {
		sep = strings.Repeat("-", headerW)
	}
	fmt.Fprintln(p.w, sep)
	for _, proc := range sorted {
		row(strconv.Itoa(proc.Port), strconv.Itoa(proc.PID), proc.Name, proc.Address)
	}
}

// padRight pads s with spaces to width n.
func padRight(s string, n int) string {
	for len(s) < n {
		s += " "
	}
	return s
}

// jsonProcess is the wire format for --json output.
type jsonProcess struct {
	Port     int    `json:"port"`
	PID      int    `json:"pid"`
	Process  string `json:"process"`
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
}

// PrintJSON renders processes as a JSON array. Output is pure JSON with
// no decorative text so it can be piped into jq.
func (p *Printer) PrintJSON(ps []ports.Process) error {
	out := make([]jsonProcess, 0, len(ps))
	for _, proc := range ps {
		out = append(out, jsonProcess{
			Port:     proc.Port,
			PID:      proc.PID,
			Process:  proc.Name,
			Protocol: strings.ToLower(proc.Protocol),
			Address:  proc.Address,
		})
	}
	enc := json.NewEncoder(p.w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// PrintTableNoHeader renders just the data rows (PID/PROCESS + port),
// used by info when several processes share one port.
func (p *Printer) PrintTableNoHeader(ps []ports.Process) {
	if p.quiet {
		return
	}
	if len(ps) == 0 {
		return
	}
	tw := tabwriter.NewWriter(p.w, 0, 4, 2, ' ', 0)
	for _, proc := range ps {
		fmt.Fprintf(tw, "%d\t%s\n", proc.PID, proc.Name)
	}
	tw.Flush()
}

// Confirm renders a y/N prompt to w (a terminal in real usage).
func (p *Printer) Confirm(msg string) {
	if p.quiet {
		return
	}
	fmt.Fprintf(p.w, "%s ", msg)
}

// Writer exposes the underlying writer (used by the CLI layer and the
// default prompt).
func (p *Printer) Writer() io.Writer { return p.w }

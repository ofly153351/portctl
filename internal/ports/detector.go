package ports

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// commandRunner runs external commands. It exists so tests can fake lsof/ps.
type commandRunner interface {
	run(name string, args ...string) (stdout string, err error)
}

// execRunner runs real commands via os/exec.
type execRunner struct{}

func (execRunner) run(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	return string(out), err
}

// LsofDetector lists ports via lsof (macOS and Linux).
type LsofDetector struct {
	run commandRunner
}

// NewLsofDetector returns a Detector backed by the system lsof.
func NewLsofDetector() *LsofDetector {
	return &LsofDetector{run: execRunner{}}
}

// newLsofDetectorWithRunner is used by tests to inject fake command output.
func newLsofDetectorWithRunner(r commandRunner) *LsofDetector {
	return &LsofDetector{run: r}
}

// lsofArgs are the arguments passed to lsof. -F selects field output with
// the fields we parse; -n and -P disable costly DNS/port-name lookups.
var lsofArgs = []string{"-nP", "-i", "-FpcLPftnT"}

// ListPorts returns every process holding a TCP or UDP endpoint, deduped
// and sorted by port.
func (d *LsofDetector) ListPorts() ([]Process, error) {
	out, err := d.run.run("lsof", lsofArgs...)
	if err != nil {
		return nil, wrapLsofError(err)
	}
	ps := parseLsofFieldOutput(out)
	ps = filterEndpoints(ps)
	ps = Unique(ps)
	SortByPort(ps)
	return ps, nil
}

// FindByPort returns the processes using the given port, or an error
// wrapping ErrPortNotInUse when nothing holds it.
func (d *LsofDetector) FindByPort(port int) ([]Process, error) {
	all, err := d.ListPorts()
	if err != nil {
		return nil, err
	}
	return filterByPort(all, port)
}

// filterEndpoints keeps listening TCP sockets, TCP sockets without a state
// (Linux lsof sometimes omits state), and unconnected UDP sockets. It drops
// outbound/established connections that would only add noise to ls output.
func filterEndpoints(ps []Process) []Process {
	out := make([]Process, 0, len(ps))
	for _, p := range ps {
		switch p.Protocol {
		case ProtocolTCP:
			if p.State == "LISTEN" || p.State == "" {
				out = append(out, p)
			}
		case ProtocolUDP:
			if !strings.Contains(p.RawName(), "->") {
				out = append(out, p)
			}
		}
	}
	return out
}

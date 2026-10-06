package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/ofly153351/portctl/internal/dnsflush"
	"github.com/ofly153351/portctl/internal/ui"
)

type dnsFlusher interface {
	Flush(goos string, run dnsflush.Runner, report func(string)) error
}

type systemDNSFlusher struct{}

func (systemDNSFlusher) Flush(goos string, run dnsflush.Runner, report func(string)) error {
	return dnsflush.Flush(goos, run, report)
}

var activeDNSFlusher dnsFlusher = systemDNSFlusher{}
var dnsRun = func(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			return fmt.Errorf("%w: %s", err, output)
		}
		return err
	}
	return nil
}

// runDNS handles `portctl dns flush`.
func runDNS(args []string, printer *ui.Printer) int {
	if len(args) != 1 || args[0] != "flush" {
		printer.Error("usage: portctl dns flush")
		printer.Info("Run:")
		printer.Info("  portctl --help")
		return ExitUsage
	}

	if err := activeDNSFlusher.Flush(runtime.GOOS, dnsRun, printer.Success); err != nil {
		printer.Error(err.Error())
		return ExitGeneral
	}
	return ExitSuccess
}

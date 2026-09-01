package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ofly153351/portctl/internal/app"
	"github.com/ofly153351/portctl/internal/ports"
	"github.com/ofly153351/portctl/internal/system"
	"github.com/ofly153351/portctl/internal/ui"
)

// Exit codes (TASK.md §19).
const (
	ExitSuccess    = 0
	ExitGeneral    = 1
	ExitUsage      = 2
	ExitPermission = 3
)

// Version is set at build time via -ldflags "-X main.version=...".
var Version = "dev"

const helpText = `portctl - Port & Process Manager

Usage:
  portctl <command> [arguments]

Commands:
  ls       List ports and processes
  info     Show detailed information about a port
  kill     Kill process using a port
  free     Find and kill process using a port

Options:
  --json       Machine-readable JSON output
  --force, -f  Skip confirmation and use SIGKILL
  --quiet, -q  Suppress non-error output (scripts)
  --help, -h   Show help
  --version    Show version

Aliases:
  l → ls, i → info, k → kill, f → free

Examples:
  portctl ls
  portctl info 3000
  portctl kill 3000
  portctl kill 3000 8080
  portctl free 3000`

// Execute runs the CLI with the given arguments and returns the process
// exit code. The App and Printer are injected for testability.
func Execute(args []string, application app.App, printer *ui.Printer) int {
	lastArgs = args
	if len(args) == 0 {
		printer.Info(helpText)
		return ExitSuccess
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "--help", "-h", "help":
		printer.Info(helpText)
		return ExitSuccess
	case "--version", "-v", "version":
		printer.Info(fmt.Sprintf("portctl version %s", Version))
		return ExitSuccess
	case "ls", "l":
		return runLs(rest, application, printer)
	case "info", "i":
		return runInfo(rest, application, printer)
	case "kill", "k":
		return runKill(rest, application, printer)
	case "free", "f":
		return runFree(rest, application, printer)
	default:
		printer.Error(fmt.Sprintf("unknown command %q", cmd))
		printer.Info("")
		printer.Info("Run:")
		printer.Info("  portctl --help")
		return ExitUsage
	}
}

// flags holds the parsed global options.
type flags struct {
	force bool
	quiet bool
	json  bool
	args  []string
}

// parseFlags splits flag arguments from positional arguments. Unknown
// flags are reported via the returned error.
func parseFlags(rest []string) (flags, error) {
	var f flags
	for i := 0; i < len(rest); i++ {
		arg := rest[i]
		switch arg {
		case "--force", "-f":
			f.force = true
		case "--quiet", "-q":
			f.quiet = true
		case "--json":
			f.json = true
		case "--":
			f.args = append(f.args, rest[i+1:]...)
			return f, nil
		default:
			if strings.HasPrefix(arg, "-") && arg != "-" {
				return f, fmt.Errorf("unknown flag %q", arg)
			}
			f.args = append(f.args, arg)
		}
	}
	return f, nil
}

// runLs implements `portctl ls [--json]`.
func runLs(rest []string, application app.App, printer *ui.Printer) int {
	f, err := parseFlags(rest)
	if err != nil {
		return usageErr(printer, err)
	}
	if len(f.args) != 0 {
		return usageErr(printer, errors.New("ls takes no arguments"))
	}
	if err := application.ListPorts(app.Opts{JSON: f.json, Quiet: f.quiet}); err != nil {
		return appErr(printer, err)
	}
	return ExitSuccess
}

// runInfo implements `portctl info <port>`.
func runInfo(rest []string, application app.App, printer *ui.Printer) int {
	f, err := parseFlags(rest)
	if err != nil {
		return usageErr(printer, err)
	}
	if len(f.args) != 1 {
		return usageErr(printer, errors.New("usage: portctl info <port>"))
	}
	port, err := ports.ValidatePort(f.args[0])
	if err != nil {
		return usageErr(printer, err)
	}
	if err := application.Info(port, app.Opts{JSON: f.json, Quiet: f.quiet}); err != nil {
		return appErr(printer, err)
	}
	return ExitSuccess
}

// runKill implements `portctl kill <ports...> [--force] [--quiet]`.
func runKill(rest []string, application app.App, printer *ui.Printer) int {
	f, err := parseFlags(rest)
	if err != nil {
		return usageErr(printer, err)
	}
	if len(f.args) == 0 {
		return usageErr(printer, errors.New("usage: portctl kill <port> [port...]"))
	}
	portList := make([]int, 0, len(f.args))
	for _, raw := range f.args {
		port, err := ports.ValidatePort(raw)
		if err != nil {
			return usageErr(printer, err)
		}
		portList = append(portList, port)
	}
	if err := application.KillPorts(portList, app.Opts{
		Force: f.force, Quiet: f.quiet, JSON: f.json,
	}); err != nil {
		return appErr(printer, err)
	}
	return ExitSuccess
}

// runFree implements `portctl free <port>`.
func runFree(rest []string, application app.App, printer *ui.Printer) int {
	f, err := parseFlags(rest)
	if err != nil {
		return usageErr(printer, err)
	}
	if len(f.args) != 1 {
		return usageErr(printer, errors.New("usage: portctl free <port>"))
	}
	port, err := ports.ValidatePort(f.args[0])
	if err != nil {
		return usageErr(printer, err)
	}
	if err := application.FreePort(port, app.Opts{
		Force: f.force, Quiet: f.quiet, JSON: f.json,
	}); err != nil {
		return appErr(printer, err)
	}
	return ExitSuccess
}

// usageErr prints a usage error and returns the usage exit code.
func usageErr(printer *ui.Printer, err error) int {
	printer.Error(err.Error())
	printer.Info("")
	printer.Info("Run:")
	printer.Info("  portctl --help")
	return ExitUsage
}

// lastArgs remembers the invocation for the sudo hint in permission
// errors. Execute stores the original args here.
var lastArgs []string

// appErr maps application errors to exit codes (TASK.md §19).
func appErr(printer *ui.Printer, err error) int {
	switch {
	case errors.Is(err, ports.ErrPortNotInUse):
		// A successful inspection with no process is not an error.
		return ExitSuccess
	case errors.Is(err, system.ErrPermissionDenied):
		printer.Error(err.Error())
		printer.Info("")
		printer.Info("Try running:")
		printer.Info("  sudo portctl " + strings.Join(lastArgs, " "))
		return ExitPermission
	default:
		printer.Error(err.Error())
		return ExitGeneral
	}
}

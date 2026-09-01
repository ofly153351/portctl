package app

import (
	"fmt"
	"io"
	"strings"

	"github.com/ofly153351/portctl/internal/ports"
	"github.com/ofly153351/portctl/internal/system"
	"github.com/ofly153351/portctl/internal/ui"
)

// Opts controls how commands behave.
type Opts struct {
	Force bool // skip confirmation and use SIGKILL
	Quiet bool // script mode: suppress non-error output
	JSON  bool // machine-readable output
}

// Prompter asks the user to confirm dangerous actions.
type Prompter interface {
	Confirm(msg string) bool
}

// App is the business-logic layer between the CLI and the OS services.
type App interface {
	ListPorts(opts Opts) error
	Info(port int, opts Opts) error
	KillPorts(portList []int, opts Opts) error
	FreePort(port int, opts Opts) error
}

type appImpl struct {
	svc    Services
	prompt Prompter
	out    io.Writer
	finds  int
}

// Services bundles the OS-facing dependencies of the app layer.
type Services struct {
	Finder  ports.Finder
	Manager system.Manager
	Printer *ui.Printer
}

// New builds the application service with its OS dependencies.
func New(finder ports.Finder, mgr system.Manager, printer *ui.Printer) App {
	return &appImpl{
		svc:    Services{Finder: finder, Manager: mgr, Printer: printer},
		prompt: defaultPrompt{printer: printer},
		out:    printer.Writer(),
	}
}

// SetPrompter overrides the confirmation source (used by tests).
func (a *appImpl) SetPrompter(p Prompter) { a.prompt = p }

// defaultPrompt reads the y/N answer from stdin. Prompt text is printed
// by the app layer via Printer.Confirm; this type only reads the reply.
type defaultPrompt struct{ printer *ui.Printer }

func (d defaultPrompt) Confirm(msg string) bool {
	var answer string
	if _, err := fmt.Scanln(&answer); err != nil {
		return false
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}

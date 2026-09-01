package main

import (
	"os"

	"github.com/ofly153351/portctl/cmd"
	"github.com/ofly153351/portctl/internal/app"
	"github.com/ofly153351/portctl/internal/ports"
	"github.com/ofly153351/portctl/internal/system"
	"github.com/ofly153351/portctl/internal/ui"
)

// version is injected at build time:
// go build -ldflags "-X main.version=0.1.0"
var version = "0.1.0"

func main() {
	cmd.Version = version

	color := isTTY(os.Stdout) && !isPiped()
	printer := ui.NewPrinter(os.Stdout, color)

	detector := ports.NewLsofDetector()
	manager := system.NewManager()
	application := app.New(detector, manager, printer)

	code := cmd.Execute(os.Args[1:], application, printer)
	os.Exit(code)
}

// isTTY reports whether f is a terminal.
func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// isPiped reports whether output is being redirected (NO_COLOR or a pipe
// disables decoration).
func isPiped() bool {
	return os.Getenv("NO_COLOR") != ""
}

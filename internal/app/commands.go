package app

import (
	"errors"
	"fmt"

	"github.com/ofly153351/portctl/internal/ports"
	"github.com/ofly153351/portctl/internal/system"
)

// ListPorts implements `portctl ls`.
func (a *appImpl) ListPorts(opts Opts) error {
	ps, err := a.svc.Finder.ListPorts()
	if err != nil {
		return err
	}
	if opts.JSON {
		return a.svc.Printer.PrintJSON(ps)
	}
	a.svc.Printer.PrintTable(ps)
	return nil
}

// Info implements `portctl info <port>`.
func (a *appImpl) Info(port int, opts Opts) error {
	a.svc.Printer.SetQuiet(opts.Quiet)
	defer a.svc.Printer.SetQuiet(false)
	a.finds++
	found, err := a.svc.Finder.FindByPort(port)
	if err != nil {
		if ports.IsPortNotInUse(err) {
			a.svc.Printer.Info(fmt.Sprintf("Port %d is not in use.", port))
			return nil
		}
		return err
	}
	if opts.JSON {
		return a.svc.Printer.PrintJSON(found)
	}
	if len(found) == 1 {
		p := found[0]
		a.svc.Printer.Info(fmt.Sprintf("Port:       %d", p.Port))
		a.svc.Printer.Info(fmt.Sprintf("Protocol:   %s", p.Protocol))
		a.svc.Printer.Info(fmt.Sprintf("Address:    %s", p.Address))
		a.svc.Printer.Info(fmt.Sprintf("PID:        %d", p.PID))
		a.svc.Printer.Info(fmt.Sprintf("Process:    %s", p.Name))
		if p.Command != "" {
			a.svc.Printer.Info(fmt.Sprintf("Command:    %s", p.Command))
		}
		if p.User != "" {
			a.svc.Printer.Info(fmt.Sprintf("User:       %s", p.User))
		}
		return nil
	}
	// Multiple processes share the port (TASK.md §5).
	a.svc.Printer.Info(fmt.Sprintf("Port: %d", port))
	a.svc.Printer.Info("")
	for _, p := range found {
		a.svc.Printer.Info(fmt.Sprintf("%-8d %s", p.PID, p.Name))
	}
	return nil
}

// KillPorts implements `portctl kill <ports...>`.
func (a *appImpl) KillPorts(portList []int, opts Opts) error {
	a.svc.Printer.SetQuiet(opts.Quiet)
	defer a.svc.Printer.SetQuiet(false)

	targets := make([]ports.Process, 0, len(portList))
	for _, port := range portList {
		a.finds++
		found, err := a.svc.Finder.FindByPort(port)
		if err != nil {
			if ports.IsPortNotInUse(err) {
				a.svc.Printer.Info(fmt.Sprintf("Port %d is not in use.", port))
				continue
			}
			return err
		}
		if !opts.JSON {
			a.reportFound(port, found)
		}
		targets = append(targets, found...)
	}
	if len(targets) == 0 {
		return nil
	}
	if !opts.Force {
		if len(portList) == 1 && len(targets) == 1 {
			a.svc.Printer.Confirm(fmt.Sprintf("Kill process %d? [y/N]", targets[0].PID))
		} else {
			a.svc.Printer.Confirm("Continue? [y/N]")
		}
		if !a.prompt.Confirm("") {
			a.svc.Printer.Info("Aborted.")
			return nil
		}
	}
	return a.killTargets(targets, opts, false)
}

// reportFound prints the pre-kill summary blocks (TASK.md §6/§7).
func (a *appImpl) reportFound(port int, found []ports.Process) {
	if len(found) == 1 {
		p := found[0]
		a.svc.Printer.Info("Found process:")
		a.svc.Printer.Info("")
		a.svc.Printer.Info("PID      PROCESS      PORT")
		a.svc.Printer.Info(fmt.Sprintf("%-8d %-12s %d", p.PID, p.Name, p.Port))
		a.svc.Printer.Info("")
		return
	}
	a.svc.Printer.Info(fmt.Sprintf("Port %d", port))
	for _, p := range found {
		a.svc.Printer.Info(fmt.Sprintf("  PID %d %s", p.PID, p.Name))
	}
	a.svc.Printer.Info("")
}

// killTargets signals every target and verifies the ports are free.
// freeStyle switches the final wording to the `free` phrasing (§8).
func (a *appImpl) killTargets(targets []ports.Process, opts Opts, freeStyle bool) error {
	var firstErr error
	for _, tgt := range targets {
		if system.IsProtected(tgt.PID) {
			err := system.ValidatePID(tgt.PID)
			a.svc.Printer.Error(err.Error())
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		var err error
		if opts.Force {
			err = a.svc.Manager.Kill(tgt.PID)
		} else {
			err = a.svc.Manager.Terminate(tgt.PID)
		}
		if err != nil {
			if errors.Is(err, system.ErrProcessGone) {
				a.svc.Printer.Info(fmt.Sprintf("Process %d no longer exists.", tgt.PID))
				continue
			}
			a.svc.Printer.Error(err.Error())
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		switch {
		case freeStyle:
			a.svc.Printer.Success("Process killed")
		case opts.Force:
			a.svc.Printer.Success(fmt.Sprintf("Killed %s (PID %d)", tgt.Name, tgt.PID))
		default:
			a.svc.Printer.Success(fmt.Sprintf("Process %d killed", tgt.PID))
		}
	}
	// Verify each affected port is now free.
	for _, tgt := range targets {
		a.finds++
		_, err := a.svc.Finder.FindByPort(tgt.Port)
		if err == nil {
			a.svc.Printer.Error(fmt.Sprintf("Port %d is still in use.", tgt.Port))
			if firstErr == nil {
				firstErr = fmt.Errorf("port %d still in use", tgt.Port)
			}
			continue
		}
		if !errors.Is(err, ports.ErrPortNotInUse) {
			// Detector failure during verification is not fatal.
			a.svc.Printer.Error(fmt.Sprintf("failed to inspect port %d", tgt.Port))
			continue
		}
		if freeStyle {
			a.svc.Printer.Success(fmt.Sprintf("Port %d is free", tgt.Port))
		} else {
			a.svc.Printer.Success(fmt.Sprintf("Port %d is now free", tgt.Port))
		}
	}
	return firstErr
}

// FreePort implements `portctl free <port>`: find → show → confirm →
// kill → verify (TASK.md §8).
func (a *appImpl) FreePort(port int, opts Opts) error {
	a.svc.Printer.SetQuiet(opts.Quiet)
	defer a.svc.Printer.SetQuiet(false)

	a.finds++
	found, err := a.svc.Finder.FindByPort(port)
	if err != nil {
		if ports.IsPortNotInUse(err) {
			a.svc.Printer.Info(fmt.Sprintf("Port %d is not in use.", port))
			return nil
		}
		return err
	}
	a.svc.Printer.Info(fmt.Sprintf("Port %d is used by:", port))
	a.svc.Printer.Info("")
	a.svc.Printer.Info("PID      PROCESS")
	for _, p := range found {
		a.svc.Printer.Info(fmt.Sprintf("%-8d %s", p.PID, p.Name))
	}
	a.svc.Printer.Info("")
	if !opts.Force {
		a.svc.Printer.Confirm(fmt.Sprintf("Kill process %d? [y/N]", found[0].PID))
		if !a.prompt.Confirm("") {
			a.svc.Printer.Info("Aborted.")
			return nil
		}
	}
	return a.killTargets(found, opts, true)
}

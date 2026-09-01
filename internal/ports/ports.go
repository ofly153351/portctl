package ports

import "errors"

// ErrPortNotInUse is returned by FindByPort when the port is free.
var ErrPortNotInUse = errors.New("port not in use")

// Process describes a process holding one network endpoint on a port.
type Process struct {
	PID      int
	Name     string
	Command  string
	User     string
	Port     int
	Protocol string
	Address  string
	// State is the socket state, e.g. LISTEN or ESTABLISHED. It is empty
	// when the protocol has no state (UDP) or the source omitted it.
	State string
	// rawName keeps the original endpoint string (the lsof n-field) so
	// filters can detect connected UDP sockets ("local->remote").
	rawName string
}

// RawName returns the original endpoint string, e.g. "127.0.0.1:52309" or
// "127.0.0.1:53->8.8.8.8:53". It is empty when unknown.
func (p Process) RawName() string { return p.rawName }

// Protocol names as used by portctl.
const (
	ProtocolTCP = "TCP"
	ProtocolUDP = "UDP"
)

// Detector reports processes bound to network ports.
type Detector interface {
	ListPorts() ([]Process, error)
	// FindByPort returns the processes using the given port, or an error
	// wrapping ErrPortNotInUse when nothing holds it.
	FindByPort(port int) ([]Process, error)
}

// Finder is the minimal read-only capability needed by most commands.
type Finder interface {
	ListPorts() ([]Process, error)
	FindByPort(port int) ([]Process, error)
}

package ports

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// minPort and maxPort bound the valid TCP/UDP port range.
const (
	minPort = 1
	maxPort = 65535
)

// ValidatePort returns an error when raw is not a valid port string.
func ValidatePort(raw string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid port %q", raw)
	}
	if err := ValidatePortNumber(n); err != nil {
		return 0, err
	}
	return n, nil
}

// ValidatePortNumber returns an error when n is outside 1..65535.
func ValidatePortNumber(n int) error {
	if n < minPort || n > maxPort {
		return fmt.Errorf("port must be between %d and %d", minPort, maxPort)
	}
	return nil
}

// IsPortNotInUse reports whether err is the "port not in use" condition.
func IsPortNotInUse(err error) bool {
	return errors.Is(err, ErrPortNotInUse)
}

// filterByPort returns processes that hold the given port. If none do,
// it returns ErrPortNotInUse.
func filterByPort(all []Process, port int) ([]Process, error) {
	var out []Process
	for _, p := range all {
		if p.Port == port {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("port %d: %w", port, ErrPortNotInUse)
	}
	return out, nil
}

package ports

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrLsofUnavailable is returned when the lsof binary cannot be executed.
var ErrLsofUnavailable = errors.New("lsof unavailable")

// wrapLsofError converts a raw exec error into a readable portctl error.
//
// lsof exits 1 both when nothing matched and when it failed, so we treat a
// clean stderr as "no results" and anything else as a genuine failure.
func wrapLsofError(err error) error {
	if err == nil {
		return nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if len(ee.Stderr) == 0 {
			// lsof exit 1 with no stderr: no matching records.
			return nil
		}
		return fmt.Errorf("%w: %s", ErrLsofUnavailable, firstLine(string(ee.Stderr)))
	}
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("%w: lsof not found in PATH", ErrLsofUnavailable)
	}
	return fmt.Errorf("%w: %v", ErrLsofUnavailable, err)
}

// IsLsofUnavailable reports whether err is an lsof availability failure.
func IsLsofUnavailable(err error) bool {
	return errors.Is(err, ErrLsofUnavailable)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return s
}

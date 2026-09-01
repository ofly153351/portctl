package ports

import (
	"strconv"
	"strings"
)

// parseLsofFieldOutput converts `lsof -F` field output into Process values.
//
// Expected invocation shape: lsof -nP -i -FpcLPftnT. Fields may arrive in
// any order; a process record spans from its p line to the next p line, and
// each f line starts a new socket block inside it.
func parseLsofFieldOutput(out string) []Process {
	var result []Process
	var cur *Process
	var proto, name, state string

	flushSocket := func() {
		if cur != nil && proto != "" && name != "" {
			if addr, port, ok := parseLsofName(name); ok {
				p := *cur
				p.Protocol = proto
				p.Address = addr
				p.Port = port
				p.State = state
				p.rawName = name
				result = append(result, p)
			}
		}
		proto, name, state = "", "", ""
	}

	for _, line := range strings.Split(out, "\n") {
		if len(line) < 2 {
			continue
		}
		tag, val := line[0], line[1:]
		switch tag {
		case 'p':
			flushSocket()
			pid, _ := strconv.Atoi(val)
			cur = &Process{PID: pid}
		case 'c':
			if cur != nil {
				cur.Name = val
			}
		case 'L':
			if cur != nil {
				cur.User = val
			}
		case 'f':
			flushSocket()
		case 'P':
			proto = val
		case 'n':
			name = val
		case 'T':
			if strings.HasPrefix(val, "ST=") {
				state = strings.TrimPrefix(val, "ST=")
			}
		}
	}
	flushSocket()
	return result
}

// parseLsofName splits an lsof network name such as "127.0.0.1:52309",
// "*:3000", "[::1]:8081", or "[addr]:port->[remote]:port" into host and
// port. ok is false when the name is not a socket endpoint with a port.
func parseLsofName(name string) (host string, port int, ok bool) {
	local := name
	if i := strings.Index(name, "->"); i >= 0 {
		local = name[:i]
	}
	local = strings.TrimSpace(local)
	if local == "" {
		return "", 0, false
	}
	if local == "*" {
		return "0.0.0.0", 0, false
	}
	// IPv6 bracketed form.
	if strings.HasPrefix(local, "[") {
		end := strings.Index(local, "]")
		if end < 0 {
			return "", 0, false
		}
		h := local[1:end]
		rest := local[end+1:]
		if !strings.HasPrefix(rest, ":") {
			return "", 0, false
		}
		p, ok := atoiPort(rest[1:])
		if !ok {
			return "", 0, false
		}
		return h, p, ok
	}
	// IPv4 host:port form.
	i := strings.LastIndex(local, ":")
	if i < 0 {
		return "", 0, false
	}
	h := local[:i]
	if h == "*" {
		h = "0.0.0.0"
	}
	p, ok := atoiPort(local[i+1:])
	if !ok {
		return "", 0, false
	}
	return h, p, ok
}

func atoiPort(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 65535 {
		return 0, false
	}
	return n, true
}

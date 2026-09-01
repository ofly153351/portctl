package ports

import (
	"fmt"
	"sort"
)

// SortByPort orders processes by port ascending, then PID, then protocol.
func SortByPort(ps []Process) {
	sort.SliceStable(ps, func(i, j int) bool {
		if ps[i].Port != ps[j].Port {
			return ps[i].Port < ps[j].Port
		}
		if ps[i].PID != ps[j].PID {
			return ps[i].PID < ps[j].PID
		}
		return ps[i].Protocol < ps[j].Protocol
	})
}

// Unique returns processes deduplicated by PID+Port+Protocol+Address.
func Unique(ps []Process) []Process {
	seen := make(map[string]bool)
	out := make([]Process, 0, len(ps))
	for _, p := range ps {
		key := fmt.Sprintf("%d|%d|%s|%s", p.PID, p.Port, p.Protocol, p.Address)
		if !seen[key] {
			seen[key] = true
			out = append(out, p)
		}
	}
	return out
}

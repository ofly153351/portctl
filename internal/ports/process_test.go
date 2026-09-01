package ports

import (
	"testing"
)

// Fixture A: real lsof -F output collected on macOS (probe pid 90039 held
// one UDP and one TCP LISTEN socket; Codex held established connections).
const fixtureMacOSField = `p90039
cPython
Lobx
f3
tIPv4
PUDP
n127.0.0.1:61367
f4
tIPv4
PTCP
n127.0.0.1:52309
TST=LISTEN
TQR=0
TQS=0
p16309
cCodex (Service)
Lobx
f27
tIPv6
PTCP
n[2403:6200:8827:231c:2575:d03e:14f4:38d3]:53953->[2404:6800:4003:c03::bc]:5228
TST=ESTABLISHED
TQR=0
TQS=0
p19344
crapportd
Lobx
f13
tIPv4
PTCP
n*:53977
TST=LISTEN
p10071
cPortfolio.Api
Lobx
f301
tIPv4
PTCP
n127.0.0.1:8081
TST=LISTEN
f302
tIPv6
PTCP
n[::1]:8081
TST=LISTEN
`

func TestParseLsofFieldOutput(t *testing.T) {
	got := parseLsofFieldOutput(fixtureMacOSField)
	type row struct {
		pid, port         int
		name, user, proto string
		addr, state, raw  string
	}
	rows := []row{
		{90039, 61367, "Python", "obx", "UDP", "127.0.0.1", "", "127.0.0.1:61367"},
		{90039, 52309, "Python", "obx", "TCP", "127.0.0.1", "LISTEN", "127.0.0.1:52309"},
		{16309, 53953, "Codex (Service)", "obx", "TCP", "2403:6200:8827:231c:2575:d03e:14f4:38d3", "ESTABLISHED", "[2403:6200:8827:231c:2575:d03e:14f4:38d3]:53953->[2404:6800:4003:c03::bc]:5228"},
		{19344, 53977, "rapportd", "obx", "TCP", "0.0.0.0", "LISTEN", "*:53977"},
		{10071, 8081, "Portfolio.Api", "obx", "TCP", "127.0.0.1", "LISTEN", "127.0.0.1:8081"},
		{10071, 8081, "Portfolio.Api", "obx", "TCP", "::1", "LISTEN", "[::1]:8081"},
	}
	if len(got) != len(rows) {
		t.Fatalf("len = %d, want %d", len(got), len(rows))
	}
	for i, want := range rows {
		p := got[i]
		if p.PID != want.pid || p.Port != want.port || p.Name != want.name ||
			p.User != want.user || p.Protocol != want.proto ||
			p.Address != want.addr || p.State != want.state || p.RawName() != want.raw {
			t.Fatalf("entry %d mismatch:\ngot  %+v (raw %q)\nwant %+v", i, p, p.RawName(), want)
		}
	}
}

func TestParseLsofFieldOutputUnusualOrder(t *testing.T) {
	// lsof can emit some fields in a different order; only the grouping
	// (p..f..n) matters, not the order within a socket block.
	in := `p42
csh
Lroot
tIPv6
PTCP
n[::1]:9
TST=LISTEN
f7
tIPv6
PTCP
n[::1]:9
TST=LISTEN
`
	got := parseLsofFieldOutput(in)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (one entry per fd/socket block)", len(got))
	}
	for _, p := range got {
		if p.PID != 42 || p.Name != "sh" || p.User != "root" || p.Port != 9 || p.Protocol != "TCP" {
			t.Fatalf("bad parse: %#v", p)
		}
	}
}

func TestParseLsofName(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		wantHost string
		wantPort int
		wantOK   bool
	}{
		{"ipv4", "127.0.0.1:52309", "127.0.0.1", 52309, true},
		{"wildcard v4", "*:3000", "0.0.0.0", 3000, true},
		{"ipv6 bracket", "[::1]:8081", "::1", 8081, true},
		{"ipv6 wildcard", "[::*:3000]", "", 0, false},
		{"established v4", "1.2.3.4:55555->5.6.7.8:443", "1.2.3.4", 55555, true},
		{"established v6", "[2403:aa::1]:53953->[2404:6800::bc]:5228", "2403:aa::1", 53953, true},
		{"no port", "127.0.0.1", "", 0, false},
		{"bad port", "127.0.0.1:abc", "", 0, false},
		{"port 0", "127.0.0.1:0", "", 0, false},
		{"empty", "", "", 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host, port, ok := parseLsofName(tc.in)
			if host != tc.wantHost || port != tc.wantPort || ok != tc.wantOK {
				t.Fatalf("parseLsofName(%q) = (%q,%d,%v), want (%q,%d,%v)",
					tc.in, host, port, ok, tc.wantHost, tc.wantPort, tc.wantOK)
			}
		})
	}
}

func TestSortByPort(t *testing.T) {
	ps := []Process{
		{PID: 2, Port: 8080},
		{PID: 3, Port: 3000},
		{PID: 1, Port: 3000},
	}
	SortByPort(ps)
	// port ascending; within same port, PID ascending.
	if ps[0].PID != 1 || ps[0].Port != 3000 {
		t.Fatalf("first = PID %d port %d, want PID 1 port 3000", ps[0].PID, ps[0].Port)
	}
	if ps[1].PID != 3 || ps[1].Port != 3000 {
		t.Fatalf("second = PID %d port %d, want PID 3 port 3000", ps[1].PID, ps[1].Port)
	}
	if ps[2].PID != 2 || ps[2].Port != 8080 {
		t.Fatalf("third = PID %d port %d, want PID 2 port 8080", ps[2].PID, ps[2].Port)
	}
}

func TestUnique(t *testing.T) {
	ps := []Process{
		{PID: 1, Port: 3000, Protocol: "TCP", Address: "127.0.0.1"},
		{PID: 1, Port: 3000, Protocol: "TCP", Address: "127.0.0.1"},
		{PID: 1, Port: 3000, Protocol: "UDP", Address: "127.0.0.1"},
	}
	got := Unique(ps)
	if len(got) != 2 {
		t.Fatalf("Unique len = %d, want 2", len(got))
	}
}

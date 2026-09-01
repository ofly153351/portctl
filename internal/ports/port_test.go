package ports

import (
	"errors"
	"testing"
)

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    int
		wantErr string
	}{
		{"simple", "3000", 3000, ""},
		{"trims spaces", " 8080 ", 8080, ""},
		{"lower bound", "1", 1, ""},
		{"upper bound", "65535", 65535, ""},
		{"non-numeric", "abc", 0, `invalid port "abc"`},
		{"empty", "", 0, `invalid port ""`},
		{"negative", "-1", 0, "port must be between 1 and 65535"},
		{"zero", "0", 0, "port must be between 1 and 65535"},
		{"too big", "65536", 0, "port must be between 1 and 65535"},
		{"float", "3.5", 0, `invalid port "3.5"`},
		{"plus sign", "+3000", 3000, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidatePort(tc.raw)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidatePort(%q) unexpected error: %v", tc.raw, err)
				}
				if got != tc.want {
					t.Fatalf("ValidatePort(%q) = %d, want %d", tc.raw, got, tc.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidatePort(%q) expected error %q, got nil (port %d)", tc.raw, tc.wantErr, got)
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("ValidatePort(%q) error = %q, want %q", tc.raw, err.Error(), tc.wantErr)
			}
		})
	}
}

func TestValidatePortNumber(t *testing.T) {
	if err := ValidatePortNumber(80); err != nil {
		t.Fatalf("ValidatePortNumber(80) = %v, want nil", err)
	}
	for _, n := range []int{0, -1, 65536, 100000} {
		if err := ValidatePortNumber(n); err == nil {
			t.Fatalf("ValidatePortNumber(%d) = nil, want error", n)
		}
	}
}

func TestFilterByPort(t *testing.T) {
	all := []Process{
		{PID: 5, Name: "a", Port: 8080, Protocol: "TCP"},
		{PID: 6, Name: "b", Port: 3000, Protocol: "TCP"},
		{PID: 7, Name: "c", Port: 3000, Protocol: "UDP"},
	}
	got, err := filterByPort(all, 3000)
	if err != nil {
		t.Fatalf("filterByPort: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	for _, p := range got {
		if p.Port != 3000 {
			t.Fatalf("got port %d in result", p.Port)
		}
	}
}

func TestFilterByPortNotInUse(t *testing.T) {
	all := []Process{{PID: 5, Name: "a", Port: 8080}}
	_, err := filterByPort(all, 3000)
	if !errors.Is(err, ErrPortNotInUse) {
		t.Fatalf("err = %v, want ErrPortNotInUse", err)
	}
}

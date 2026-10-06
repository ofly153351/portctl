# portctl

> A simple CLI for inspecting and managing processes by network port.

`portctl` is a simple CLI tool for inspecting and managing processes using network ports.
Stop memorizing `lsof` flags — just ask for the port.

```bash
portctl ls
portctl info 3000
portctl kill 3000
```

## Features

- ✓ List ports and their processes (`ls`)
- ✓ Inspect processes on a port (`info`)
- ✓ Kill processes by port (`kill`)
- ✓ Flush macOS DNS cache (`dns flush`)
- ✓ Multiple ports in one command
- ✓ Force mode (`--force` / `-f`)
- ✓ JSON output (`--json`)
- ✓ Quiet mode for scripts (`--quiet` / `-q`)
- ✓ Graceful SIGTERM by default, SIGKILL only with `--force`
- ✓ Protected processes (PID 1 and friends are never killed)
- ✓ Short aliases (`l`, `i`, `k`, `f`)
- ✓ macOS support
- ✓ Linux support

## Installation

### Install a published release (macOS / Linux)

Once a GitHub Release is published with platform binaries, install the latest version with:

```bash
curl -fsSL https://raw.githubusercontent.com/ofly153351/portctl/main/install.sh | sh
```

The installer detects OS/architecture, downloads the matching release binary, and installs it to `/usr/local/bin` (requests `sudo` if needed). It does not modify shell files, DNS settings, or network configuration. To install a specific tagged release:

```bash
curl -fsSL https://raw.githubusercontent.com/ofly153351/portctl/main/install.sh | PORTCTL_VERSION=0.2.0 sh
```

For a fork, set `PORTCTL_REPO=owner/repo`. To use a writable custom install directory, set `PORTCTL_INSTALL_DIR="$HOME/.local/bin"` and ensure it is on your `PATH`.

Note: the installer downloads published GitHub Release assets; pushing source changes alone does not publish an installable binary. A release needs assets named `portctl_darwin_arm64`, `portctl_darwin_amd64`, `portctl_linux_arm64`, and `portctl_linux_amd64` for all listed platforms.

### Build from source

```bash
git clone https://github.com/ofly153351/portctl.git
cd portctl

go build -o portctl .
sudo mv portctl /usr/local/bin/
```

Or use make:

```bash
make build          # produces bin/portctl
make install        # installs to /usr/local/bin (sudo)
```

Verify:

```bash
portctl --version
```

### Shell aliases (optional)

Want it even shorter? Add these to your `.zshrc` / `.bashrc`:

```bash
alias ports="portctl ls"
alias kp="portctl kill"
```

Then your workflow becomes:

```bash
ports
kp 3000
```

(portctl never touches your shell config files itself.)

## Usage

```text
portctl - Port & Process Manager

Usage:
  portctl <command> [arguments]

Commands:
  ls       List ports and processes
  info     Show detailed information about a port
  kill     Kill process using a port
  free     Find and kill process using a port
  dns      Manage DNS cache

Options:
  --json       Machine-readable JSON output
  --force, -f  Skip confirmation and use SIGKILL
  --quiet, -q  Suppress non-error output (scripts)
  --help, -h   Show help
  --version    Show version

Aliases:
  l → ls, i → info, k → kill, f → free
```

## Examples

```bash
# List ports
portctl ls

# Inspect port
portctl info 3000

# Kill process (asks for confirmation)
portctl kill 3000

# Kill multiple ports
portctl kill 3000 8080

# Flush macOS DNS cache
portctl dns flush

# Force kill (no confirmation, SIGKILL)
portctl kill 3000 --force

# JSON
portctl ls --json

# Script-friendly
if portctl kill 3000 --force --quiet; then
    echo "Port freed"
fi
```

Sample output:

```text
PORT     PID      PROCESS       ADDRESS
3000     18231    node          127.0.0.1
5432     1298     postgres      127.0.0.1
8080     19342    pos-backend   0.0.0.0
```

JSON is pipe-ready:

```bash
portctl ls --json | jq '.[].port'
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0    | success (including "port not in use") |
| 1    | general error |
| 2    | invalid usage / invalid arguments |
| 3    | permission denied |

## Development

Requires Go 1.21+.

```bash
make build   # bin/portctl
make test    # go test ./...
make lint    # gofmt + go vet
make clean   # remove build artifacts
```

Architecture:

```text
CLI (cmd/) → Service (internal/app) → Port Detector (internal/ports) → OS
                                     → Process Manager (internal/system)
```

OS-specific code is isolated; business logic is OS-independent and testable
via `PortDetector` / `ProcessManager` interfaces.

## Testing

```bash
go test ./...        # unit + integration tests
```

Integration tests spawn their own temporary listener on a free dynamic port
and kill only that process — they never touch real processes on your machine.

## Contributing

1. Fork and create a branch
2. Run `make lint` and `make test` before committing
3. Keep the standard library only — avoid new dependencies
4. Open a pull request with a clear description

## License

[MIT](LICENSE)

# TASK.md — Build `portctl`

## 0. Project Overview

สร้าง Open Source CLI สำหรับจัดการ process และ network ports บนเครื่อง Developer โดยใช้ **Go**

Project name:

```text
portctl
```

เป้าหมายคือสร้าง CLI ที่ใช้งานง่ายแทนการจำ command อย่าง `lsof`, `kill`, `ps` และ command ที่เกี่ยวข้องกับ port

ตัวอย่างการใช้งาน:

```bash
portctl ls
portctl info 3000
portctl kill 3000
portctl kill 3000 8080
portctl free 3000
```

หลัง build แล้วต้องสามารถติดตั้งเป็น binary และเรียกจาก terminal ได้:

```bash
portctl
```

---

# 1. Requirements

## 1.1 Language

ใช้:

- Go
- Go Modules
- Standard Library เป็นหลัก

หลีกเลี่ยง dependency ที่ไม่จำเป็น

ถ้าจำเป็นต้องใช้ CLI framework ให้พิจารณา Cobra แต่ควรเริ่มจาก standard library ก่อน

---

# 2. Supported OS

ต้องออกแบบให้รองรับอย่างน้อย:

- macOS
- Linux

Windows สามารถวาง architecture รองรับในอนาคตได้ แต่ไม่จำเป็นต้อง implement ใน MVP

OS-specific code ต้องแยกออกจาก business logic

ตัวอย่าง:

```text
internal/
└── ports/
    ├── ports.go
    ├── ports_unix.go
    └── ports_windows.go
```

หาก Windows ยังไม่ implement ให้ return error ที่ชัดเจน

---

# 3. CLI Commands

ต้อง implement commands ต่อไปนี้

## 3.1 Help

```bash
portctl
portctl --help
portctl -h
```

แสดง:

```text
portctl - Port & Process Manager

Usage:
  portctl <command> [arguments]

Commands:
  ls       List ports and processes
  info     Show detailed information about a port
  kill     Kill process using a port
  free     Find and kill process using a port

Examples:
  portctl ls
  portctl info 3000
  portctl kill 3000
  portctl kill 3000 8080
  portctl free 3000
```

---

# 4. `portctl ls`

Command:

```bash
portctl ls
```

แสดง ports ที่กำลังถูกใช้งาน

ตัวอย่าง:

```text
PORT     PID      PROCESS       ADDRESS
3000     18231    node          127.0.0.1
8080     19342    pos-backend   0.0.0.0
5432     1298     postgres      127.0.0.1
```

ต้องสามารถตรวจสอบ:

- TCP
- UDP (ถ้าระบบรองรับ)
- PID
- Process name
- Address
- Port

ควร sort ตาม port จากน้อยไปมาก

---

# 5. `portctl info`

Command:

```bash
portctl info 3000
```

ตัวอย่าง output:

```text
Port:       3000
Protocol:   TCP
Address:    127.0.0.1
PID:        18231
Process:    node
Command:    node server.js
User:       obx
```

ถ้ามีหลาย process:

```text
Port: 3000

PID      PROCESS
18231    node
19420    node
```

ถ้าไม่มี process:

```text
Port 3000 is not in use.
```

Exit code ต้องเป็น non-zero เมื่อเกิด error จริง

---

# 6. `portctl kill`

Command:

```bash
portctl kill 3000
```

ต้อง:

1. ตรวจสอบ process ที่ใช้ port
2. แสดง process ที่พบ
3. ขอ confirmation ก่อน kill โดย default
4. kill process
5. แสดงผลลัพธ์

ตัวอย่าง:

```text
Found process:

PID      PROCESS      PORT
18231    node         3000

Kill process 18231? [y/N]:
```

เมื่อ user ตอบ `y`:

```text
✓ Process 18231 killed
✓ Port 3000 is now free
```

---

# 7. Multiple Ports

รองรับ:

```bash
portctl kill 3000 8080 4000
```

ตัวอย่าง:

```text
Port 3000
  PID 18231 node

Port 8080
  PID 19342 pos-backend

Port 4000
  PID 20012 node

Continue? [y/N]:
```

ต้องไม่หยุดทำงานเพียงเพราะ port ใด port หนึ่งไม่มี process

เช่น:

```bash
portctl kill 3000 9999 8080
```

ถ้า `9999` ไม่ถูกใช้งาน ให้แจ้ง:

```text
Port 9999 is not in use.
```

และดำเนินการกับ port อื่นต่อ

---

# 8. `portctl free`

`free` เป็น shortcut สำหรับหา process แล้ว kill

Command:

```bash
portctl free 3000
```

พฤติกรรม:

```text
portctl free 3000
```

เทียบเท่ากับ:

```text
find process
→ show process
→ confirm
→ kill
→ verify port is free
```

Output:

```text
Port 3000 is used by:

PID      PROCESS
18231    node

Kill process 18231? [y/N]: y

✓ Process killed
✓ Port 3000 is free
```

---

# 9. Force Mode

เพิ่ม option:

```bash
portctl kill 3000 --force
```

หรือ:

```bash
portctl kill 3000 -f
```

เมื่อใช้ `--force`:

- ไม่ต้องถาม confirmation
- kill process ทันที

ตัวอย่าง:

```bash
portctl kill 3000 -f
```

Output:

```text
✓ Killed node (PID 18231)
✓ Port 3000 is free
```

---

# 10. Signal Support

Default:

```bash
portctl kill 3000
```

ใช้ graceful termination ก่อน:

```text
SIGTERM
```

ถ้า process ยังไม่หยุด และ user ใช้:

```bash
--force
```

ให้ใช้:

```text
SIGKILL
```

ไม่ควรใช้ `kill -9` เป็น default

---

# 11. Safety

ห้าม kill process โดยไม่ตรวจสอบ

ต้องป้องกันกรณี:

```bash
portctl kill
portctl kill abc
portctl kill -1
portctl kill 0
```

ต้องแสดง error ที่อ่านง่าย:

```text
Error: invalid port "abc"
```

Port ต้องอยู่ในช่วง:

```text
1 - 65535
```

หาก port ไม่ถูกใช้งาน:

```text
Port 3000 is not in use.
```

---

# 12. Protected Processes

เพิ่ม safety check สำหรับ process สำคัญ

อย่างน้อยต้องระวัง:

```text
PID 1
```

และ system-critical processes

ห้าม kill PID 1 โดยเด็ดขาด

ถ้าพบ:

```text
Error: refusing to kill protected process (PID 1)
```

---

# 13. Permission Handling

หากไม่มี permission:

```text
Permission denied.
Try running:

sudo portctl kill 3000
```

ไม่ควร crash

ต้อง return error ที่อ่านง่าย

---

# 14. Architecture

ใช้ architecture ที่แยก CLI ออกจาก port detection

แนะนำ:

```text
portctl/
│
├── main.go
│
├── cmd/
│   ├── root.go
│   ├── ls.go
│   ├── info.go
│   ├── kill.go
│   └── free.go
│
├── internal/
│   ├── ports/
│   │   ├── port.go
│   │   ├── process.go
│   │   └── detector.go
│   │
│   └── system/
│       └── process.go
│
├── tests/
│
├── README.md
├── LICENSE
├── Makefile
├── go.mod
└── TASK.md
```

สามารถปรับ structure ได้หากมีเหตุผลที่ดีกว่า แต่ต้องรักษา separation ระหว่าง:

```text
CLI
↓
Service
↓
Port Detector
↓
OS
```

---

# 15. Domain Models

สร้าง model สำหรับ process:

```go
type Process struct {
    PID      int
    Name     string
    Command  string
    User     string
    Port     int
    Protocol string
    Address  string
}
```

สามารถปรับ fields ได้ตามความเหมาะสม

อย่าให้ CLI layer ผูกกับ `lsof` output โดยตรง

---

# 16. Port Detection

บน macOS/Linux สามารถใช้ OS command ที่มีอยู่ เช่น:

```bash
lsof
```

แต่ต้องสร้าง abstraction เช่น:

```go
type PortDetector interface {
    ListPorts() ([]Process, error)
    FindByPort(port int) ([]Process, error)
}
```

ไม่ให้ business logic เรียก `lsof` โดยตรงทุกที่

---

# 17. Process Killing

สร้าง abstraction:

```go
type ProcessManager interface {
    Terminate(pid int) error
    Kill(pid int) error
}
```

Default behavior:

```text
Terminate → SIGTERM
Kill      → SIGKILL
```

---

# 18. Error Handling

Error ต้อง readable

ตัวอย่าง:

```text
Error: invalid port: abc
Error: port must be between 1 and 65535
Error: permission denied
Error: process 12345 no longer exists
Error: failed to inspect port 3000
```

ห้ามแสดง stack trace ให้ user ใน normal CLI usage

---

# 19. Exit Codes

กำหนด exit codes:

```text
0 = success

1 = general error

2 = invalid usage / invalid arguments

3 = permission denied
```

ถ้า port ไม่ถูกใช้งาน:

```text
portctl info 3000
```

สามารถ return `0` เพราะ command ทำงานสำเร็จ เพียงแต่ไม่มี process

---

# 20. Output Modes

เพิ่ม option:

```bash
portctl ls --json
```

ตัวอย่าง:

```json
[
  {
    "port": 3000,
    "pid": 18231,
    "process": "node",
    "protocol": "tcp",
    "address": "127.0.0.1"
  }
]
```

JSON output ต้องไม่มี decorative text เช่น:

```text
✓
```

เพื่อให้สามารถใช้กับ:

```bash
portctl ls --json | jq
```

ได้

---

# 21. Quiet Mode

เพิ่ม:

```bash
portctl kill 3000 --force --quiet
```

ใช้สำหรับ scripts

ตัวอย่าง:

```bash
if portctl kill 3000 --force --quiet; then
    echo "Port freed"
fi
```

---

# 22. Command Aliases

รองรับ aliases:

```bash
portctl l
portctl i 3000
portctl k 3000
portctl f 3000
```

mapping:

```text
l → ls
i → info
k → kill
f → free
```

---

# 23. Shell Alias Recommendation

ใน README ให้แนะนำ user ที่ต้องการ command สั้นกว่านี้สามารถเพิ่ม:

```bash
alias ports="portctl ls"
alias kp="portctl kill"
```

ตัวอย่าง:

```bash
kp 3000
```

แต่ไม่ต้องให้ application แก้ไข `.zshrc` หรือ `.bashrc` อัตโนมัติ

---

# 24. Installation

ต้อง support:

## Build from source

```bash
git clone <repository>
cd portctl

go build -o portctl .
```

ติดตั้ง:

```bash
sudo mv portctl /usr/local/bin/
```

ตรวจสอบ:

```bash
portctl --version
```

---

# 25. Version

ต้องมี:

```bash
portctl --version
```

Output:

```text
portctl version 0.1.0
```

Version ต้องสามารถ inject ตอน build ได้ เช่น:

```bash
go build -ldflags "-X main.version=0.1.0"
```

---

# 26. Makefile

สร้าง Makefile:

```text
make build
make test
make lint
make install
make clean
```

ตัวอย่าง:

```bash
make build
```

สร้าง:

```text
bin/portctl
```

---

# 27. Testing

ต้องเขียน unit tests สำหรับ:

- port validation
- argument parsing
- process model
- port filtering
- multiple ports
- invalid ports
- protected PID
- JSON output
- exit codes

ห้ามให้ test ต้อง kill process จริงในเครื่อง developer

สำหรับ process manager ให้ใช้ mock/fake implementation

---

# 28. Integration Testing

สร้าง test สำหรับ:

```bash
portctl ls
portctl info <port>
```

ถ้าต้องการ test kill ให้สร้าง test process ชั่วคราวขึ้นมาเอง

ตัวอย่าง:

```text
start temporary process
↓
bind test port
↓
portctl info
↓
portctl kill
↓
verify process stopped
```

ห้ามใช้ port ที่ application จริงอาจใช้งาน

ให้เลือก dynamic/free port

---

# 29. Formatting

ต้องใช้:

```bash
gofmt
go vet
```

ก่อน commit

ถ้าใช้ linter ให้เลือก tool ที่เหมาะสม เช่น `golangci-lint`

---

# 30. README.md

README ต้องมี:

## Project description

อธิบายว่า:

> `portctl` is a simple CLI tool for inspecting and managing processes using network ports.

## Features

อย่างน้อย:

```text
✓ List ports
✓ Inspect processes
✓ Kill processes by port
✓ Multiple ports
✓ Force mode
✓ JSON output
✓ macOS support
✓ Linux support
```

## Installation

## Usage

## Examples

## Development

## Testing

## Contributing

## License

---

# 31. README Example

ต้องมีตัวอย่าง:

```bash
# List ports
portctl ls

# Inspect port
portctl info 3000

# Kill process
portctl kill 3000

# Kill multiple ports
portctl kill 3000 8080

# Force kill
portctl kill 3000 --force

# JSON
portctl ls --json
```

---

# 32. Open Source License

ใช้:

```text
MIT License
```

สร้าง:

```text
LICENSE
```

Copyright:

```text
Copyright (c) <YEAR> <AUTHOR>
```

ให้ user สามารถเปลี่ยนชื่อและ year ได้

---

# 33. GitHub Repository

เตรียม project ให้เหมาะกับ GitHub Open Source

Repository:

```text
portctl
```

แนะนำ description:

```text
A simple CLI for inspecting and managing processes by network port.
```

Topics:

```text
go
golang
cli
terminal
developer-tools
ports
process-manager
macos
linux
```

---

# 34. GitHub Actions

สร้าง:

```text
.github/
└── workflows/
    └── ci.yml
```

CI ต้องทำ:

```text
checkout
↓
setup Go
↓
go test ./...
↓
go vet ./...
↓
go build
```

ต้อง test อย่างน้อย:

```text
ubuntu-latest
macos-latest
```

---

# 35. Cross Compilation

ต้องสามารถ build:

```text
darwin/arm64
darwin/amd64
linux/amd64
linux/arm64
```

ตัวอย่าง:

```bash
GOOS=darwin GOARCH=arm64 go build -o bin/portctl-darwin-arm64 .
GOOS=darwin GOARCH=amd64 go build -o bin/portctl-darwin-amd64 .
GOOS=linux GOARCH=amd64 go build -o bin/portctl-linux-amd64 .
GOOS=linux GOARCH=arm64 go build -o bin/portctl-linux-arm64 .
```

---

# 36. Release

เตรียม release process สำหรับ:

```text
v0.1.0
v0.2.0
v1.0.0
```

แนะนำใช้ GitHub Releases

Artifact names:

```text
portctl_darwin_arm64
portctl_darwin_amd64
portctl_linux_arm64
portctl_linux_amd64
```

---

# 37. Optional: Homebrew

หลัง MVP เสร็จ สามารถเตรียม Homebrew Formula

เป้าหมาย:

```bash
brew install portctl
```

แต่ไม่ต้อง implement ก่อน MVP จะเสร็จ

---

# 38. CLI UX

CLI ต้องเน้น:

- อ่านง่าย
- command สั้น
- error ชัดเจน
- ไม่ spam output
- ใช้สีได้ถ้า terminal รองรับ
- ถ้า pipe output ต้องไม่ใส่ color/emoji

ตัวอย่าง:

```bash
portctl ls
```

ควรดูประมาณ:

```text
PORT     PID      PROCESS       ADDRESS
────────────────────────────────────────
3000     18231    node          127.0.0.1
4000     19231    python        127.0.0.1
5432     1298     postgres      127.0.0.1
8080     19342    pos-backend   0.0.0.0
```

---

# 39. Performance

`portctl ls` ต้องทำงานเร็ว

หลีกเลี่ยง:

```text
เรียก lsof หลายครั้งโดยไม่จำเป็น
```

ควรเรียก system command เท่าที่จำเป็น และ parse output ใน memory

---

# 40. Security

ห้าม:

- execute arbitrary user input ผ่าน shell
- concatenate command string แล้วส่งให้ shell
- ใช้ `sh -c` ถ้าไม่จำเป็น

ตัวอย่างที่ควรหลีกเลี่ยง:

```go
exec.Command("sh", "-c", "kill "+userInput)
```

ควรใช้:

```go
exec.Command("kill", pidString)
```

และ validate PID ก่อน

---

# 41. UX Edge Cases

ต้องจัดการ:

### Port ไม่มี process

```text
Port 3000 is not in use.
```

### Process หายไปก่อน kill

```text
Process 12345 no longer exists.
```

### Permission denied

```text
Permission denied.
Try running with sudo.
```

### Invalid port

```text
Error: port must be between 1 and 65535
```

### Invalid command

```text
Error: unknown command "foo"

Run:
  portctl --help
```

### No arguments

```bash
portctl
```

แสดง help

---

# 42. Development Workflow

ทำตามลำดับนี้:

## Phase 1 — Project

- [ ] Initialize Go module
- [ ] Create project structure
- [ ] Create `main.go`
- [ ] Create version system

## Phase 2 — Port Detection

- [ ] Implement port model
- [ ] Implement process model
- [ ] Implement macOS detector
- [ ] Implement Linux detector
- [ ] Parse `lsof`
- [ ] Add detector abstraction

## Phase 3 — CLI

- [ ] Implement root command
- [ ] Implement `ls`
- [ ] Implement `info`
- [ ] Implement `kill`
- [ ] Implement `free`
- [ ] Implement aliases

## Phase 4 — Safety

- [ ] Validate ports
- [ ] Validate PIDs
- [ ] Protect PID 1
- [ ] Add confirmation
- [ ] Add `--force`
- [ ] Add SIGTERM/SIGKILL behavior
- [ ] Handle permission errors

## Phase 5 — Output

- [ ] Table output
- [ ] JSON output
- [ ] Quiet mode
- [ ] Terminal detection
- [ ] Color handling

## Phase 6 — Tests

- [ ] Unit tests
- [ ] Mock process manager
- [ ] Mock port detector
- [ ] Integration tests
- [ ] Edge case tests

## Phase 7 — Documentation

- [ ] README
- [ ] LICENSE
- [ ] Usage examples
- [ ] Development guide

## Phase 8 — CI

- [ ] GitHub Actions
- [ ] Go test
- [ ] Go vet
- [ ] Build macOS
- [ ] Build Linux

## Phase 9 — Release

- [ ] Version `0.1.0`
- [ ] Build binaries
- [ ] Git tag
- [ ] GitHub Release
- [ ] Release notes

## Phase 10 — Optional

- [ ] Homebrew
- [ ] Shell completion
- [ ] Windows support
- [ ] Docker port detection
- [ ] Interactive mode

---

# 43. Shell Completion — Optional

รองรับในอนาคต:

```bash
portctl completion zsh
portctl completion bash
portctl completion fish
```

เป้าหมาย:

```bash
portctl k <TAB>
```

แสดง:

```text
kill
```

และ:

```bash
portctl kill --<TAB>
```

แสดง:

```text
--force
--quiet
```

---

# 44. Future Features

ไม่ต้อง implement ใน MVP แต่ architecture ควรรองรับ:

```text
portctl watch 3000
portctl kill --all
portctl restart 3000
portctl tree 3000
portctl docker
portctl open 3000
portctl scan
```

ตัวอย่าง:

```bash
portctl watch 3000
```

สามารถ monitor ว่า process เปลี่ยนหรือไม่

---

# 45. Definition of Done

Project ถือว่าเสร็จเมื่อ:

- [ ] `go test ./...` ผ่าน
- [ ] `go vet ./...` ผ่าน
- [ ] `go build` ผ่าน
- [ ] macOS ใช้งานได้
- [ ] Linux ใช้งานได้
- [ ] `portctl ls` ใช้งานได้
- [ ] `portctl info 3000` ใช้งานได้
- [ ] `portctl kill 3000` ใช้งานได้
- [ ] `portctl kill 3000 8080` ใช้งานได้
- [ ] `portctl free 3000` ใช้งานได้
- [ ] `--force` ใช้งานได้
- [ ] `--json` ใช้งานได้
- [ ] `--quiet` ใช้งานได้
- [ ] invalid input ไม่ทำให้ program crash
- [ ] protected process ไม่สามารถถูก kill โดยไม่ได้ตั้งใจ
- [ ] README ครบ
- [ ] LICENSE มี
- [ ] GitHub Actions ผ่าน
- [ ] สามารถ build macOS ARM64
- [ ] สามารถ build Linux AMD64
- [ ] สามารถสร้าง GitHub Release ได้

---

# 46. Final Validation

หลัง implement เสร็จ ให้ทดสอบจริง:

```bash
go test ./...
go vet ./...
go build -o portctl .
```

จากนั้น:

```bash
./portctl --version
./portctl --help
./portctl ls
./portctl info 3000
```

สร้าง test server:

```bash
python3 -m http.server 3000
```

เปิด terminal อีกอัน:

```bash
./portctl info 3000
```

ต้องพบ Python process

จากนั้น:

```bash
./portctl kill 3000
```

ยืนยันว่า:

```bash
curl http://localhost:3000
```

ไม่สามารถเชื่อมต่อได้

จากนั้นทดสอบ:

```bash
./portctl ls --json
```

และ:

```bash
./portctl kill 3000 --force
```

---

# 47. Important Implementation Rules

1. อย่าเขียนทุกอย่างไว้ใน `main.go`
2. อย่าให้ CLI ผูกกับ `lsof` โดยตรง
3. แยก OS-specific implementation
4. ใช้ interface สำหรับ detector และ process manager
5. Validate user input ทุกครั้ง
6. อย่าใช้ shell execution กับ user input
7. Default kill ต้องใช้ graceful termination
8. `--force` เท่านั้นที่ใช้ SIGKILL
9. ต้องมี confirmation ก่อน kill ถ้าไม่ใช้ `--force`
10. ต้องมี unit tests
11. ต้องไม่ kill PID 1
12. ต้อง handle permission errors
13. ต้องรองรับ pipe/automation
14. JSON output ต้องเป็น valid JSON เท่านั้น
15. Code ต้องผ่าน `gofmt`, `go test`, `go vet`
16. Documentation ต้องทำพร้อม implementation
17. อย่าเพิ่ม dependency ถ้า standard library เพียงพอ
18. ถ้าพบ design ที่ดีกว่าในระหว่างพัฒนา สามารถปรับ architecture ได้ แต่ต้องรักษา requirements ทั้งหมดในไฟล์นี้

---

# 48. Final Goal

เมื่อ project เสร็จ ผู้ใช้ต้องสามารถทำสิ่งนี้ได้:

```bash
# ดู port ทั้งหมด
portctl ls

# ดู port
portctl info 3000

# kill port
portctl kill 3000

# kill หลาย port
portctl kill 3000 4000 8080

# force kill
portctl kill 3000 -f

# shortcut
portctl k 3000

# JSON
portctl ls --json
```

และสามารถเพิ่ม shell alias:

```bash
alias ports="portctl ls"
alias kp="portctl kill"
```

ทำให้ workflow กลายเป็น:

```bash
ports
kp 3000
```

Project ต้องพร้อมสำหรับการเปิดเป็น **Open Source GitHub project** และสามารถพัฒนาต่อเป็น Developer Tool ที่มี release/versioning ได้

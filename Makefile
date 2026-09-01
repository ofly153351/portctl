# portctl Makefile

BINARY := portctl
BIN_DIR := bin
VERSION ?= 0.1.0

LDFLAGS := -ldflags "-X main.version=$(VERSION)"

PLATFORMS := \
	darwin/arm64 \
	darwin/amd64 \
	linux/amd64 \
	linux/arm64

.PHONY: build test lint install clean cross-compile

## build: build bin/portctl
build:
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) .

## test: run all unit and integration tests
test:
	go test ./...

## lint: gofmt check + go vet
lint:
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then \
		echo "gofmt needed on:"; echo "$$out"; exit 1; \
	fi
	go vet ./...

## install: build and install to /usr/local/bin
install: build
	sudo mv $(BIN_DIR)/$(BINARY) /usr/local/bin/$(BINARY)

## clean: remove build artifacts
clean:
	rm -rf $(BIN_DIR)

## cross-compile: build release binaries for all supported platforms
cross-compile:
	@mkdir -p $(BIN_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		echo "Building $${os}/$${arch}..."; \
		GOOS=$${os} GOARCH=$${arch} \
			go build $(LDFLAGS) \
			-o $(BIN_DIR)/$(BINARY)_$${os}_$${arch} . ; \
	done
	@ls -la $(BIN_DIR)

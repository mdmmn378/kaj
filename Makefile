# Kaj Todo List - Build Configuration

# Get version info from git
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE)

# Default target
.PHONY: build
build:
	go build -ldflags="$(LDFLAGS)" -o kaj

.PHONY: install
install: build
	sudo mv kaj /usr/local/bin/

.PHONY: clean
clean:
	rm -f kaj kaj-*

.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

.PHONY: release
release:
	# Build for Linux AMD64
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o kaj-linux-amd64
	
	# Build for macOS ARM64 (Apple Silicon)  
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o kaj-darwin-arm64
	
	# Build for macOS AMD64 (Intel)
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o kaj-darwin-amd64
	
	# Build for Windows AMD64
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o kaj-windows-amd64.exe
	
	# Create checksums
	sha256sum kaj-* > checksums.sha256

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build     - Build the binary with version info from git"
	@echo "  install   - Build and install to /usr/local/bin"
	@echo "  clean     - Remove built binaries"
	@echo "  version   - Show version information"
	@echo "  release   - Build binaries for all platforms"
	@echo "  help      - Show this help message"

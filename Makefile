# gowsoos Makefile

# Variables
BINARY_NAME=gowsoos
MAIN_FILE=main.go
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X gowsoos/cmd.Version=$(VERSION) -X gowsoos/cmd.Commit=$(COMMIT) -X gowsoos/cmd.Date=$(DATE)"

# FHS (Filesystem Hierarchy Standard) paths
PREFIX?=/usr
BINDIR=$(PREFIX)/bin
SYSCONFDIR=/etc/gowsoos
SYSTEMDDIR=/etc/systemd/system
LOGDIR=/var/log/gowsoos
RUNDIR=/var/run/gowsoos

# User and group for service
GOWSOOS_USER?=gowsoos
GOWSOOS_GROUP?=gowsoos

# Default target
.PHONY: all
all: clean deps build

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Build the binary
.PHONY: build
build:
	mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)

# Build for multiple platforms
.PHONY: build-all
build-all:
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_FILE)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_FILE)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_FILE)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_FILE)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_FILE)

# Install the application (FHS compliant)
.PHONY: install
install: build
	# Create directories
	install -d $(DESTDIR)$(BINDIR)
	install -d $(DESTDIR)$(SYSCONFDIR)
	install -d $(DESTDIR)$(SYSTEMDDIR)
	install -d $(DESTDIR)$(LOGDIR)
	install -d $(DESTDIR)$(RUNDIR)
	
	# Install binary
	install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(DESTDIR)$(BINDIR)/$(BINARY_NAME)
	
	# Install configuration file
	install -m 644 config.yaml $(DESTDIR)$(SYSCONFDIR)/config.yaml.example
	
	# Install systemd service
	install -m 644 gowsoos.service $(DESTDIR)$(SYSTEMDDIR)/gowsoos.service

	# Create user and group (if not exists)
	@if ! id $(GOWSOOS_USER) >/dev/null 2>&1; then \
		echo "Creating user $(GOWSOOS_USER)..."; \
		useradd -r -s /bin/false -d $(RUNDIR) $(GOWSOOS_USER) || true; \
	fi

	# Set ownership
	chown -R $(GOWSOOS_USER):$(GOWSOOS_GROUP) $(DESTDIR)$(LOGDIR) $(DESTDIR)$(RUNDIR)

	@echo "Installation completed!"
	@echo "Binary: $(BINDIR)/$(BINARY_NAME)"
	@echo "Config: $(SYSCONFDIR)/config.yaml.example"
	@echo "Service: $(SYSTEMDDIR)/gowsoos.service"
	@echo ""
	@echo "Next steps:"
	@echo "1. Copy and edit config: sudo cp $(SYSCONFDIR)/config.yaml.example $(SYSCONFDIR)/config.yaml"
	@echo "2. Enable service: sudo systemctl enable gowsoos"
	@echo "3. Start service: sudo systemctl start gowsoos"

# Uninstall the application
.PHONY: uninstall
uninstall:
	# Stop and disable service
	@if systemctl is-active --quiet gowsoos 2>/dev/null; then \
		echo "Stopping gowsoos service..."; \
		systemctl stop gowsoos; \
	fi
	@if systemctl is-enabled --quiet gowsoos 2>/dev/null; then \
		echo "Disabling gowsoos service..."; \
		systemctl disable gowsoos; \
	fi

	# Remove files
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY_NAME)
	rm -f $(DESTDIR)$(SYSTEMDDIR)/gowsoos.service
	rm -rf $(DESTDIR)$(SYSCONFDIR)

	# Remove user (optional, commented for safety)
	# @if id $(GOWSOOS_USER) >/dev/null 2>&1; then \
	# 	echo "Removing user $(GOWSOOS_USER)..."; \
	# 	userdel $(GOWSOOS_USER); \
	# fi
	
	@echo "Uninstallation completed!"

# Install systemd service only
.PHONY: install-service
install-service:
	install -d $(DESTDIR)$(SYSTEMDDIR)
	install -m 644 gowsoos.service $(DESTDIR)$(SYSTEMDDIR)/gowsoos.service
	systemctl daemon-reload
	@echo "Service installed to $(SYSTEMDDIR)/gowsoos.service"

# Enable and start service
.PHONY: enable-service
enable-service:
	systemctl enable gowsoos
	systemctl start gowsoos
	@echo "gowsoos service enabled and started"

# Disable and stop service
.PHONY: disable-service
disable-service:
	systemctl stop gowsoos || true
	systemctl disable gowsoos || true
	@echo "gowsoos service disabled and stopped"

# Status of service
.PHONY: status
status:
	systemctl status gowsoos

# View logs
.PHONY: logs
logs:
	journalctl -u gowsoos -f

# Run the application
.PHONY: run
run:
	go run $(MAIN_FILE)

# Run with configuration file
.PHONY: run-config
run-config:
	go run $(MAIN_FILE) --config $(SYSCONFDIR)/config.yaml

# Run with TLS enabled
.PHONY: run-tls
run-tls:
	go run $(MAIN_FILE) --tls --tls-addr :443 --private-key /path/to/private.pem --public-key /path/to/public.key

# Run with metrics
.PHONY: run-metrics
run-metrics:
	go run $(MAIN_FILE) --metrics --metrics-port :9090

# Test the application
.PHONY: test
test:
	go test -v ./...

# Benchmark the application
.PHONY: bench
bench:
	go test -bench=. ./...

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	golangci-lint run

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
	go clean

# Create release package
.PHONY: package
package: build
	# Create tarball
	mkdir -p $(BUILD_DIR)/package
	cp $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/package/
	cp config.yaml $(BUILD_DIR)/package/
	cp gowsoos.service $(BUILD_DIR)/package/
	cp README.md $(BUILD_DIR)/package/
	cd $(BUILD_DIR)/package && tar -czf ../$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz .
	@echo "Package created: $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz"

# Development setup
.PHONY: dev-setup
dev-setup:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build           - Build the binary"
	@echo "  build-all       - Build for multiple platforms"
	@echo "  install         - Install to FHS paths (/usr/bin, /etc/gowsoos)"
	@echo "  uninstall       - Remove from FHS paths"
	@echo "  install-service - Install systemd service only"
	@echo "  enable-service  - Enable and start systemd service"
	@echo "  disable-service - Disable and stop systemd service"
	@echo "  status          - Show service status"
	@echo "  logs            - Show service logs"
	@echo "  package         - Create release package"
	@echo "  run             - Run the application"
	@echo "  run-config      - Run with config from /etc/gowsoos"
	@echo "  run-tls         - Run with TLS enabled"
	@echo "  run-metrics     - Run with metrics enabled"
	@echo "  test            - Run tests"
	@echo "  bench           - Run benchmarks"
	@echo "  fmt             - Format code"
	@echo "  lint            - Lint code"
	@echo "  clean           - Clean build artifacts"
	@echo "  deps            - Install dependencies"
	@echo "  dev-setup       - Setup development tools"
	@echo "  help            - Show this help"
	@echo ""
	@echo "Variables:"
	@echo "  PREFIX          - Installation prefix (default: /usr)"
	@echo "  DESTDIR         - Destination directory for packaging"
	@echo "  GOWSOOS_USER   - Service user (default: gowsoos)"
	@echo "  GOWSOOS_GROUP  - Service group (default: gowsoos)"

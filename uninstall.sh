#!/bin/bash
# gowsoos Uninstallation Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SERVICE_NAME="gowsoos"
SERVICE_USER="gowsoos"
INSTALL_DIR="/usr/bin"
CONFIG_DIR="/etc/gowsoos"
SERVICE_DIR="/etc/systemd/system"

# Functions
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root"
        exit 1
    fi
}

# Stop and disable service
stop_service() {
    print_status "Stopping and disabling service..."
    
    if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        systemctl stop "$SERVICE_NAME"
        print_status "Service stopped"
    else
        print_warning "Service was not running"
    fi
    
    if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
        systemctl disable "$SERVICE_NAME"
        print_status "Service disabled"
    else
        print_warning "Service was not enabled"
    fi
}

# Remove files
remove_files() {
    print_status "Removing installed files..."
    
    # Remove binary
    if [[ -f "$INSTALL_DIR/gowsoos" ]]; then
        rm -f "$INSTALL_DIR/gowsoos"
        print_status "Removed binary: $INSTALL_DIR/gowsoos"
    fi

    # Remove service file
    if [[ -f "$SERVICE_DIR/gowsoos.service" ]]; then
        rm -f "$SERVICE_DIR/gowsoos.service"
        print_status "Removed service file: $SERVICE_DIR/gowsoos.service"
    fi
    
    # Remove configuration directory
    if [[ -d "$CONFIG_DIR" ]]; then
        read -p "Remove configuration directory $CONFIG_DIR? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -rf "$CONFIG_DIR"
            print_status "Removed configuration directory"
        else
            print_warning "Configuration directory preserved"
        fi
    fi
    
    # Remove log directory
    if [[ -d "/var/log/gowsoos" ]]; then
        read -p "Remove log directory /var/log/gowsoos? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -rf "/var/log/gowsoos"
            print_status "Removed log directory"
        else
            print_warning "Log directory preserved"
        fi
    fi

    # Remove run directory
    if [[ -d "/var/run/gowsoos" ]]; then
        rm -rf "/var/run/gowsoos"
        print_status "Removed run directory"
    fi
}

# Remove service user (optional)
remove_user() {
    if id "$SERVICE_USER" &>/dev/null; then
        read -p "Remove service user $SERVICE_USER? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            userdel "$SERVICE_USER"
            print_status "Removed service user"
        else
            print_warning "Service user preserved"
        fi
    else
        print_warning "Service user does not exist"
    fi
}

# Reload systemd
reload_systemd() {
    print_status "Reloading systemd..."
    systemctl daemon-reload
}

# Show completion message
show_completion() {
    print_status "Uninstallation completed!"
    echo
    echo "gowsoos has been removed from your system."
    echo "If you kept the configuration directory, you can reuse it for future installations."
}

# Main function
main() {
    echo "gowsoos Uninstallation Script"
    echo "============================"
    echo
    echo "This will remove gowsoos from your system."
    echo
    
    read -p "Continue with uninstallation? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Uninstallation cancelled."
        exit 0
    fi
    
    check_root
    stop_service
    remove_files
    remove_user
    reload_systemd
    show_completion
}

# Run main function
main
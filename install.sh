#!/bin/bash
# gowsoos Installation Script
# This script installs gowsoos with FHS-compliant paths

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SERVICE_NAME="gowsoos"
SERVICE_USER="gowsoos"
SERVICE_GROUP="gowsoos"
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

# Check system requirements
check_requirements() {
    print_status "Checking system requirements..."
    
    # Check if systemd is available
    if ! command -v systemctl &> /dev/null; then
        print_error "systemd is required but not installed"
        exit 1
    fi
    
    # Check if go is available (for source installation)
    if [[ "$1" == "source" ]] && ! command -v go &> /dev/null; then
        print_error "Go is required for source installation"
        exit 1
    fi
    
    print_status "System requirements check passed"
}

# Create service user
create_user() {
    print_status "Creating service user..."
    
    if ! id "$SERVICE_USER" &>/dev/null; then
        useradd -r -s /bin/false -d /var/run/gowsoos "$SERVICE_USER"
        print_status "Created user: $SERVICE_USER"
    else
        print_warning "User $SERVICE_USER already exists"
    fi
}

# Install from binary package
install_binary() {
    local package_file="$1"
    
    print_status "Installing from binary package: $package_file"
    
    # Extract package
    cd /tmp
    tar -xzf "$package_file"
    
    # Install binary
    cp gowsoos "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/gowsoos"

    # Install configuration
    mkdir -p "$CONFIG_DIR"
    cp config.yaml "$CONFIG_DIR/config.yaml.example"

    # Install service
    cp gowsoos.service "$SERVICE_DIR/"

    # Create directories
    mkdir -p /var/log/gowsoos
    mkdir -p /var/run/gowsoos
    mkdir -p "$CONFIG_DIR/tls"

    # Set ownership
    chown -R "$SERVICE_USER:$SERVICE_GROUP" /var/log/gowsoos
    chown -R "$SERVICE_USER:$SERVICE_GROUP" /var/run/gowsoos
    chown -R "$SERVICE_USER:$SERVICE_GROUP" "$CONFIG_DIR"
    
    # Cleanup
    rm -rf gowsoos config.yaml gowsoos.service README.md
    
    print_status "Binary installation completed"
}

# Install from source
install_source() {
    local source_dir="$1"
    
    print_status "Installing from source: $source_dir"
    
    cd "$source_dir"
    
    # Build
    make clean
    make build
    
    # Install using make
    make install
    
    print_status "Source installation completed"
}

# Setup configuration
setup_config() {
    print_status "Setting up configuration..."
    
    if [[ ! -f "$CONFIG_DIR/config.yaml" ]]; then
        cp "$CONFIG_DIR/config.yaml.example" "$CONFIG_DIR/config.yaml"
        print_status "Created default configuration file"
    else
        print_warning "Configuration file already exists, skipping"
    fi
    
    print_status "Configuration file location: $CONFIG_DIR/config.yaml"
}

# Enable and start service
setup_service() {
    print_status "Setting up systemd service..."
    
    # Reload systemd
    systemctl daemon-reload
    
    # Enable service
    systemctl enable "$SERVICE_NAME"
    
    print_status "Service enabled. Start with: systemctl start $SERVICE_NAME"
}

# Show post-installation information
show_info() {
    print_status "Installation completed successfully!"
    echo
    echo "Important paths:"
    echo "  Binary: $INSTALL_DIR/gowsoos"
    echo "  Config: $CONFIG_DIR/config.yaml"
    echo "  Service: $SERVICE_DIR/$SERVICE_NAME.service"
    echo "  Logs: /var/log/gowsoos/"
    echo
    echo "Next steps:"
    echo "  1. Edit configuration: sudo nano $CONFIG_DIR/config.yaml"
    echo "  2. Start service: sudo systemctl start $SERVICE_NAME"
    echo "  3. Check status: sudo systemctl status $SERVICE_NAME"
    echo "  4. View logs: sudo journalctl -u $SERVICE_NAME -f"
    echo
    echo "For TLS configuration:"
    echo "  1. Place certificates in $CONFIG_DIR/tls/"
    echo "  2. Update config.yaml with certificate paths"
    echo "  3. Restart service: sudo systemctl restart $SERVICE_NAME"
}

# Main installation function
main() {
    echo "gowsoos Installation Script"
    echo "=========================="
    echo
    
    check_root
    check_requirements "$1"
    create_user
    
    case "$1" in
        "binary")
            if [[ -z "$2" ]]; then
                print_error "Please provide binary package file"
                echo "Usage: $0 binary <package-file.tar.gz>"
                exit 1
            fi
            install_binary "$2"
            ;;
        "source")
            if [[ -z "$2" ]]; then
                print_error "Please provide source directory"
                echo "Usage: $0 source <source-directory>"
                exit 1
            fi
            install_source "$2"
            ;;
        *)
            print_error "Invalid installation type"
            echo "Usage: $0 <binary|source> <path>"
            exit 1
            ;;
    esac
    
    setup_config
    setup_service
    show_info
}

# Run main function with all arguments
main "$@"
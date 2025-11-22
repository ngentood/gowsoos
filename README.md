# gowsoos
SSH over HTTP Websocket Proxy with SSL SNI Support (up to 20 times faster than Python similar proxy and 100 times more users by CPU)

Tunnel for SSH with HTTP Websocket handler.

## Features

- High-performance SSH over HTTP WebSocket proxy
- TLS/SSL support with SNI (Server Name Indication)
- Two TLS modes: `handshake` and `stunnel`
- Configuration file support (YAML)
- Prometheus metrics integration
- Structured JSON logging
- Graceful shutdown
- Backward compatibility with original CLI

## Installation

### From Source
```bash
git clone <repository-url>
cd gowsoos
make build
sudo make install
```



### Manual Installation
```bash
# Build
make build

# Install to FHS paths
sudo make install

# Enable and start service
sudo systemctl enable gowsoos
sudo systemctl start gowsoos
```




## Systemd Service Management

### Service Commands
```bash
# Start service
sudo systemctl start gowsoos

# Stop service
sudo systemctl stop gowsoos

# Enable service (start on boot)
sudo systemctl enable gowsoos

# Disable service
sudo systemctl disable gowsoos

# Check service status
sudo systemctl status gowsoos

# View service logs
sudo journalctl -u gowsoos -f

# Restart service
sudo systemctl restart gowsoos
```

### Service Configuration
The systemd service includes:
- **Security**: Running as non-root user `gowsoos`
- **Sandboxing**: PrivateTmp, ProtectSystem, ProtectHome
- **Resource Limits**: File descriptors and process limits
- **Capabilities**: Only CAP_NET_BIND_SERVICE for privileged ports
- **Logging**: Structured logging to journald

## Configuration

### Command Line Options
```bash
./gowsoos --help
```

### Configuration File
Create a `config.yaml` file:
```yaml
# Server configuration
address: ":2086"
dst_address: "127.0.0.1:22"

# TLS configuration
tls_enabled: false
tls_address: ":443"
tls_private_key: "/path/to/private.pem"
tls_public_key: "/path/to/public.key"
tls_mode: "handshake"

# Logging and metrics
log_level: "info"
metrics_enabled: false
metrics_port: ":9090"
```

## Usage

### Basic HTTP Mode
```bash
./gowsoos -addr :80 -dstAddr 127.0.0.1:22
```

### Using Configuration File
```bash
./gowsoos --config config.yaml
```

### TLS Stunnel Mode
```bash
./gowsoos --tls-mode "stunnel" --addr :80 --tls --tls-addr :443 \
  --private-key /root/cert/fullchain.pem \
  --public-key /root/cert/yourdomaintls.key
```

### TLS Handshake Mode
```bash
./gowsoos --tls-mode "handshake" --addr :80 --tls --tls-addr :443 \
  --private-key /root/cert/fullchain.pem \
  --public-key /root/cert/yourdomaintls.key \
  --custom-handshake "101 Switching Protocols"
```

### With Metrics
```bash
./gowsoos --metrics --metrics-port :9090
```

## Client Configuration

### HTTP Injector for Android
**Client:**
```
Payload: GET / HTTP/1.1[crlf]Host: myserver.com[crlf]Upgrade: websocket[crlf][crlf]
Proxy: IP: 192.168.1.10, Port: 80
```

**Server:**
```bash
./gowsoos -addr :80 -dstAddr 127.0.0.1:22
```

### SSL Stunnel Mode
**Client:**
```
SNI: yourdomaintls.com
SSH: 192.168.1.10
Port: 443
```

**Server:**
```bash
./gowsoos --tls-mode "stunnel" --addr :80 --tls --tls-addr :443 \
  --private-key /root/cert/fullchain.pem \
  --public-key /root/cert/yourdomaintls.key
```

### SSL+HTTP Payload Mode
**Client:**
```
SNI: yourdomaintls.com
Payload: GET / HTTP/1.1[crlf][crlf]
SSH: 192.168.1.10
Port: 443
```

**Server:**
```bash
./gowsoos --tls-mode "handshake" --addr :80 --tls --tls-addr :443 \
  --private-key /root/cert/fullchain.pem \
  --public-key /root/cert/yourdomaintls.key
```

## Metrics

When metrics are enabled, Prometheus metrics are available at `http://localhost:9090/metrics`:

- `gowsoos_connections_total` - Total number of connections
- `gowsoos_connections_active` - Number of active connections
- `gowsoos_bytes_transferred_total` - Total bytes transferred
- `gowsoos_connection_duration_seconds` - Connection duration
- `gowsoos_errors_total` - Total number of errors

## Development

### Build
```bash
make build
```

### Run Tests
```bash
make test
```

### Development Mode
```bash
make run
```

### Build for Multiple Platforms
```bash
make build-all
```

## TLS Certificate

Get your free TLS certificate from CertBot:
```bash
sudo certbot certonly --webroot
https://certbot.eff.org/instructions?ws=webproduct&os=ubuntufocal
```

## Backward Compatibility

The refactored version maintains full backward compatibility with the original command-line interface:

```bash
# Original usage still works
./gowsoos -addr :80 -dstAddr 127.0.0.1:22 -tls -tls_addr :443
```

## Environment Variables

Configuration can be overridden with environment variables:
```bash
export GOWSOOS_ADDRESS=:8080
export GOWSOOS_DST_ADDRESS=192.168.1.100:22
export GOWSOOS_TLS_ENABLED=true
./gowsoos
```

## Uninstallation

### Using Uninstall Script (Recommended)
```bash
sudo ./uninstall.sh
```

### Manual Uninstallation
```bash
# Stop and disable service
sudo systemctl stop gowsoos
sudo systemctl disable gowsoos

# Remove files
sudo rm -f /usr/bin/gowsoos
sudo rm -f /etc/systemd/system/gowsoos.service
sudo rm -rf /etc/gowsoos
sudo rm -rf /var/log/gowsoos
sudo rm -rf /var/run/gowsoos

# Remove service user (optional)
sudo userdel gowsoos

# Reload systemd
sudo systemctl daemon-reload
```


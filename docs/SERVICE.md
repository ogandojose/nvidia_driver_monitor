# NVIDIA Driver Monitor - Systemd Service

This document describes how to install and manage the NVIDIA Driver Monitor as a systemd service.

## Quick Start

1. **Build the application:**
   ```bash
   make web
   ```

2. **Install as systemd service:**
   ```bash
   make install-service
   ```

3. **Start the service:**
   ```bash
   make service-start
   ```

4. **Access the web interface:**
   Open http://localhost:8080 in your browser

## Service Installation

### Automatic Installation (Recommended)

```bash
# Build and install the service
make install-service

# Start the service
make service-start

# Check service status
make service-status
```

### Manual Installation

```bash
# Make scripts executable
chmod +x install-service.sh

# Run the installation script
sudo ./install-service.sh
```

## Service Management

### Using Makefile (Recommended)

```bash
# Start the service
make service-start

# Stop the service
make service-stop

# Restart the service
make service-restart

# Check service status
make service-status

# View service logs
make service-logs
```

### Using Service Manager Script

```bash
# Start the service
./service-manager.sh start

# Stop the service
./service-manager.sh stop

# Restart the service
./service-manager.sh restart

# Check status
./service-manager.sh status

# View logs
./service-manager.sh logs

# Enable auto-start on boot
./service-manager.sh enable

# Disable auto-start on boot
./service-manager.sh disable
```

### Using systemctl Directly

```bash
# Start the service
sudo systemctl start nvidia-driver-monitor

# Stop the service
sudo systemctl stop nvidia-driver-monitor

# Restart the service
sudo systemctl restart nvidia-driver-monitor

# Check status
sudo systemctl status nvidia-driver-monitor

# Enable auto-start on boot
sudo systemctl enable nvidia-driver-monitor

# View logs
sudo journalctl -u nvidia-driver-monitor -f
```

## Service Configuration

### Default Configuration

- **Service Name:** nvidia-driver-monitor
- **User/Group:** nvidia-monitor
- **Install Directory:** /opt/nvidia-driver-monitor
- **Web Interface:** http://localhost:8080
- **Log File:** /var/log/nvidia-driver-monitor.log

### Security Features

The service includes several security hardening features:

- Runs as dedicated non-privileged user
- Network access restricted to localhost
- Protected system directories
- Resource limits (CPU, Memory)
- No new privileges
- Private temporary directory

### Environment Configuration

Edit `/opt/nvidia-driver-monitor/nvidia-monitor.env` to customize:

```bash
# Web server port
NVIDIA_MONITOR_ADDR=":8080"

# Logging level
NVIDIA_MONITOR_LOG_LEVEL="info"

# Update intervals
NVIDIA_MONITOR_UPDATE_INTERVAL="3600"  # 1 hour
```

## Monitoring and Logs

### View Live Logs
```bash
# Using service manager
./service-manager.sh logs

# Using journalctl
sudo journalctl -u nvidia-driver-monitor -f

# View recent logs
sudo journalctl -u nvidia-driver-monitor -n 100
```

### Log Files

- **System logs:** `journalctl -u nvidia-driver-monitor`
- **Application logs:** `/var/log/nvidia-driver-monitor.log`

## Troubleshooting

### Service Won't Start

1. Check service status:
   ```bash
   sudo systemctl status nvidia-driver-monitor
   ```

2. Check logs:
   ```bash
   sudo journalctl -u nvidia-driver-monitor -n 50
   ```

3. Verify binary permissions:
   ```bash
   ls -la /opt/nvidia-driver-monitor/
   ```

### Port Already in Use

1. Check what's using port 8080:
   ```bash
   sudo lsof -i :8080
   ```

2. Kill existing processes:
   ```bash
   make kill-web
   ```

### Permission Issues

1. Check service user:
   ```bash
   id nvidia-monitor
   ```

2. Verify file ownership:
   ```bash
   ls -la /opt/nvidia-driver-monitor/
   ```

### Configuration Issues

1. Validate configuration:
   ```bash
   sudo -u nvidia-monitor /opt/nvidia-driver-monitor/nvidia-web-server -help
   ```

2. Test manually:
   ```bash
   sudo -u nvidia-monitor /opt/nvidia-driver-monitor/nvidia-web-server -addr :8081
   ```

## Uninstallation

### Using Makefile
```bash
make uninstall-service
```

### Manual Uninstallation
```bash
sudo ./uninstall-service.sh
```

This will:
- Stop and disable the service
- Remove service files
- Remove installation directory
- Optionally remove the service user and group

## Files Created

### System Files
- `/etc/systemd/system/nvidia-driver-monitor.service` - Service definition
- `/var/log/nvidia-driver-monitor.log` - Log file

### Application Files
- `/opt/nvidia-driver-monitor/nvidia-web-server` - Binary
- `/opt/nvidia-driver-monitor/supportedReleases.json` - Configuration
- `/opt/nvidia-driver-monitor/nvidia-monitor.env` - Environment config

### User Account
- User: `nvidia-monitor`
- Group: `nvidia-monitor`
- Home: `/opt/nvidia-driver-monitor`

## API Endpoints

Once the service is running, the following endpoints are available:

- **Web Interface:** http://localhost:8080/
- **JSON API:** http://localhost:8080/api
- **Package Specific:** http://localhost:8080/package?package=nvidia-graphics-drivers-550

## Updates

To update the service with a new version:

1. Build the new version:
   ```bash
   make web
   ```

2. Stop the service:
   ```bash
   make service-stop
   ```

3. Update the binary:
   ```bash
   sudo cp nvidia-web-server /opt/nvidia-driver-monitor/
   sudo chown nvidia-monitor:nvidia-monitor /opt/nvidia-driver-monitor/nvidia-web-server
   ```

4. Start the service:
   ```bash
   make service-start
   ```

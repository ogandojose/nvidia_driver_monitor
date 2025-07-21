# HTTPS Support for NVIDIA Driver Monitor

The NVIDIA Driver Monitor web service now supports HTTPS with automatically generated self-signed certificates.

## Quick Start

### HTTP Mode (Default)
```bash
# Build and run with HTTP
make web
./nvidia-web-server

# Or use Makefile target
make run-web
```
Access: http://localhost:8080

### HTTPS Mode
```bash
# Build and run with HTTPS
make web
./nvidia-web-server -https

# Or use Makefile target
make run-web-https
```
Access: https://localhost:8080

## Command Line Options

- `-addr :PORT` - Server address (default: :8080)
- `-https` - Enable HTTPS with self-signed certificate
- `-cert FILE` - Custom certificate file (default: server.crt)
- `-key FILE` - Custom private key file (default: server.key)

### Examples
```bash
# HTTPS on custom port
./nvidia-web-server -https -addr :8443

# HTTPS with custom certificate
./nvidia-web-server -https -cert /path/to/cert.pem -key /path/to/key.pem

# HTTP on custom port
./nvidia-web-server -addr :9000
```

## Self-Signed Certificate

When HTTPS mode is enabled, the server will:

1. Check for existing certificate files (`server.crt` and `server.key`)
2. If not found, automatically generate a new self-signed certificate
3. The certificate is valid for 1 year and includes:
   - DNS name: `localhost`
   - IP addresses: `127.0.0.1` and `::1`
   - Organization: "NVIDIA Driver Monitor"

### Certificate Management

```bash
# Remove certificates (they'll be regenerated on next HTTPS start)
make clean-cert

# View certificate details
openssl x509 -in server.crt -text -noout

# Check certificate expiration
openssl x509 -in server.crt -noout -dates
```

## Browser Access

Since the certificate is self-signed, your browser will show a security warning. This is normal and expected for local development.

### Chrome/Edge
1. Click "Advanced"
2. Click "Proceed to localhost (unsafe)"

### Firefox
1. Click "Advanced"
2. Click "Accept the Risk and Continue"

### Command Line Testing
```bash
# Test with curl (ignore certificate)
curl -k https://localhost:8080

# Test certificate details
curl -vk https://localhost:8080 2>&1 | grep -A 10 "Server certificate"
```

## Systemd Service with HTTPS

Two service files are provided:

1. `nvidia-driver-monitor.service` - HTTP mode (port 8080)
2. `nvidia-driver-monitor-https.service` - HTTPS mode (port 8443)

```bash
# Install HTTPS service
sudo cp nvidia-driver-monitor-https.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable nvidia-driver-monitor-https
sudo systemctl start nvidia-driver-monitor-https
```

## Security Considerations

### Self-Signed Certificates
- Suitable for local development and internal networks
- Not recommended for public-facing production servers
- No certificate authority validation

### Production Deployment
For production environments, consider:
- Using certificates from a trusted CA (Let's Encrypt, etc.)
- Reverse proxy with proper SSL termination (nginx, Apache)
- Network security (firewalls, VPNs)

### TLS Configuration
The server uses secure TLS settings:
- Minimum TLS version 1.2
- Strong cipher suites (AES-256-GCM, ChaCha20-Poly1305)
- Secure curves (P-256, X25519)
- 15-second timeouts for read/write operations

## Makefile Targets

```bash
make run-web          # Run HTTP server
make run-web-https    # Run HTTPS server
make generate-cert    # Information about certificate generation
make clean-cert       # Remove certificate files
make clean-all        # Clean everything including certificates
```

## Troubleshooting

### "Certificate not found" Error
The server automatically generates certificates, so this shouldn't happen. If it does:
```bash
make clean-cert
./nvidia-web-server -https
```

### "Permission denied" on Port
For ports < 1024, you need elevated privileges:
```bash
sudo ./nvidia-web-server -https -addr :443
```

### Certificate Expired
Certificates are valid for 1 year. To regenerate:
```bash
make clean-cert
./nvidia-web-server -https
```

### Browser Won't Connect
- Ensure the server is running: `curl -k https://localhost:8080`
- Check firewall settings
- Try different ports: `-addr :8443`

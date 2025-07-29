# Configuration Guide

The NVIDIA Driver Monitor supports JSON-based configuration for customizing server behavior.

## Configuration File

By default, the application looks for `config.json` in the current directory. You can specify a different file using the `--config` flag.

### Default Configuration

```json
{
  "server": {
    "port": 8080,
    "https_port": 8443,
    "enable_https": false
  },
  "cache": {
    "refresh_interval": "15m",
    "enabled": true
  },
  "rate_limit": {
    "requests_per_minute": 60,
    "enabled": true
  }
}
```

## Configuration Options

### Server Configuration

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `port` | integer | `8080` | HTTP server port |
| `https_port` | integer | `8443` | HTTPS server port |
| `enable_https` | boolean | `false` | Enable HTTPS with self-signed certificates |

### Cache Configuration

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `refresh_interval` | string | `"15m"` | Data refresh interval (Go duration format) |
| `enabled` | boolean | `true` | Enable background data caching |

**Duration Format Examples:**
- `"5m"` - 5 minutes
- `"1h"` - 1 hour  
- `"30s"` - 30 seconds
- `"2h30m"` - 2 hours 30 minutes

### Rate Limiting Configuration

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `requests_per_minute` | integer | `60` | Maximum requests per minute per IP |
| `enabled` | boolean | `true` | Enable rate limiting |

## Command Line Flags

Command line flags override configuration file settings:

```bash
./nvidia-web-server [OPTIONS]

Options:
  -addr string
        Server address (default ":8080")
  -cert string
        Certificate file path (for HTTPS) (default "server.crt")
  -config string
        Configuration file path (default "config.json")
  -https
        Enable HTTPS with self-signed certificate
  -key string
        Private key file path (for HTTPS) (default "server.key")
  -rate-limit int
        Rate limit (requests per minute, 0 to use config)
  -templates string
        Templates directory path (default "templates")
```

## Examples

### Basic HTTP Server

```bash
# Use default configuration
./nvidia-web-server

# Custom port
./nvidia-web-server -addr :9090

# Custom rate limit
./nvidia-web-server -rate-limit 30
```

### HTTPS Server

```bash
# Enable HTTPS with default certificates
./nvidia-web-server -https

# HTTPS with custom certificates
./nvidia-web-server -https -cert /path/to/cert.pem -key /path/to/key.pem
```

### Custom Configuration

Create a custom `myconfig.json`:

```json
{
  "server": {
    "port": 9090,
    "enable_https": true
  },
  "cache": {
    "refresh_interval": "5m",
    "enabled": true
  },
  "rate_limit": {
    "requests_per_minute": 120,
    "enabled": true
  }
}
```

Run with custom configuration:

```bash
./nvidia-web-server -config myconfig.json
```

## Environment Considerations

### Development

```json
{
  "cache": {
    "refresh_interval": "2m",
    "enabled": true
  },
  "rate_limit": {
    "requests_per_minute": 120,
    "enabled": false
  }
}
```

### Production

```json
{
  "server": {
    "enable_https": true
  },
  "cache": {
    "refresh_interval": "15m",
    "enabled": true
  },
  "rate_limit": {
    "requests_per_minute": 60,
    "enabled": true
  }
}
```

## Template Directory

The `--templates` flag specifies where to find HTML template files:

```bash
./nvidia-web-server -templates /custom/templates/
```

Template directory should contain:
- `lrm_verifier.html` - LRM Verifier page template

## Configuration Validation

The application validates configuration on startup:

- Invalid duration formats fall back to defaults
- Missing configuration file uses built-in defaults
- Invalid JSON shows error and exits
- Invalid port numbers use defaults with warning

## Troubleshooting

### Configuration Not Loading

1. Check file path and permissions
2. Validate JSON syntax
3. Check log output for parsing errors

### Rate Limiting Issues

1. Verify `rate_limit.enabled` is `true`
2. Check `requests_per_minute` value
3. Use CLI flag to override: `-rate-limit N`

### Cache Not Refreshing

1. Verify `cache.enabled` is `true`
2. Check `refresh_interval` format
3. Monitor logs for refresh activity

# API Documentation

This document describes the REST API endpoints available in the NVIDIA Driver Monitor.

## Base URL

```
http://localhost:8080/api/
```

## Authentication

No authentication is required. Rate limiting is applied based on client IP address.

## Endpoints

### Health Check

**GET** `/api/health`

Returns the service health status.

**Response:**
```json
{
  "status": "healthy",
  "service": "nvidia-driver-monitor"
}
```

### LRM Data

**GET** `/api/lrm`

Returns Linux Restricted Modules (LRM) verification data with detailed driver information.

**Query Parameters:**

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `series` | string | Filter by Ubuntu series | `22.04` |
| `status` | string | Filter by support status | `SUPPORTED`, `LTS`, `DEV` |
| `routing` | string | Filter by routing | `ubuntu/4`, `pro/3` |
| `limit` | integer | Limit number of results | `10` |
| `offset` | integer | Offset for pagination | `20` |

### Available Routings

**GET** `/api/routings`

Returns all available routing values from the kernel-series.yaml data.

**Response:**
```json
{
  "routings": [
    "ubuntu/4",
    "pro/3", 
    "azure-6.8-ubuntu/4",
    "fips-pro/3",
    "realtime-pro/3"
  ],
  "count": 57
}
```

**Examples:**

```bash
# Get all LRM data
curl "http://localhost:8080/api/lrm"

# Get data for Ubuntu 22.04 only
curl "http://localhost:8080/api/lrm?series=22.04"

# Get supported kernels with pagination
curl "http://localhost:8080/api/lrm?status=SUPPORTED&limit=5&offset=0"

# Get kernels with specific routing
curl "http://localhost:8080/api/lrm?routing=ubuntu/4"

# Get available routings
curl "http://localhost:8080/api/routings"

# Combine filters
curl "http://localhost:8080/api/lrm?series=22.04&routing=pro/3&status=SUPPORTED"
```

**Response:**
```json
{
  "data": {
    "kernel_results": [
      {
        "Series": "22.04",
        "Codename": "jammy",
        "Source": "linux",
        "Routing": "ubuntu/4",
        "LRMPackages": ["linux-restricted-modules"],
        "HasLRM": true,
        "Supported": true,
        "Development": false,
        "LTS": true,
        "ESM": false,
        "LatestLRMVersion": "5.15.0-151.161 (Security)",
        "SourceVersion": "5.15.0-151.161 (Security)",
        "NvidiaDriverStatuses": [
          {
            "DriverName": "nvidia-graphics-drivers-535",
            "DSCVersion": "535.247.01-0ubuntu0.22.04.1",
            "DKMSVersion": "535.247.01-0ubuntu0.22.04.1",
            "Status": "✅ Up to date",
            "FullString": "nvidia-graphics-drivers-535=535.247.01-0ubuntu0.22.04.1"
          }
        ]
      }
    ],
    "total_kernels": 120,
    "supported_lrm": 38,
    "last_updated": "2025-07-29T18:06:55.114829459+02:00",
    "is_initialized": true
  },
  "meta": {
    "total": 38,
    "filtered": 1
  }
}
```

**Response Fields:**

- `data.kernel_results`: Array of kernel LRM results
- `data.total_kernels`: Total number of kernels in the system
- `data.supported_lrm`: Number of kernels with LRM support
- `data.last_updated`: Timestamp of last data refresh
- `data.is_initialized`: Whether the data has been initialized
- `meta.total`: Total number of results before filtering
- `meta.filtered`: Number of results after applying filters

**Individual Driver Status:**

Each kernel result contains `NvidiaDriverStatuses` with individual driver information:

- `DriverName`: Full driver package name
- `DSCVersion`: Version from Debian Source Control files
- `DKMSVersion`: Version from DKMS/Updates-Security
- `Status`: Update status with emoji indicators:
  - `✅ Up to date`: DSC and DKMS versions match
  - `🔄 Update available`: DKMS version is newer than DSC
  - `⚠️ Unknown`: DKMS version not available or comparison failed
- `FullString`: Complete driver string with version

## Rate Limiting

API endpoints are subject to rate limiting:

- Default: 60 requests per minute per IP address
- Configurable via CLI flag: `--rate-limit N`
- Rate limit exceeded returns HTTP 429

## Error Handling

All endpoints return appropriate HTTP status codes:

- `200 OK`: Success
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error

Error responses include JSON with error description:

```json
{
  "error": "Rate limit exceeded"
}
```

## CORS Support

The API includes CORS headers for browser-based requests:

- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type`

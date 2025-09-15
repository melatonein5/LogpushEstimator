# API Reference Documentation

## Table of Contents
1. [Overview](#overview)
2. [Authentication](#authentication)
3. [Common Patterns](#common-patterns)
4. [Ingestion API](#ingestion-api)
5. [Statistics API](#statistics-api)
6. [Logs API](#logs-api)
7. [Charts API](#charts-api)
8. [Health Check API](#health-check-api)
9. [Error Codes](#error-codes)
10. [SDK Examples](#sdk-examples)

## Overview

The LogpushEstimator API provides RESTful endpoints for log data ingestion and retrieval. The API is split across two servers:

- **Ingestion Server** (Port 8080): Handles log data ingestion and health checks
- **GUI Server** (Port 8081): Provides data retrieval APIs and web dashboard

### Base URLs

```
Ingestion Server: http://localhost:8080
GUI Server:       http://localhost:8081
```

### API Versioning

Current API version is v1 (implicit). Future versions will include explicit versioning in the URL path.

### Content Types

- **Request Content-Type**: `application/json` (for POST requests with JSON data)
- **Response Content-Type**: `application/json` (for all API responses)
- **Raw Data**: `text/plain`, `application/octet-stream` (for log ingestion)

## Authentication

**Current Implementation**: No authentication required.

**Future Considerations**:
- API key authentication for ingestion endpoints
- JWT tokens for dashboard access
- IP whitelisting for production deployments

## Common Patterns

### Response Format

All API responses follow a consistent structure:

#### Success Response
```json
{
  "success": true,
  "data": {
    // Endpoint-specific data
  }
}
```

#### Error Response
```json
{
  "success": false,
  "error": "Error description message"
}
```

### HTTP Status Codes

| Status Code | Meaning | Usage |
|-------------|---------|-------|
| 200 | OK | Successful request |
| 400 | Bad Request | Invalid request parameters or body |
| 404 | Not Found | Endpoint or resource not found |
| 405 | Method Not Allowed | HTTP method not supported |
| 500 | Internal Server Error | Server-side error |

### Query Parameters

#### Common Parameters

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `limit` | integer | Maximum number of records to return | `?limit=100` |
| `start` | ISO 8601 datetime | Start time for time range queries | `?start=2025-09-15T00:00:00Z` |
| `end` | ISO 8601 datetime | End time for time range queries | `?end=2025-09-15T23:59:59Z` |
| `hours` | integer | Number of hours to look back | `?hours=24` |

### CORS Headers

All API responses include CORS headers for development:

```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Content-Type
```

## Ingestion API

### POST /ingest

Accepts log data and stores the payload size with timestamp.

#### Request

**URL**: `http://localhost:8080/ingest`  
**Method**: `POST`  
**Content-Type**: Any (raw log data)

**Headers**:
```
Content-Type: application/json
Content-Length: [size in bytes]
```

**Body**: Raw log data (any format)

#### Examples

**Example 1: JSON Log Data**
```bash
curl -X POST http://localhost:8080/ingest \
  -H "Content-Type: application/json" \
  -d '{"timestamp":"2025-09-15T14:30:45Z","level":"info","message":"User login"}'
```

**Example 2: Plain Text Log**
```bash
curl -X POST http://localhost:8080/ingest \
  -H "Content-Type: text/plain" \
  -d "2025-09-15 14:30:45 INFO User login successful"
```

**Example 3: Large Binary Data**
```bash
curl -X POST http://localhost:8080/ingest \
  -H "Content-Type: application/octet-stream" \
  --data-binary @large_log_file.bin
```

#### Response

**Success Response (200)**:
```
HTTP/1.1 200 OK
Content-Type: text/plain

OK
```

**Error Responses**:

**Method Not Allowed (405)**:
```
HTTP/1.1 405 Method Not Allowed
Content-Type: text/plain

Method not allowed
```

**Bad Request (400)**:
```
HTTP/1.1 400 Bad Request
Content-Type: text/plain

Request body cannot be empty
```

**Internal Server Error (500)**:
```
HTTP/1.1 500 Internal Server Error
Content-Type: text/plain

Database error
```

#### Implementation Notes

- The endpoint measures the Content-Length of the request body
- Actual log content is not stored, only the size and timestamp
- Concurrent requests are supported
- Empty requests are rejected with 400 status

## Statistics API

### GET /api/stats/summary

Returns comprehensive summary statistics for all stored log data.

#### Request

**URL**: `http://localhost:8081/api/stats/summary`  
**Method**: `GET`  
**Parameters**: None

#### Example

```bash
curl -X GET http://localhost:8081/api/stats/summary
```

#### Response

**Success Response (200)**:
```json
{
  "success": true,
  "data": {
    "total_records": 15420,
    "total_size": 2048576000,
    "average_size": 132874.2,
    "min_size": 1024,
    "max_size": 5242880,
    "last_updated": "2025-09-15T14:30:45Z"
  }
}
```

**Response Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `total_records` | integer | Total number of log records |
| `total_size` | integer | Total size of all logs in bytes |
| `average_size` | float | Average log size in bytes |
| `min_size` | integer | Smallest log size in bytes |
| `max_size` | integer | Largest log size in bytes |
| `last_updated` | string | ISO 8601 timestamp of most recent log |

**Error Response (500)**:
```json
{
  "success": false,
  "error": "Failed to fetch statistics"
}
```

## Logs API

### GET /api/logs/recent

Returns the most recent log entries, ordered by timestamp descending.

#### Request

**URL**: `http://localhost:8081/api/logs/recent`  
**Method**: `GET`

**Query Parameters**:
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `limit` | integer | No | 100 | Maximum number of records to return |

#### Examples

**Default Request (100 records)**:
```bash
curl -X GET http://localhost:8081/api/logs/recent
```

**Limited Request (10 records)**:
```bash
curl -X GET "http://localhost:8081/api/logs/recent?limit=10"
```

#### Response

**Success Response (200)**:
```json
{
  "success": true,
  "data": [
    {
      "id": 15420,
      "timestamp": "2025-09-15T14:30:45Z",
      "filesize": 2048
    },
    {
      "id": 15419,
      "timestamp": "2025-09-15T14:30:30Z",
      "filesize": 4096
    }
  ]
}
```

**Response Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `id` | integer | Unique record identifier |
| `timestamp` | string | ISO 8601 timestamp when log was recorded |
| `filesize` | integer | Size of log data in bytes |

### GET /api/logs/time-range

Returns log entries within a specified time range.

#### Request

**URL**: `http://localhost:8081/api/logs/time-range`  
**Method**: `GET`

**Query Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `start` | ISO 8601 datetime | Yes | Start time (inclusive) |
| `end` | ISO 8601 datetime | Yes | End time (exclusive) |

#### Examples

**Last 24 Hours**:
```bash
curl -X GET "http://localhost:8081/api/logs/time-range?start=2025-09-14T14:30:00Z&end=2025-09-15T14:30:00Z"
```

**Specific Day**:
```bash
curl -X GET "http://localhost:8081/api/logs/time-range?start=2025-09-15T00:00:00Z&end=2025-09-16T00:00:00Z"
```

#### Response

**Success Response (200)**:
```json
{
  "success": true,
  "data": [
    {
      "id": 15400,
      "timestamp": "2025-09-15T12:00:00Z",
      "filesize": 1024
    },
    {
      "id": 15401,
      "timestamp": "2025-09-15T12:05:00Z",
      "filesize": 2048
    }
  ]
}
```

**Error Responses**:

**Missing Parameters (400)**:
```json
{
  "success": false,
  "error": "Missing required parameter: start"
}
```

**Invalid Date Format (400)**:
```json
{
  "success": false,
  "error": "Invalid start time format"
}
```

## Charts API

### GET /api/charts/timeseries

Returns time-series data aggregated by hour for chart visualization.

#### Request

**URL**: `http://localhost:8081/api/charts/timeseries`  
**Method**: `GET`

**Query Parameters**:
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `hours` | integer | No | 24 | Number of hours to look back |

#### Examples

**Last 24 Hours (Default)**:
```bash
curl -X GET http://localhost:8081/api/charts/timeseries
```

**Last 7 Days**:
```bash
curl -X GET "http://localhost:8081/api/charts/timeseries?hours=168"
```

#### Response

**Success Response (200)**:
```json
{
  "success": true,
  "data": [
    {
      "timestamp": "2025-09-15T14:00:00Z",
      "count": 45,
      "total_size": 2048000
    },
    {
      "timestamp": "2025-09-15T13:00:00Z",
      "count": 38,
      "total_size": 1843200
    }
  ]
}
```

**Response Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | string | Hour bucket timestamp (ISO 8601) |
| `count` | integer | Number of log records in this hour |
| `total_size` | integer | Total size of logs in this hour (bytes) |

### GET /api/charts/size-breakdown

Returns size distribution analysis for chart visualization.

#### Request

**URL**: `http://localhost:8081/api/charts/size-breakdown`  
**Method**: `GET`  
**Parameters**: None

#### Example

```bash
curl -X GET http://localhost:8081/api/charts/size-breakdown
```

#### Response

**Success Response (200)**:
```json
{
  "success": true,
  "data": [
    {
      "range": "Small (<1KB)",
      "count": 1250,
      "percentage": 8.1
    },
    {
      "range": "Medium (1-10KB)",
      "count": 8500,
      "percentage": 55.1
    },
    {
      "range": "Large (10-100KB)",
      "count": 4800,
      "percentage": 31.1
    },
    {
      "range": "Very Large (>100KB)",
      "count": 870,
      "percentage": 5.6
    }
  ]
}
```

**Response Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `range` | string | Size range description |
| `count` | integer | Number of records in this size range |
| `percentage` | float | Percentage of total records |

## Health Check API

### GET /health

Returns the health status of the ingestion service.

#### Request

**URL**: `http://localhost:8080/health`  
**Method**: `GET`  
**Parameters**: None

#### Example

```bash
curl -X GET http://localhost:8080/health
```

#### Response

**Success Response (200)**:
```json
{
  "status": "ok",
  "service": "LogpushEstimator"
}
```

**Response Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `status` | string | Service health status ("ok" or "error") |
| `service` | string | Service identifier |

## Error Codes

### HTTP Status Codes

| Code | Name | Description | Common Causes |
|------|------|-------------|---------------|
| 200 | OK | Request successful | - |
| 400 | Bad Request | Invalid request | Missing parameters, invalid format |
| 404 | Not Found | Resource not found | Invalid endpoint, missing resource |
| 405 | Method Not Allowed | HTTP method not supported | Wrong HTTP method for endpoint |
| 500 | Internal Server Error | Server-side error | Database errors, system failures |

### Application Error Messages

| Error Message | Cause | Solution |
|---------------|-------|----------|
| "Method not allowed" | Wrong HTTP method | Use correct method (GET/POST) |
| "Request body cannot be empty" | Empty POST body | Include data in request body |
| "Failed to read request body" | Body parsing error | Check request format |
| "Missing required parameter: start" | Missing query parameter | Include required parameters |
| "Invalid start time format" | Bad datetime format | Use ISO 8601 format |
| "Failed to fetch statistics" | Database error | Check database connectivity |
| "Failed to fetch logs" | Database error | Check database connectivity |

## SDK Examples

### JavaScript/Node.js

```javascript
class LogpushEstimatorClient {
    constructor(ingestionUrl = 'http://localhost:8080', apiUrl = 'http://localhost:8081') {
        this.ingestionUrl = ingestionUrl;
        this.apiUrl = apiUrl;
    }

    // Ingest log data
    async ingestLog(logData) {
        const response = await fetch(`${this.ingestionUrl}/ingest`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(logData)
        });
        
        if (!response.ok) {
            throw new Error(`Ingestion failed: ${response.statusText}`);
        }
        
        return await response.text();
    }

    // Get summary statistics
    async getSummary() {
        const response = await fetch(`${this.apiUrl}/api/stats/summary`);
        const data = await response.json();
        
        if (!data.success) {
            throw new Error(data.error);
        }
        
        return data.data;
    }

    // Get recent logs
    async getRecentLogs(limit = 100) {
        const response = await fetch(`${this.apiUrl}/api/logs/recent?limit=${limit}`);
        const data = await response.json();
        
        if (!data.success) {
            throw new Error(data.error);
        }
        
        return data.data;
    }

    // Get logs by time range
    async getLogsByTimeRange(startTime, endTime) {
        const params = new URLSearchParams({
            start: startTime.toISOString(),
            end: endTime.toISOString()
        });
        
        const response = await fetch(`${this.apiUrl}/api/logs/time-range?${params}`);
        const data = await response.json();
        
        if (!data.success) {
            throw new Error(data.error);
        }
        
        return data.data;
    }
}

// Usage example
const client = new LogpushEstimatorClient();

// Ingest log
await client.ingestLog({
    timestamp: new Date().toISOString(),
    level: 'info',
    message: 'User action completed'
});

// Get statistics
const stats = await client.getSummary();
console.log(`Total records: ${stats.total_records}`);

// Get recent logs
const recentLogs = await client.getRecentLogs(10);
console.log(`Latest log size: ${recentLogs[0].filesize} bytes`);
```

### Python

```python
import requests
import json
from datetime import datetime, timedelta
from typing import Dict, List, Optional

class LogpushEstimatorClient:
    def __init__(self, ingestion_url: str = "http://localhost:8080", 
                 api_url: str = "http://localhost:8081"):
        self.ingestion_url = ingestion_url
        self.api_url = api_url

    def ingest_log(self, log_data: str) -> bool:
        """Ingest log data."""
        response = requests.post(
            f"{self.ingestion_url}/ingest",
            data=log_data,
            headers={"Content-Type": "application/json"}
        )
        response.raise_for_status()
        return response.text == "OK"

    def get_summary(self) -> Dict:
        """Get summary statistics."""
        response = requests.get(f"{self.api_url}/api/stats/summary")
        response.raise_for_status()
        data = response.json()
        
        if not data["success"]:
            raise Exception(data["error"])
        
        return data["data"]

    def get_recent_logs(self, limit: int = 100) -> List[Dict]:
        """Get recent log entries."""
        response = requests.get(
            f"{self.api_url}/api/logs/recent",
            params={"limit": limit}
        )
        response.raise_for_status()
        data = response.json()
        
        if not data["success"]:
            raise Exception(data["error"])
        
        return data["data"]

    def get_logs_by_time_range(self, start_time: datetime, 
                              end_time: datetime) -> List[Dict]:
        """Get logs within time range."""
        params = {
            "start": start_time.isoformat() + "Z",
            "end": end_time.isoformat() + "Z"
        }
        
        response = requests.get(
            f"{self.api_url}/api/logs/time-range",
            params=params
        )
        response.raise_for_status()
        data = response.json()
        
        if not data["success"]:
            raise Exception(data["error"])
        
        return data["data"]

    def get_timeseries_data(self, hours: int = 24) -> List[Dict]:
        """Get time series chart data."""
        response = requests.get(
            f"{self.api_url}/api/charts/timeseries",
            params={"hours": hours}
        )
        response.raise_for_status()
        data = response.json()
        
        if not data["success"]:
            raise Exception(data["error"])
        
        return data["data"]

# Usage example
client = LogpushEstimatorClient()

# Ingest log
log_entry = json.dumps({
    "timestamp": datetime.now().isoformat(),
    "level": "info",
    "message": "Processing completed"
})
client.ingest_log(log_entry)

# Get statistics
stats = client.get_summary()
print(f"Total records: {stats['total_records']}")
print(f"Average size: {stats['average_size']:.2f} bytes")

# Get logs from last hour
end_time = datetime.now()
start_time = end_time - timedelta(hours=1)
recent_logs = client.get_logs_by_time_range(start_time, end_time)
print(f"Logs in last hour: {len(recent_logs)}")
```

### curl Examples

```bash
#!/bin/bash
# LogpushEstimator API Examples

# Configuration
INGESTION_URL="http://localhost:8080"
API_URL="http://localhost:8081"

# Ingest a log entry
echo "Ingesting log data..."
curl -X POST ${INGESTION_URL}/ingest \
  -H "Content-Type: application/json" \
  -d '{"level":"info","message":"API test log","timestamp":"2025-09-15T14:30:45Z"}'

# Get health status
echo -e "\n\nChecking health..."
curl -X GET ${INGESTION_URL}/health

# Get summary statistics
echo -e "\n\nGetting summary statistics..."
curl -X GET ${API_URL}/api/stats/summary | jq '.'

# Get recent logs
echo -e "\n\nGetting recent logs..."
curl -X GET "${API_URL}/api/logs/recent?limit=5" | jq '.'

# Get logs from specific time range
echo -e "\n\nGetting logs from time range..."
START_TIME="2025-09-15T00:00:00Z"
END_TIME="2025-09-15T23:59:59Z"
curl -X GET "${API_URL}/api/logs/time-range?start=${START_TIME}&end=${END_TIME}" | jq '.'

# Get time series data
echo -e "\n\nGetting time series data..."
curl -X GET "${API_URL}/api/charts/timeseries?hours=24" | jq '.'

# Get size breakdown
echo -e "\n\nGetting size breakdown..."
curl -X GET ${API_URL}/api/charts/size-breakdown | jq '.'
```

---

*This API reference is current as of September 15, 2025. For the most up-to-date API information, always refer to the application's built-in documentation or source code.*
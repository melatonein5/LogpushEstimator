# LogpushEstimator

[![Go Version](https://img.shields.io/badge/go-1.21%2B-blue.svg)](https://golang.org/doc/install)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

LogpushEstimator is a Cloudflare log ingestion monitoring tool designed to help analyze and estimate data usage from Cloudflare's Logpush service. It tracks log file sizes over time and provides both an ingestion endpoint for receiving log data and a web-based dashboard for visualizing usage patterns.

## Features

- **Real-time Log Ingestion**: HTTP endpoint for receiving Cloudflare log data
- **Web Dashboard**: Interactive interface for visualizing log analytics
- **REST API**: Comprehensive API for accessing stored data and statistics
- **SQLite Storage**: Lightweight, serverless database for data persistence
- **Time-Series Analytics**: Track log volume trends over time
- **Size Distribution Analysis**: Understand log size patterns and distributions

## Installation

### Prerequisites

- Go 1.21 or later
- CGO enabled (required for SQLite)

### Install from Source

```bash
git clone https://github.com/melatonein5/LogpushEstimator.git
cd LogpushEstimator
go build -o logpush-estimator .
```

### Run Directly

```bash
go run main.go
```

## Quick Start

1. **Start the application**:
   ```bash
   ./logpush-estimator
   ```

2. **Send log data to the ingestion endpoint**:
   ```bash
   curl -X POST http://localhost:8080/ingest \
        -H "Content-Type: application/json" \
        -d '{"log": "sample log data"}'
   ```

3. **Access the web dashboard**:
   Open your browser to `http://localhost:8081`

## Architecture

LogpushEstimator consists of two main HTTP servers:

### Ingestion Server (Port 8080)
- **POST /ingest**: Accept log data for size tracking
- **GET /health**: Health check endpoint

### GUI Server (Port 8081)
- **GET /**: Main dashboard interface
- **GET /api/stats/summary**: Summary statistics
- **GET /api/logs/recent**: Recent log entries
- **GET /api/logs/time-range**: Time-filtered log data
- **GET /api/charts/time-series**: Time series chart data
- **GET /api/charts/size-breakdown**: Size breakdown chart data
- **GET /static/***: Static assets (CSS, JS, images)

## API Reference

### Summary Statistics

```bash
curl http://localhost:8081/api/stats/summary
```

Response:
```json
{
  "success": true,
  "data": {
    "total_records": 1500,
    "total_size": 157286400,
    "average_size": 104857.6,
    "min_size": 1024,
    "max_size": 2097152,
    "last_updated": "2025-09-15T21:30:45Z"
  }
}
```

### Recent Logs

```bash
curl "http://localhost:8081/api/logs/recent?limit=10"
```

### Time Range Query

```bash
curl "http://localhost:8081/api/logs/time-range?start=2025-09-15T00:00:00Z&end=2025-09-15T23:59:59Z"
```

## Configuration

The application uses the following default configuration:

- **Ingestion Port**: 8080
- **GUI Port**: 8081
- **Database Path**: `logpush.db` (SQLite)
- **Log Level**: Info

These can be modified by editing the source code variables in `main.go`.

## Development

### Project Structure

```
LogpushEstimator/
├── main.go                          # Main application entry point
├── src/
│   ├── database/
│   │   ├── sqlite_controller.go     # Database operations
│   │   └── sqlite_controller_test.go
│   └── gui/
│       ├── handlers/
│       │   ├── api.go              # REST API handlers
│       │   ├── dashboard.go        # Web interface handlers
│       │   └── handlers_test.go
│       ├── static/                 # Static web assets
│       └── templates/              # HTML templates
├── go.mod                          # Go module definition
├── go.sum                          # Go module checksums
└── README.md
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...
```

### Building for Production

```bash
# Build for current platform
go build -ldflags="-s -w" -o logpush-estimator .

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o logpush-estimator-linux .
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with Go's standard library and minimal dependencies
- Uses SQLite for lightweight data persistence
- Designed specifically for Cloudflare Logpush analytics
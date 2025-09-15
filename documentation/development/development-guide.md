# Development Guide

## Table of Contents
1. [Getting Started](#getting-started)
2. [Development Environment](#development-environment)
3. [Project Structure](#project-structure)
4. [Coding Standards](#coding-standards)
5. [Testing Strategy](#testing-strategy)
6. [Debugging](#debugging)
7. [Contributing](#contributing)
8. [Build and Release](#build-and-release)
9. [Performance Optimization](#performance-optimization)
10. [Troubleshooting](#troubleshooting)

## Getting Started

### Quick Setup

1. **Prerequisites**:
   ```bash
   # Verify Go installation
   go version  # Should be 1.21 or later
   
   # Install development tools
   go install golang.org/x/tools/cmd/goimports@latest
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

2. **Clone and Setup**:
   ```bash
   git clone <repository-url>
   cd LogpushEstimator
   go mod download
   go mod tidy
   ```

3. **Verify Setup**:
   ```bash
   # Run tests
   go test ./...
   
   # Run application
   go run main.go
   
   # Verify services
   curl http://localhost:8080/health
   curl http://localhost:8081/api/stats/summary
   ```

### IDE Configuration

#### VS Code

Recommended extensions:
- Go (official Google extension)
- GitLens
- REST Client
- SQLite Viewer

`.vscode/settings.json`:
```json
{
    "go.testFlags": ["-v"],
    "go.coverageDecorator": {
        "type": "gutter",
        "coveredHighlightColor": "rgba(64,128,128,0.5)",
        "uncoveredHighlightColor": "rgba(128,64,64,0.25)"
    },
    "go.lintTool": "golangci-lint",
    "go.lintFlags": ["--fast"],
    "files.exclude": {
        "**/.git": true,
        "**/node_modules": true,
        "build/": true,
        "*.db": true
    }
}
```

`.vscode/launch.json`:
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Application",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/main.go",
            "env": {
                "DB_PATH": "./data/dev-logs.db",
                "LOG_LEVEL": "debug"
            },
            "args": []
        },
        {
            "name": "Debug Tests",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "${workspaceFolder}",
            "args": ["-test.v"]
        }
    ]
}
```

#### GoLand/IntelliJ

1. Import project as Go module
2. Enable Go modules integration
3. Configure run configurations for main.go
4. Set up test configurations for package testing

## Development Environment

### Environment Variables

Create `.env.development`:
```bash
# Development configuration
DB_PATH=./data/dev-logs.db
LOG_LEVEL=debug
INGESTION_PORT=8080
GUI_PORT=8081
STATIC_DIR=./src/gui/static
TEMPLATES_DIR=./src/gui/templates

# Development flags
DEBUG_MODE=true
ENABLE_CORS=true
```

Load environment:
```bash
export $(cat .env.development | xargs)
go run main.go
```

### Database Management

#### Development Database

```bash
# Create development database directory
mkdir -p data/

# Initialize development database (automatic on first run)
go run main.go
```

#### Database Inspection

```bash
# Connect to SQLite database
sqlite3 data/dev-logs.db

# Useful SQLite commands
.tables              # List tables
.schema logs         # Show table schema
.quit               # Exit
```

#### Sample Data Generation

Create `scripts/generate-test-data.go`:
```go
package main

import (
    "bytes"
    "fmt"
    "math/rand"
    "net/http"
    "time"
)

func main() {
    client := &http.Client{Timeout: 5 * time.Second}
    
    // Generate 1000 sample log entries
    for i := 0; i < 1000; i++ {
        size := rand.Intn(10000) + 100  // 100-10100 bytes
        data := make([]byte, size)
        
        // Fill with random data
        for j := range data {
            data[j] = byte(rand.Intn(26) + 97)  // random lowercase letters
        }
        
        // Send to ingestion endpoint
        resp, err := client.Post("http://localhost:8080/ingest", 
            "application/octet-stream", bytes.NewReader(data))
        if err != nil {
            fmt.Printf("Error sending request %d: %v\n", i, err)
            continue
        }
        resp.Body.Close()
        
        // Random delay between requests
        time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
        
        if (i+1)%100 == 0 {
            fmt.Printf("Generated %d sample entries\n", i+1)
        }
    }
    
    fmt.Println("Sample data generation complete!")
}
```

Run sample data generation:
```bash
go run scripts/generate-test-data.go
```

### Hot Reload Development

Install air for hot reloading:
```bash
go install github.com/cosmtrek/air@latest
```

Create `.air.toml`:
```toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ."
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata", "build", "data"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html", "css", "js"]
  kill_delay = "0s"
  log = "build-errors.log"
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  time = false

[misc]
  clean_on_exit = false
```

Start development with hot reload:
```bash
air
```

## Project Structure

### Directory Layout

```
LogpushEstimator/
├── main.go                    # Application entry point
├── go.mod                     # Go module definition
├── go.sum                     # Go module checksums
├── README.md                  # Project overview
├── .env.development          # Development environment variables
├── .air.toml                 # Hot reload configuration
├── .gitignore                # Git ignore rules
├── .golangci.yml             # Linting configuration
├── Dockerfile                # Container build instructions
├── docker-compose.yml        # Multi-container setup
├── Makefile                  # Build automation
├── src/                      # Source code directory
│   ├── database/             # Database layer
│   │   ├── sqlite_controller.go     # SQLite implementation
│   │   └── sqlite_controller_test.go # Database tests
│   ├── gui/                  # Web interface
│   │   ├── handlers/         # HTTP handlers
│   │   │   ├── api.go        # API endpoints
│   │   │   ├── api_test.go   # API tests
│   │   │   ├── dashboard.go  # Dashboard handler
│   │   │   └── dashboard_test.go # Dashboard tests
│   │   ├── static/           # Static web assets
│   │   │   ├── css/          # Stylesheets
│   │   │   ├── js/           # JavaScript files
│   │   │   └── images/       # Image assets
│   │   └── templates/        # HTML templates
│   │       ├── dashboard.html # Main dashboard template
│   │       └── base.html     # Base template
│   └── cloudflare/           # External integrations
│       └── cloudflare-client.go # Cloudflare API client
├── scripts/                  # Development and build scripts
│   ├── build.sh              # Build automation
│   ├── test.sh               # Test automation
│   ├── generate-test-data.go # Test data generator
│   └── deploy.sh             # Deployment automation
├── data/                     # Development database files
│   └── dev-logs.db           # SQLite development database
├── build/                    # Build artifacts
│   ├── linux/                # Linux builds
│   ├── windows/              # Windows builds
│   └── macos/                # macOS builds
├── documentation/            # Project documentation
│   ├── README.md             # Documentation index
│   ├── api/                  # API documentation
│   ├── design/               # Design documents
│   ├── deployment/           # Deployment guides
│   └── development/          # Development guides
└── tests/                    # Integration tests
    ├── integration_test.go   # Integration test suite
    └── fixtures/             # Test fixtures and data
```

### Package Organization

#### main.go
- Application bootstrap and configuration
- Server setup and routing
- Graceful shutdown handling

#### src/database/
- Database abstraction layer
- SQLite implementation
- Database migration logic
- Connection pooling and management

#### src/gui/handlers/
- HTTP request handlers
- API endpoint implementations
- Template rendering
- Static file serving

#### src/gui/static/ and templates/
- Web assets (CSS, JavaScript, images)
- HTML templates using Go's html/template
- Responsive dashboard components

#### scripts/
- Build automation scripts
- Testing utilities
- Deployment helpers
- Development tools

## Coding Standards

### Go Style Guide

Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

#### Naming Conventions

```go
// Package names: lowercase, no underscores
package database

// Function names: CamelCase for exported, camelCase for unexported
func NewSQLiteController() *SQLiteController { }
func createTable() error { }

// Constant names: CamelCase or ALL_CAPS for package-level
const DefaultPort = 8080
const MAX_RETRY_ATTEMPTS = 3

// Variable names: descriptive, avoid abbreviations
var connectionTimeout time.Duration
var httpClient *http.Client

// Interface names: -er suffix when possible
type Logger interface {
    Log(message string)
}
```

#### Error Handling

```go
// Always handle errors explicitly
result, err := database.Query(sql)
if err != nil {
    return fmt.Errorf("failed to execute query: %w", err)
}

// Use custom error types for specific cases
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}
```

#### Documentation

```go
// Package-level documentation
// Package database provides SQLite database operations for LogpushEstimator.
// It implements connection pooling, query optimization, and data persistence
// for log size measurements and statistics.
package database

// Function documentation: what it does, parameters, return values, errors
// NewSQLiteController creates a new SQLite database controller with the specified
// database file path. It initializes the database schema if it doesn't exist.
//
// Parameters:
//   - dbPath: absolute path to the SQLite database file
//
// Returns:
//   - *SQLiteController: initialized database controller
//   - error: initialization error, if any
//
// Example:
//   controller, err := NewSQLiteController("/var/lib/app/logs.db")
//   if err != nil {
//       log.Fatal(err)
//   }
//   defer controller.Close()
func NewSQLiteController(dbPath string) (*SQLiteController, error) {
    // Implementation...
}
```

### Code Formatting

#### Linting Configuration

Create `.golangci.yml`:
```yaml
run:
  timeout: 5m
  go: '1.21'

issues:
  max-issues-per-linter: 0
  max-same-issues: 0

linters:
  enable:
    - errcheck      # Check for unchecked errors
    - goimports     # Check imports formatting
    - golint        # Google's linter
    - govet         # Examine Go source code and reports suspicious constructs
    - ineffassign   # Detect unused variable assignments
    - misspell      # Find misspelled words
    - staticcheck   # Advanced Go linter
    - unused        # Check for unused constants, variables, functions and types
    - gosimple      # Simplify Go code
    - goconst       # Find repeated strings that could be constants
    - gocyclo       # Check cyclomatic complexity
    - dupl          # Check for duplicate code

linters-settings:
  gocyclo:
    min-complexity: 15
  goconst:
    min-len: 3
    min-occurrences: 3
  misspell:
    locale: US
```

#### Formatting Commands

```bash
# Format code
go fmt ./...

# Organize imports
goimports -w .

# Run linter
golangci-lint run

# Pre-commit check
make lint
```

### Makefile

Create a `Makefile` for common tasks:
```makefile
.PHONY: build test lint clean run dev

# Build the application
build:
	go build -o bin/logpush-estimator .

# Run tests with coverage
test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run linting
lint:
	golangci-lint run
	go fmt ./...
	goimports -w .

# Clean build artifacts
clean:
	rm -rf bin/ build/ tmp/ coverage.out coverage.html

# Run the application in development mode
run:
	export $$(cat .env.development | xargs) && go run main.go

# Development mode with hot reload
dev:
	export $$(cat .env.development | xargs) && air

# Install development dependencies
deps:
	go mod download
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/cosmtrek/air@latest

# Generate test data
sample-data:
	go run scripts/generate-test-data.go

# Database operations
db-reset:
	rm -f data/dev-logs.db
	$(MAKE) run &
	sleep 2
	pkill -f "go run main.go"

# Build for all platforms
build-all:
	./scripts/build.sh

# Run integration tests
integration-test:
	go test -tags=integration ./tests/...
```

## Testing Strategy

### Test Types

#### Unit Tests

Test individual functions and methods:

```go
// src/database/sqlite_controller_test.go
func TestSQLiteController_InsertLogSize(t *testing.T) {
    // Setup
    db, err := NewSQLiteController(":memory:")
    require.NoError(t, err)
    defer db.Close()

    // Test data
    testSize := int64(1024)
    
    // Execute
    err = db.InsertLogSize(testSize)
    
    // Assert
    assert.NoError(t, err)
    
    // Verify insertion
    logs, err := db.GetAll()
    assert.NoError(t, err)
    assert.Len(t, logs, 1)
    assert.Equal(t, testSize, logs[0].FileSize)
}

func TestSQLiteController_QueryByTimeRange(t *testing.T) {
    // Test time range queries with edge cases
    db, err := NewSQLiteController(":memory:")
    require.NoError(t, err)
    defer db.Close()

    // Insert test data with specific timestamps
    testData := []struct {
        size      int64
        timestamp time.Time
    }{
        {1000, time.Date(2025, 9, 15, 10, 0, 0, 0, time.UTC)},
        {2000, time.Date(2025, 9, 15, 11, 0, 0, 0, time.UTC)},
        {3000, time.Date(2025, 9, 15, 12, 0, 0, 0, time.UTC)},
    }

    for _, td := range testData {
        err := db.insertLogSizeWithTimestamp(td.size, td.timestamp)
        require.NoError(t, err)
    }

    // Test query
    start := time.Date(2025, 9, 15, 10, 30, 0, 0, time.UTC)
    end := time.Date(2025, 9, 15, 11, 30, 0, 0, time.UTC)
    
    logs, err := db.QueryByTimeRange(start, end)
    assert.NoError(t, err)
    assert.Len(t, logs, 1)
    assert.Equal(t, int64(2000), logs[0].FileSize)
}
```

#### Integration Tests

Test component interactions:

```go
// tests/integration_test.go
// +build integration

func TestIngestionToDatabase(t *testing.T) {
    // Setup test database
    testDB := "./test_data/integration_test.db"
    os.Remove(testDB) // Clean start
    
    // Start application with test configuration
    app := startTestApplication(testDB)
    defer app.Stop()
    
    // Test data
    testData := []byte("test log entry data")
    
    // Send ingestion request
    resp, err := http.Post("http://localhost:8080/ingest", 
        "application/octet-stream", bytes.NewReader(testData))
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, resp.StatusCode)
    resp.Body.Close()
    
    // Verify data was stored
    time.Sleep(100 * time.Millisecond) // Allow processing time
    
    stats, err := getStatsFromAPI()
    require.NoError(t, err)
    assert.Equal(t, 1, stats.TotalRecords)
    assert.Equal(t, int64(len(testData)), stats.TotalSize)
}

func TestDashboardAPI(t *testing.T) {
    // Test all API endpoints end-to-end
    app := startTestApplication("./test_data/api_test.db")
    defer app.Stop()
    
    // Insert test data
    insertTestData(t, 100) // 100 sample entries
    
    // Test statistics endpoint
    resp, err := http.Get("http://localhost:8081/api/stats/summary")
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, resp.StatusCode)
    
    var statsResp map[string]interface{}
    err = json.NewDecoder(resp.Body).Decode(&statsResp)
    require.NoError(t, err)
    resp.Body.Close()
    
    assert.True(t, statsResp["success"].(bool))
    assert.NotNil(t, statsResp["data"])
    
    // Test recent logs endpoint
    resp, err = http.Get("http://localhost:8081/api/logs/recent?limit=10")
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, resp.StatusCode)
    resp.Body.Close()
}
```

#### Performance Tests

Test application performance:

```go
func BenchmarkIngestion(b *testing.B) {
    db, err := NewSQLiteController(":memory:")
    require.NoError(b, err)
    defer db.Close()
    
    testData := make([]byte, 1024) // 1KB test data
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            err := db.InsertLogSize(int64(len(testData)))
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}

func BenchmarkAPIResponse(b *testing.B) {
    // Setup test server
    app := startTestApplication(":memory:")
    defer app.Stop()
    
    // Pre-populate with data
    insertTestData(b, 10000)
    
    client := &http.Client{Timeout: 5 * time.Second}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        resp, err := client.Get("http://localhost:8081/api/stats/summary")
        if err != nil {
            b.Fatal(err)
        }
        resp.Body.Close()
    }
}
```

### Test Execution

```bash
# Run all unit tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test
go test -run TestSQLiteController_InsertLogSize ./src/database

# Run integration tests
go test -tags=integration ./tests/...

# Run benchmarks
go test -bench=. ./...

# Run race detection
go test -race ./...

# Run tests in parallel
go test -parallel 4 ./...
```

### Test Data Management

Create `tests/fixtures/test_data.go`:
```go
package fixtures

import (
    "time"
    "math/rand"
)

type LogEntry struct {
    Size      int64
    Timestamp time.Time
}

func GenerateTestLogs(count int) []LogEntry {
    logs := make([]LogEntry, count)
    baseTime := time.Now().Add(-24 * time.Hour)
    
    for i := 0; i < count; i++ {
        logs[i] = LogEntry{
            Size:      int64(rand.Intn(10000) + 100),
            Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
        }
    }
    
    return logs
}

func GenerateRealisticLogSizes() []int64 {
    // Generate log sizes following realistic distribution
    sizes := make([]int64, 1000)
    
    for i := range sizes {
        switch rand.Intn(4) {
        case 0: // Small logs (80% of traffic)
            sizes[i] = int64(rand.Intn(1000) + 100)
        case 1: // Medium logs (15% of traffic)
            sizes[i] = int64(rand.Intn(10000) + 1000)
        case 2: // Large logs (4% of traffic)
            sizes[i] = int64(rand.Intn(100000) + 10000)
        case 3: // Very large logs (1% of traffic)
            sizes[i] = int64(rand.Intn(1000000) + 100000)
        }
    }
    
    return sizes
}
```

## Debugging

### Logging Configuration

The application uses structured logging with different levels:

```go
// src/utils/logger.go
package utils

import (
    "log/slog"
    "os"
)

func NewLogger(level string) *slog.Logger {
    var logLevel slog.Level
    switch level {
    case "debug":
        logLevel = slog.LevelDebug
    case "info":
        logLevel = slog.LevelInfo
    case "warn":
        logLevel = slog.LevelWarn
    case "error":
        logLevel = slog.LevelError
    default:
        logLevel = slog.LevelInfo
    }
    
    opts := &slog.HandlerOptions{
        Level: logLevel,
    }
    
    handler := slog.NewJSONHandler(os.Stdout, opts)
    return slog.New(handler)
}

// Usage in application
logger := utils.NewLogger(os.Getenv("LOG_LEVEL"))
logger.Info("Application starting", 
    "ingestion_port", 8080,
    "gui_port", 8081)
```

### Debug Endpoints

Add debug endpoints for development:

```go
// Add to main.go for development builds
// +build debug

func addDebugRoutes(mux *http.ServeMux, db *database.SQLiteController) {
    mux.HandleFunc("/debug/stats", func(w http.ResponseWriter, r *http.Request) {
        stats := collectDetailedStats(db)
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(stats)
    })
    
    mux.HandleFunc("/debug/logs", func(w http.ResponseWriter, r *http.Request) {
        logs := getRecentLogs(db, 1000)
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(logs)
    })
}
```

### Common Debugging Scenarios

#### Database Connection Issues

```go
func debugDatabaseConnection(dbPath string) {
    logger.Debug("Attempting database connection", "path", dbPath)
    
    // Check file permissions
    if info, err := os.Stat(dbPath); err != nil {
        logger.Error("Database file stat failed", "error", err)
    } else {
        logger.Debug("Database file info", 
            "size", info.Size(),
            "mode", info.Mode(),
            "modified", info.ModTime())
    }
    
    // Test connection
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        logger.Error("Database open failed", "error", err)
        return
    }
    defer db.Close()
    
    if err := db.Ping(); err != nil {
        logger.Error("Database ping failed", "error", err)
        return
    }
    
    logger.Info("Database connection successful")
}
```

#### HTTP Request Debugging

```go
func debugHTTPHandler(handler http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        logger.Debug("HTTP request started",
            "method", r.Method,
            "url", r.URL.String(),
            "remote_addr", r.RemoteAddr,
            "content_length", r.ContentLength)
        
        // Wrap response writer to capture status code
        wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: 200}
        
        handler(wrappedWriter, r)
        
        duration := time.Since(start)
        logger.Debug("HTTP request completed",
            "method", r.Method,
            "url", r.URL.String(),
            "status_code", wrappedWriter.statusCode,
            "duration_ms", duration.Milliseconds())
    }
}

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}
```

### Performance Profiling

Add profiling endpoints:

```go
// +build debug

import _ "net/http/pprof"

func main() {
    // Add pprof endpoints in debug mode
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    
    // Regular application startup...
}
```

Use profiling:
```bash
# CPU profiling
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profiling
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profiling
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

## Contributing

### Git Workflow

1. **Branch Naming Convention**:
   ```
   feature/add-authentication
   bugfix/fix-database-connection
   hotfix/security-vulnerability
   docs/update-api-documentation
   ```

2. **Commit Message Format**:
   ```
   type(scope): description
   
   Longer explanation if needed
   
   Fixes #123
   ```
   
   Types: feat, fix, docs, style, refactor, test, chore

3. **Pull Request Process**:
   - Create feature branch from main
   - Make changes with tests
   - Run full test suite
   - Submit PR with description
   - Address code review feedback
   - Merge after approval

### Code Review Checklist

- [ ] Code follows Go style guidelines
- [ ] All tests pass
- [ ] New features have tests
- [ ] Documentation is updated
- [ ] No hardcoded secrets or credentials
- [ ] Error handling is comprehensive
- [ ] Performance impact is considered
- [ ] Security implications are reviewed

### Development Workflow

```bash
# Start new feature
git checkout main
git pull origin main
git checkout -b feature/new-feature

# Make changes
# ... develop and test ...

# Pre-commit checks
make lint
make test

# Commit changes
git add .
git commit -m "feat(api): add new statistics endpoint"

# Push and create PR
git push origin feature/new-feature
# Create pull request through GitHub/GitLab
```

## Build and Release

### Automated Build

Create `scripts/build.sh`:
```bash
#!/bin/bash
set -e

VERSION=${1:-"dev"}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD)

echo "Building LogpushEstimator v$VERSION"

# Clean previous builds
rm -rf build/

# Create build directories
mkdir -p build/{linux,windows,macos,docker}

# Build flags
LDFLAGS="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.GitCommit=$GIT_COMMIT"

# Build for Linux (production)
echo "Building for Linux..."
GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o build/linux/logpush-estimator .

# Build for Windows
echo "Building for Windows..."
GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o build/windows/logpush-estimator.exe .

# Build for macOS
echo "Building for macOS..."
GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" -o build/macos/logpush-estimator .

# Copy static files
for platform in linux windows macos; do
    cp -r src/gui/static build/$platform/
    cp -r src/gui/templates build/$platform/
done

# Create archives
cd build
tar -czf logpush-estimator-$VERSION-linux-amd64.tar.gz linux/
zip -r logpush-estimator-$VERSION-windows-amd64.zip windows/
tar -czf logpush-estimator-$VERSION-macos-amd64.tar.gz macos/

echo "Build complete! Artifacts in build/ directory"
```

### Release Process

```bash
# Create release
./scripts/build.sh v1.0.0

# Test release artifacts
cd build/linux
./logpush-estimator --version

# Create GitHub release
gh release create v1.0.0 \
    --title "Release v1.0.0" \
    --notes "Release notes here" \
    build/*.tar.gz build/*.zip
```

### Version Management

Add version information to main.go:
```go
var (
    Version   = "dev"
    BuildTime = "unknown"
    GitCommit = "unknown"
)

func main() {
    if len(os.Args) > 1 && os.Args[1] == "--version" {
        fmt.Printf("LogpushEstimator %s\n", Version)
        fmt.Printf("Build time: %s\n", BuildTime)
        fmt.Printf("Git commit: %s\n", GitCommit)
        os.Exit(0)
    }
    
    // Regular application startup...
}
```

## Performance Optimization

### Database Optimization

```go
// Use prepared statements for repeated queries
type SQLiteController struct {
    db                *sql.DB
    insertStmt        *sql.Stmt
    queryRangeStmt    *sql.Stmt
    summaryStmt       *sql.Stmt
}

func (s *SQLiteController) prepareStatements() error {
    var err error
    
    s.insertStmt, err = s.db.Prepare(`
        INSERT INTO logs (timestamp, filesize) VALUES (?, ?)
    `)
    if err != nil {
        return err
    }
    
    s.queryRangeStmt, err = s.db.Prepare(`
        SELECT id, timestamp, filesize FROM logs 
        WHERE timestamp BETWEEN ? AND ? 
        ORDER BY timestamp DESC
    `)
    if err != nil {
        return err
    }
    
    return nil
}

// Use transactions for bulk operations
func (s *SQLiteController) InsertBulk(entries []LogEntry) error {
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    stmt := tx.Stmt(s.insertStmt)
    for _, entry := range entries {
        _, err := stmt.Exec(entry.Timestamp, entry.FileSize)
        if err != nil {
            return err
        }
    }
    
    return tx.Commit()
}
```

### HTTP Performance

```go
// Use connection pooling
func createHTTPClient() *http.Client {
    transport := &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    }
    
    return &http.Client{
        Transport: transport,
        Timeout:   30 * time.Second,
    }
}

// Add response compression
func compressionMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
            next.ServeHTTP(w, r)
            return
        }
        
        w.Header().Set("Content-Encoding", "gzip")
        gz := gzip.NewWriter(w)
        defer gz.Close()
        
        gzw := &gzipResponseWriter{Writer: gz, ResponseWriter: w}
        next.ServeHTTP(gzw, r)
    })
}
```

### Memory Management

```go
// Use object pools for frequent allocations
var logEntryPool = sync.Pool{
    New: func() interface{} {
        return &LogEntry{}
    },
}

func getLogEntry() *LogEntry {
    return logEntryPool.Get().(*LogEntry)
}

func putLogEntry(entry *LogEntry) {
    entry.Reset() // Clear fields
    logEntryPool.Put(entry)
}

// Stream large responses
func (h *Handlers) streamLargeDataset(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    
    encoder := json.NewEncoder(w)
    w.Write([]byte(`{"data":[`))
    
    first := true
    err := h.db.StreamLogs(func(entry *LogEntry) error {
        if !first {
            w.Write([]byte(`,`))
        }
        first = false
        return encoder.Encode(entry)
    })
    
    w.Write([]byte(`]}`))
    return err
}
```

## Troubleshooting

### Common Development Issues

#### Go Module Issues

```bash
# Clean module cache
go clean -modcache

# Verify and tidy dependencies
go mod verify
go mod tidy

# Download dependencies
go mod download
```

#### Build Issues

```bash
# Clean build cache
go clean -cache
go clean -testcache

# Rebuild with verbose output
go build -v .

# Check for race conditions
go build -race .
```

#### Database Issues

```bash
# Reset development database
rm -f data/dev-logs.db

# Check database integrity
sqlite3 data/dev-logs.db "PRAGMA integrity_check;"

# Analyze database performance
sqlite3 data/dev-logs.db "PRAGMA optimize;"
```

#### Port Conflicts

```bash
# Find process using port
lsof -i :8080
lsof -i :8081

# Kill process
kill -9 <PID>

# Use different ports
export INGESTION_PORT=8082
export GUI_PORT=8083
```

### Debugging Tools

#### Database Query Analysis

```sql
-- Enable query planning
EXPLAIN QUERY PLAN SELECT * FROM logs WHERE timestamp > datetime('now', '-1 hour');

-- Check index usage
.schema
PRAGMA index_info(idx_logs_timestamp);
```

#### HTTP Request Debugging

```bash
# Test ingestion endpoint
curl -v -X POST http://localhost:8080/ingest \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'

# Test API endpoints
curl -v http://localhost:8081/api/stats/summary | jq .

# Load testing
ab -n 1000 -c 10 http://localhost:8080/ingest
```

#### Performance Monitoring

```bash
# Monitor system resources
top -p $(pgrep logpush-estimator)
iostat -x 1

# Monitor database operations
strace -e trace=openat,read,write -p $(pgrep logpush-estimator)
```

For additional troubleshooting help, check the logs and refer to the [deployment troubleshooting guide](../deployment/deployment-guide.md#troubleshooting).

---

*This development guide is current as of September 15, 2025. For the most up-to-date development information, always refer to the official documentation and CONTRIBUTING.md file.*
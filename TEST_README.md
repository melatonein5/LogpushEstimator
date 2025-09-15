# Test Configuration and Helper Scripts

## Running Tests

Run all tests:
```bash
go test ./...
```

Run tests with verbose output:
```bash
go test -v ./...
```

Run tests with coverage:
```bash
go test -v -cover ./...
```

Run specific test packages:
```bash
# Database tests only
go test -v ./src/database/

# Main package tests only
go test -v ./ -run "Test.*" -skip "TestFull.*|TestConcurrent.*|TestAPI.*Integration|TestErrorHandling"

# GUI handler tests only
go test -v ./src/gui/handlers/

# Integration tests only
go test -v ./ -run "Test.*Integration|TestFull.*|TestConcurrent.*|TestErrorHandling"
```

## Test Coverage

Generate coverage report:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Test Database Cleanup

Test databases are automatically cleaned up, but if tests fail, you may need to manually remove:
```bash
rm -f test_*.db
```

## Test Structure

- `src/database/sqlite_controller_test.go` - Database layer unit tests
- `main_test.go` - Main package unit tests (HTTP handlers, server creation)  
- `src/gui/handlers/handlers_test.go` - GUI handlers unit tests (API endpoints, static files)
- `integration_test.go` - Full application integration tests

## Test Features Covered

### Database Tests (`src/database/sqlite_controller_test.go`)
- Database creation and initialization
- Record insertion and retrieval
- Time range queries
- Concurrent access
- Error handling

### Main Package Tests (`main_test.go`)
- Health endpoint
- Ingestion endpoint (all HTTP methods)
- Request validation
- Server creation
- Database integration
- Concurrent request handling

### GUI Handler Tests (`src/gui/handlers/handlers_test.go`)
- Dashboard template serving
- Static file serving with correct MIME types
- API endpoints:
  - `/api/logs/recent`
  - `/api/logs/range` 
  - `/api/stats/summary`
  - `/api/charts/timeseries`
  - `/api/charts/breakdown`
- Error response handling
- CORS headers
- Data aggregation functions

### Integration Tests (`integration_test.go`)
- Full application workflow (ingest → query → verify)
- Concurrent ingestion and querying
- Time range API integration
- Error handling across components
- HTTP status code validation
- JSON response validation

## Mock and Test Data

Tests use temporary SQLite databases that are automatically created and cleaned up. Test data includes:
- Various file sizes (small, medium, large)
- Multiple timestamps
- Edge cases (empty data, invalid formats)
- Concurrent operations

## Performance Testing

The concurrent tests validate:
- Multiple simultaneous ingestion requests
- Concurrent read/write operations
- Database performance under load
- API response times

## Error Handling Testing

Tests cover:
- Invalid HTTP methods
- Empty request bodies
- Malformed parameters
- Database connection issues
- Template parsing errors
- File serving errors
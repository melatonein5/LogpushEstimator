# Design Specifications

## Table of Contents
1. [System Design Patterns](#system-design-patterns)
2. [Component Design](#component-design)
3. [Interface Design](#interface-design)
4. [Database Design](#database-design)
5. [Error Handling Design](#error-handling-design)
6. [Performance Design](#performance-design)
7. [Security Design](#security-design)

## System Design Patterns

### 1. Dual-Server Architecture Pattern

**Pattern Type**: Microservices-like separation of concerns  
**Implementation**: Two independent HTTP servers with shared database

```go
// Server separation pattern
type IngestionServer struct {
    port     string
    database *database.SQLiteController
    logger   *slog.Logger
}

type GUIServer struct {
    port     string
    database *database.SQLiteController
    logger   *slog.Logger
}
```

**Benefits**:
- Clear separation of data ingestion and presentation concerns
- Independent scaling of ingestion vs. dashboard workloads
- Fault isolation between critical ingestion and user-facing services
- Different security policies for different endpoints

**Trade-offs**:
- Increased complexity compared to single-server design
- Shared database creates coupling between services
- Additional port management and firewall configuration

### 2. Repository Pattern

**Pattern Type**: Data Access Layer abstraction  
**Implementation**: SQLiteController abstracts database operations

```go
// Repository pattern interface
type LogRepository interface {
    InsertLogSize(filesize int64) error
    GetAll() ([]LogSize, error)
    QueryByTimeRange(start, end time.Time) ([]LogSize, error)
    Close() error
}

// Concrete implementation
type SQLiteController struct {
    db     *sql.DB
    logger *slog.Logger
}
```

**Benefits**:
- Database implementation can be swapped without changing business logic
- Centralizes data access patterns and error handling
- Enables comprehensive testing with mock implementations
- Provides type safety for database operations

### 3. Handler Factory Pattern

**Pattern Type**: Function factory for HTTP handlers  
**Implementation**: Make* functions that return configured handlers

```go
// Handler factory pattern
func MakeIngestionHandler(db *database.SQLiteController) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Handler implementation with closure over dependencies
    }
}

func MakeAPIHandlers(db *database.SQLiteController, logger *slog.Logger) map[string]http.HandlerFunc {
    // Returns map of configured handlers
}
```

**Benefits**:
- Dependency injection through closures
- Handler configuration at startup time
- Consistent error handling across handlers
- Testable handler creation

### 4. Structured Response Pattern

**Pattern Type**: Consistent API response formatting  
**Implementation**: APIResponse wrapper for all API endpoints

```go
// Structured response pattern
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

func sendSuccessResponse(w http.ResponseWriter, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    response := APIResponse{Success: true, Data: data}
    json.NewEncoder(w).Encode(response)
}
```

**Benefits**:
- Consistent client experience across all endpoints
- Standardized error handling
- Easy to parse programmatically
- Support for both success and error scenarios

## Component Design

### 1. Database Controller Design

**Architecture**: Singleton-like controller with method-based operations

```go
type SQLiteController struct {
    db     *sql.DB      // SQLite connection
    logger *slog.Logger // Structured logger
}

// Lifecycle management
func NewSQLiteController(path string, logger *slog.Logger) (*SQLiteController, error)
func (c *SQLiteController) Close() error

// Data operations
func (c *SQLiteController) InsertLogSize(filesize int64) error
func (c *SQLiteController) GetAll() ([]LogSize, error)
func (c *SQLiteController) QueryByTimeRange(start, end time.Time) ([]LogSize, error)
```

**Design Decisions**:
- **Connection Management**: Single connection per controller instance
- **Error Handling**: Comprehensive error wrapping with context
- **Logging**: Structured logging for all database operations
- **Type Safety**: Strong typing for all data operations

**Concurrency Design**:
- SQLite handles concurrent reads efficiently
- Write operations are serialized by SQLite
- Connection is safe for concurrent use from multiple goroutines
- No application-level locking required

### 2. HTTP Handler Design

**Architecture**: Functional composition with dependency injection

```go
// Handler signature pattern
type HandlerFunc func(http.ResponseWriter, *http.Request)

// Dependency injection through closures
func MakeHandler(deps Dependencies) HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Access deps through closure
    }
}
```

**Request Processing Pipeline**:
1. **Request Validation**: Method, headers, content validation
2. **Parameter Extraction**: Query parameters, path variables, body parsing
3. **Business Logic**: Core processing using injected dependencies
4. **Response Formatting**: Consistent response structure
5. **Error Handling**: Centralized error processing and logging

**Handler Categories**:

#### Ingestion Handlers
```go
func makeIngestionHandler(db *database.SQLiteController) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Validate POST method
        // 2. Read and measure body
        // 3. Validate non-empty content
        // 4. Store in database
        // 5. Return success/error response
    }
}
```

#### API Handlers
```go
func MakeAPIHandlers(db *database.SQLiteController, logger *slog.Logger) map[string]http.HandlerFunc {
    return map[string]http.HandlerFunc{
        "/api/stats/summary":           makeSummaryHandler(db, logger),
        "/api/logs/recent":             makeRecentLogsHandler(db, logger),
        "/api/logs/time-range":         makeTimeRangeHandler(db, logger),
        "/api/charts/timeseries":       makeTimeSeriesHandler(db, logger),
        "/api/charts/size-breakdown":   makeSizeBreakdownHandler(db, logger),
    }
}
```

#### Static Handlers
```go
func MakeStaticFileHandler(logger *slog.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Extract file path from URL
        // 2. Validate path (prevent traversal)
        // 3. Determine MIME type
        // 4. Set appropriate headers
        // 5. Serve file content
    }
}
```

### 3. Logging Design

**Architecture**: Structured logging with contextual information

```go
// Global logger configuration
var slogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

// Contextual logging pattern
func (c *SQLiteController) InsertLogSize(filesize int64) error {
    c.logger.Info("Inserting log size", "filesize", filesize)
    // ... operation ...
    c.logger.Info("Log size inserted successfully", "filesize", filesize)
    return nil
}
```

**Log Levels**:
- **Info**: Normal operations, request processing, successful operations
- **Warn**: Recoverable errors, validation failures, deprecated usage
- **Error**: System errors, database failures, unrecoverable conditions

**Log Structure**:
```json
{
  "time": "2025-09-15T22:01:11.115+01:00",
  "level": "INFO",
  "msg": "Ingestion request received",
  "method": "POST",
  "remote_addr": "127.0.0.1:57066",
  "user_agent": "Go-http-client/1.1",
  "content_length": 15
}
```

## Interface Design

### 1. HTTP API Interface Design

**RESTful Conventions**:
- Resource-based URLs
- HTTP methods reflect operations
- Consistent response formats
- Meaningful HTTP status codes

**URL Structure**:
```
/api/{resource}/{action}
/api/{resource}/{id}
/api/{resource}?{query_params}
```

**Examples**:
```
GET  /api/stats/summary           # Get summary statistics
GET  /api/logs/recent?limit=100   # Get recent logs with limit
GET  /api/logs/time-range?start={iso_date}&end={iso_date}
GET  /api/charts/timeseries?hours=24
POST /ingest                      # Data ingestion
GET  /health                      # Health check
```

### 2. Response Interface Design

**Success Response Format**:
```json
{
  "success": true,
  "data": {
    // Endpoint-specific data structure
  }
}
```

**Error Response Format**:
```json
{
  "success": false,
  "error": "Descriptive error message"
}
```

**Data Type Specifications**:

#### Summary Statistics Response
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

#### Time Series Response
```json
{
  "success": true,
  "data": [
    {
      "timestamp": "2025-09-15T14:00:00Z",
      "count": 45,
      "total_size": 2048000
    }
  ]
}
```

### 3. Database Interface Design

**Entity Design**:
```go
type LogSize struct {
    ID        int64     `json:"id"`
    Timestamp time.Time `json:"timestamp"`
    Filesize  int64     `json:"filesize"`
}
```

**Interface Methods**:
```go
type DatabaseController interface {
    // Lifecycle
    Close() error
    
    // Data operations
    InsertLogSize(filesize int64) error
    GetAll() ([]LogSize, error)
    QueryByTimeRange(start, end time.Time) ([]LogSize, error)
}
```

## Database Design

### Schema Design

```sql
-- Primary data table
CREATE TABLE log_sizes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    filesize INTEGER NOT NULL
);

-- Performance index
CREATE INDEX idx_timestamp ON log_sizes(timestamp);
```

### Design Rationale

**Table Structure**:
- **id**: Auto-incrementing primary key for unique record identification
- **timestamp**: When the log was recorded (indexed for time-range queries)
- **filesize**: Size of the log data in bytes (main metric being tracked)

**Index Strategy**:
- Primary index on `id` (automatic with PRIMARY KEY)
- Secondary index on `timestamp` for time-range query optimization
- No index on `filesize` (not commonly used for filtering)

**Data Types**:
- `INTEGER` for IDs and sizes (sufficient for expected data ranges)
- `DATETIME` for timestamps (SQLite handles ISO 8601 format)
- No VARCHAR/TEXT fields (reduces storage overhead)

### Query Patterns

**Insert Pattern**:
```sql
INSERT INTO log_sizes (timestamp, filesize) VALUES (?, ?)
```

**Time Range Query Pattern**:
```sql
SELECT id, timestamp, filesize 
FROM log_sizes 
WHERE timestamp >= ? AND timestamp < ? 
ORDER BY timestamp
```

**Summary Statistics Pattern**:
```sql
SELECT 
    COUNT(*) as total_records,
    SUM(filesize) as total_size,
    AVG(filesize) as average_size,
    MIN(filesize) as min_size,
    MAX(filesize) as max_size,
    MAX(timestamp) as last_updated
FROM log_sizes
```

### Performance Considerations

**Query Optimization**:
- Time-range queries use index on timestamp
- Summary queries scan full table (acceptable for current scale)
- No complex JOINs required

**Storage Efficiency**:
- Minimal schema reduces storage overhead
- Integer types are storage-efficient
- No nullable columns (all fields required)

**Concurrency**:
- SQLite handles concurrent reads efficiently
- Writes are serialized (acceptable for current throughput)
- WAL mode could be enabled for better concurrency

## Error Handling Design

### Error Categories

**1. Client Errors (4xx)**:
- Invalid HTTP methods
- Empty request bodies
- Malformed query parameters
- Missing required parameters

**2. Server Errors (5xx)**:
- Database connection failures
- Database operation failures
- Template parsing errors
- File system errors

**3. Application Errors**:
- Validation failures
- Resource not found
- Configuration errors

### Error Handling Patterns

**HTTP Handler Error Pattern**:
```go
func handler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    body, err := io.ReadAll(r.Body)
    if err != nil {
        logger.Error("Failed to read request body", "error", err)
        http.Error(w, "Failed to read request body", http.StatusBadRequest)
        return
    }
    
    if len(body) == 0 {
        logger.Warn("Empty request body received")
        http.Error(w, "Request body cannot be empty", http.StatusBadRequest)
        return
    }
    
    // Success path...
}
```

**Database Error Pattern**:
```go
func (c *SQLiteController) InsertLogSize(filesize int64) error {
    _, err := c.db.Exec(`INSERT INTO log_sizes (timestamp, filesize) VALUES (?, ?)`, 
        time.Now(), filesize)
    if err != nil {
        c.logger.Error("Failed to insert log size", "error", err, "filesize", filesize)
        return fmt.Errorf("database insert failed: %w", err)
    }
    return nil
}
```

**API Error Response Pattern**:
```go
func sendErrorResponse(w http.ResponseWriter, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusInternalServerError)
    response := APIResponse{Success: false, Error: message}
    json.NewEncoder(w).Encode(response)
}
```

### Logging Strategy

**Error Context**:
- Include request context (remote address, user agent)
- Include operation context (parameters, data being processed)
- Include error details (original error, stack trace if applicable)

**Error Escalation**:
- Client errors: Log at WARN level
- Server errors: Log at ERROR level
- Critical errors: Log at ERROR level with additional context

## Performance Design

### Response Time Targets

| Endpoint | Target | Rationale |
|----------|--------|-----------|
| `/ingest` | <100ms | Critical path for data ingestion |
| `/health` | <50ms | Used by monitoring systems |
| `/api/*` | <500ms | Interactive dashboard usage |
| Dashboard | <2s | Initial page load |

### Optimization Strategies

**Database Optimization**:
- Index on timestamp for time-range queries
- Prepared statements for common queries
- Connection reuse

**HTTP Optimization**:
- Keep-alive connections
- Appropriate caching headers
- GZIP compression for large responses

**Memory Optimization**:
- Streaming for large result sets
- Bounded result sets with pagination
- Efficient JSON marshaling

### Caching Strategy

**Static Assets**:
```go
w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour cache
```

**API Responses**:
- No caching for real-time data
- ETags for conditional requests (future enhancement)
- Last-Modified headers for static content

## Security Design

### Input Validation

**HTTP Method Validation**:
```go
if r.Method != http.MethodPost {
    http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    return
}
```

**Path Validation** (Static Files):
```go
// Prevent directory traversal
cleanPath := filepath.Clean(path)
if strings.Contains(cleanPath, "..") {
    http.NotFound(w, r)
    return
}
```

**Parameter Validation**:
```go
startStr := r.URL.Query().Get("start")
if startStr == "" {
    sendErrorResponse(w, "Missing required parameter: start")
    return
}

start, err := time.Parse(time.RFC3339, startStr)
if err != nil {
    sendErrorResponse(w, "Invalid start time format")
    return
}
```

### Data Sanitization

**SQL Injection Prevention**:
- Use parameterized queries exclusively
- No dynamic SQL construction
- Validate data types before database operations

**XSS Prevention**:
- HTML template auto-escaping
- Content-Type headers for API responses
- No user-generated content in HTML

### Security Headers

**CORS Headers**:
```go
w.Header().Set("Access-Control-Allow-Origin", "*") // Development only
```

**Content Security Headers**:
```go
w.Header().Set("Content-Type", "application/json")
w.Header().Set("X-Content-Type-Options", "nosniff")
```

### Audit Logging

**Security Events**:
- Failed authentication attempts (when implemented)
- Invalid requests
- Suspicious activity patterns

**Operational Events**:
- All data modifications
- Administrative actions
- System access events

---

*This design specification serves as the technical blueprint for implementing the LogpushEstimator system. It should be referenced during development and updated as the system evolves.*
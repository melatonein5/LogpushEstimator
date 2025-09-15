# Architecture Overview

## Table of Contents
1. [System Architecture](#system-architecture)
2. [Component Overview](#component-overview)
3. [Data Flow](#data-flow)
4. [Technology Stack](#technology-stack)
5. [Scalability Considerations](#scalability-considerations)
6. [Security Architecture](#security-architecture)

## System Architecture

LogpushEstimator follows a dual-server architecture pattern designed for separation of concerns and operational flexibility.

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          External Clients                              │
├─────────────────────────┬───────────────────────────────────────────────┤
│    Cloudflare Logpush   │              Web Browsers                    │
│    Services/Webhooks    │              API Clients                     │
└─────────────────────────┼───────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     LogpushEstimator Application                        │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────┐ ┌─────────────────────────────────┐ │
│  │       Ingestion Server          │ │         GUI Server              │ │
│  │         (Port 8080)             │ │        (Port 8081)              │ │
│  │                                 │ │                                 │ │
│  │ ┌─────────────────────────────┐ │ │ ┌─────────────────────────────┐ │ │
│  │ │ POST /ingest               │ │ │ │ GET /                       │ │ │
│  │ │ GET /health                │ │ │ │ GET /dashboard              │ │ │
│  │ └─────────────────────────────┘ │ │ │ GET /api/stats/summary      │ │ │
│  │                                 │ │ │ GET /api/logs/recent        │ │ │
│  │ ┌─────────────────────────────┐ │ │ │ GET /api/logs/time-range    │ │ │
│  │ │   HTTP Request Handler      │ │ │ │ GET /api/charts/timeseries  │ │ │
│  │ │   - Method validation       │ │ │ │ GET /api/charts/breakdown   │ │ │
│  │ │   - Body size measurement   │ │ │ │ GET /static/*               │ │ │
│  │ │   - Database insertion      │ │ │ └─────────────────────────────┘ │ │
│  │ └─────────────────────────────┘ │ │                                 │ │
│  └─────────────────────────────────┘ │ ┌─────────────────────────────┐ │ │
│                                      │ │   Template Engine           │ │ │
│                                      │ │   - HTML rendering          │ │ │
│                                      │ │   - Static file serving     │ │ │
│                                      │ └─────────────────────────────┘ │ │
│                                      │                                 │ │
│                                      │ ┌─────────────────────────────┐ │ │
│                                      │ │   API Handlers              │ │ │
│                                      │ │   - JSON response formatting│ │ │
│                                      │ │   - Data aggregation        │ │ │
│                                      │ │   - Error handling          │ │ │
│                                      │ └─────────────────────────────┘ │ │
│                                      └─────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────────────┤
│                          Shared Components                              │
│  ┌─────────────────────────────────────────────────────────────────────┐ │
│  │                    Database Controller                              │ │
│  │  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────────┐   │ │
│  │  │ InsertLogSize() │ │    GetAll()     │ │ QueryByTimeRange()  │   │ │
│  │  └─────────────────┘ └─────────────────┘ └─────────────────────┘   │ │
│  └─────────────────────────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────────────────┐ │
│  │                     Structured Logger                              │ │
│  │  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────────┐   │ │
│  │  │   Info Logs     │ │   Error Logs    │ │   Debug Logs        │   │ │
│  │  └─────────────────┘ └─────────────────┘ └─────────────────────┘   │ │
│  └─────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         SQLite Database                                │
│                           (logpush.db)                                 │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                        log_sizes table                         │   │
│  │  ┌──────────┬──────────────┬─────────────────────────────────┐  │   │
│  │  │ id       │ timestamp    │ filesize                        │  │   │
│  │  │ INTEGER  │ DATETIME     │ INTEGER                         │  │   │
│  │  │ PK       │ NOT NULL     │ NOT NULL                        │  │   │
│  │  │ AUTOINCR │ (indexed)    │ (bytes)                         │  │   │
│  │  └──────────┴──────────────┴─────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

## Component Overview

### 1. Ingestion Server (Port 8080)

**Purpose**: Handles incoming log data from Cloudflare Logpush services and external clients.

**Key Components**:
- **HTTP Request Handler**: Processes POST requests to `/ingest` endpoint
- **Body Size Measurement**: Calculates payload size for storage and analysis
- **Validation Layer**: Ensures request method and content validity
- **Database Interface**: Stores log size records with timestamps

**Endpoints**:
- `POST /ingest` - Primary log data ingestion endpoint
- `GET /health` - Health check for monitoring systems

### 2. GUI Server (Port 8081)

**Purpose**: Provides web-based dashboard and REST API for data visualization and access.

**Key Components**:
- **Template Engine**: Renders HTML dashboard using Go templates
- **Static File Server**: Serves CSS, JavaScript, and image assets
- **API Handlers**: Provide JSON endpoints for data retrieval
- **Response Formatter**: Ensures consistent API response structure

**Endpoints**:
- `GET /` - Main dashboard interface
- `GET /dashboard` - Alternative dashboard path
- `GET /api/*` - REST API endpoints (see [API Reference](../api/api-reference.md))
- `GET /static/*` - Static asset serving

### 3. Database Controller

**Purpose**: Provides abstraction layer over SQLite database operations.

**Key Features**:
- **Connection Management**: Handles database connection lifecycle
- **Schema Management**: Ensures proper table and index creation
- **Type Safety**: Provides strongly-typed Go interfaces for database operations
- **Error Handling**: Comprehensive error handling with structured logging

**Methods**:
- `NewSQLiteController()` - Initialize database connection
- `InsertLogSize()` - Store log size records
- `GetAll()` - Retrieve all records
- `QueryByTimeRange()` - Time-filtered queries
- `Close()` - Clean connection shutdown

### 4. Structured Logger

**Purpose**: Provides consistent, structured logging across all components.

**Features**:
- **JSON Format**: Machine-readable log output
- **Contextual Logging**: Includes request context and metadata
- **Log Levels**: Info, Warning, Error levels for different scenarios
- **Performance Logging**: Request timing and database operation metrics

## Data Flow

### Log Ingestion Flow

```
1. Cloudflare Logpush Service
   │
   └─ POST /ingest (log data)
      │
2. Ingestion Server
   ├─ Validate HTTP method (POST only)
   ├─ Read request body
   ├─ Measure body size
   ├─ Validate non-empty content
   │
3. Database Controller
   ├─ Insert record (timestamp, filesize)
   ├─ Log operation result
   │
4. Response
   └─ HTTP 200 OK / Error Status
```

### Dashboard Data Flow

```
1. Web Browser Request
   │
   └─ GET /dashboard
      │
2. GUI Server
   ├─ Parse dashboard template
   ├─ Execute template with context
   │
3. Template Engine
   ├─ Render HTML content
   ├─ Include static asset references
   │
4. Response
   └─ HTML dashboard page
```

### API Data Flow

```
1. API Client Request
   │
   └─ GET /api/stats/summary
      │
2. GUI Server
   ├─ Route to appropriate handler
   ├─ Extract query parameters
   │
3. Database Controller
   ├─ Execute data query
   ├─ Return result set
   │
4. API Handler
   ├─ Process/aggregate data
   ├─ Format JSON response
   │
5. Response
   └─ JSON API response
```

## Technology Stack

### Core Technologies

| Component | Technology | Version | Purpose |
|-----------|------------|---------|---------|
| **Runtime** | Go | 1.21+ | Primary programming language |
| **Database** | SQLite3 | 3.x | Data persistence layer |
| **HTTP Server** | net/http | stdlib | HTTP request handling |
| **Logging** | slog | stdlib | Structured logging |
| **Testing** | testing | stdlib | Unit and integration testing |
| **Templates** | html/template | stdlib | HTML rendering |

### Dependencies

```go
module github.com/melatonein5/LogpushEstimator

go 1.21

require (
    github.com/mattn/go-sqlite3 v1.14.17
)
```

### Frontend Technologies

- **HTML5**: Semantic markup for dashboard interface
- **CSS3**: Styling with modern CSS features
- **Vanilla JavaScript**: Client-side interactivity without frameworks
- **Chart Libraries**: For data visualization (if implemented)

## Scalability Considerations

### Current Architecture Limitations

1. **Single Node**: Application runs on single server instance
2. **SQLite Constraints**: File-based database limits concurrent write operations
3. **Memory Usage**: All-in-memory data processing for analytics
4. **No Load Balancing**: Single point of failure for both servers

### Scaling Strategies

#### Horizontal Scaling Options

1. **Load Balancer + Multiple Instances**
   ```
   Load Balancer
   ├─ LogpushEstimator Instance 1
   ├─ LogpushEstimator Instance 2
   └─ LogpushEstimator Instance N
   ```

2. **Separate Ingestion and GUI Scaling**
   ```
   Ingestion Load Balancer → Multiple Ingestion Servers
   GUI Load Balancer → Multiple GUI Servers
   ```

#### Database Scaling Options

1. **PostgreSQL Migration**: Replace SQLite with PostgreSQL for better concurrency
2. **Database Clustering**: Master-slave or cluster setup
3. **Read Replicas**: Separate read and write operations
4. **Data Partitioning**: Time-based table partitioning

#### Caching Strategies

1. **In-Memory Cache**: Redis or Memcached for frequently accessed data
2. **Application-Level Cache**: Go sync.Map for session-based caching
3. **CDN**: For static assets and dashboard content

### Performance Optimization

1. **Database Indexing**: Optimize queries with proper indexes
2. **Connection Pooling**: Manage database connections efficiently
3. **Batch Processing**: Group database operations for better throughput
4. **Compression**: Compress API responses and static assets

## Security Architecture

### Current Security Measures

1. **Input Validation**: HTTP method and content validation
2. **Error Handling**: No sensitive information in error responses
3. **Logging**: Comprehensive audit trail of all operations

### Security Recommendations

#### Authentication & Authorization

1. **API Keys**: Implement API key authentication for ingestion endpoint
2. **JWT Tokens**: Use JWT for dashboard authentication
3. **Role-Based Access**: Different access levels for users
4. **Rate Limiting**: Prevent abuse of API endpoints

#### Network Security

1. **HTTPS**: TLS encryption for all communications
2. **Firewall Rules**: Restrict access to necessary ports only
3. **VPN Access**: Secure access to management interfaces
4. **IP Whitelisting**: Restrict ingestion to known Cloudflare IPs

#### Data Security

1. **Encryption at Rest**: Encrypt SQLite database file
2. **Encryption in Transit**: HTTPS/TLS for all communications
3. **Data Sanitization**: Sanitize log data before storage
4. **Backup Encryption**: Encrypt database backups

#### Application Security

1. **Input Sanitization**: Prevent injection attacks
2. **CORS Configuration**: Proper CORS headers for API endpoints
3. **Security Headers**: Implement security HTTP headers
4. **Dependency Scanning**: Regular vulnerability scans of dependencies

### Compliance Considerations

1. **Data Retention**: Configurable data retention policies
2. **GDPR Compliance**: If processing EU data
3. **Audit Logging**: Comprehensive audit trails
4. **Data Anonymization**: Remove personally identifiable information

---

*For detailed implementation guides, see the [Development](../development/) and [Deployment](../deployment/) documentation.*
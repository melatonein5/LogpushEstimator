# Database Schema Documentation

## Table of Contents
1. [Overview](#overview)
2. [Schema Definition](#schema-definition)
3. [Data Models](#data-models)
4. [Indexes and Performance](#indexes-and-performance)
5. [Query Patterns](#query-patterns)
6. [Data Migration](#data-migration)
7. [Maintenance and Optimization](#maintenance-and-optimization)

## Overview

LogpushEstimator uses SQLite as its primary database for storing log size records with timestamps. The schema is designed for simplicity, performance, and efficient time-series data analysis.

### Design Principles

- **Simplicity**: Minimal schema with only essential fields
- **Performance**: Optimized for time-range queries and aggregations
- **Scalability**: Designed to handle millions of records efficiently
- **Integrity**: Strong data consistency and type safety
- **Maintainability**: Clear structure with comprehensive documentation

### Database Technology

- **Engine**: SQLite 3.35+
- **File Format**: Single file database (logpush.db)
- **Concurrent Access**: Multiple readers, single writer
- **ACID Compliance**: Full ACID transaction support
- **Backup**: File-based backup and restore

## Schema Definition

### Core Tables

#### log_sizes Table

The primary table storing all log size records with timestamps.

```sql
CREATE TABLE log_sizes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    filesize INTEGER NOT NULL
);
```

**Field Specifications**:

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY, AUTOINCREMENT | Unique record identifier |
| `timestamp` | DATETIME | NOT NULL | When the log was recorded (ISO 8601 format) |
| `filesize` | INTEGER | NOT NULL | Size of log data in bytes |

### Indexes

#### Primary Index
```sql
-- Automatically created with PRIMARY KEY
CREATE UNIQUE INDEX sqlite_autoindex_log_sizes_1 ON log_sizes(id);
```

#### Performance Indexes
```sql
-- Time-range query optimization
CREATE INDEX idx_timestamp ON log_sizes(timestamp);
```

**Index Strategy**:
- **Primary Index**: Ensures unique identification and fast ID-based lookups
- **Timestamp Index**: Optimizes time-range queries (most common access pattern)
- **No Filesize Index**: Size-based filtering is rare, index would add overhead

### Schema Creation Script

```sql
-- LogpushEstimator Database Schema
-- Version: 1.0
-- Created: 2025-09-15

-- Enable foreign key constraints (good practice)
PRAGMA foreign_keys = ON;

-- Enable WAL mode for better concurrency (optional)
-- PRAGMA journal_mode = WAL;

-- Create main data table
CREATE TABLE IF NOT EXISTS log_sizes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    filesize INTEGER NOT NULL
);

-- Create performance index
CREATE INDEX IF NOT EXISTS idx_timestamp ON log_sizes(timestamp);

-- Insert schema version for migration tracking
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO schema_version (version) VALUES (1);
```

## Data Models

### LogSize Entity

**Go Struct Representation**:
```go
type LogSize struct {
    ID        int64     `json:"id" db:"id"`
    Timestamp time.Time `json:"timestamp" db:"timestamp"`
    Filesize  int64     `json:"filesize" db:"filesize"`
}
```

**JSON Representation**:
```json
{
    "id": 12345,
    "timestamp": "2025-09-15T14:30:45Z",
    "filesize": 2048
}
```

**Database Representation**:
```
id=12345, timestamp='2025-09-15 14:30:45', filesize=2048
```

### Data Constraints

#### Field Constraints

**ID Field**:
- Type: 64-bit signed integer
- Range: 1 to 9,223,372,036,854,775,807
- Auto-generated, sequential
- Guaranteed unique within table

**Timestamp Field**:
- Format: ISO 8601 (YYYY-MM-DDTHH:MM:SSZ)
- Timezone: UTC (recommended)
- Range: 1970-01-01 to 2038-01-19 (Unix timestamp compatible)
- Precision: Second-level precision

**Filesize Field**:
- Type: 64-bit signed integer
- Range: 0 to 9,223,372,036,854,775,807 bytes
- Unit: Bytes
- Validation: Must be non-negative

#### Business Rules

1. **Timestamp Validation**: Must be valid datetime, preferably in UTC
2. **Filesize Validation**: Must be positive integer (>0 bytes)
3. **Uniqueness**: No unique constraints beyond primary key (duplicates allowed)
4. **Referential Integrity**: No foreign key relationships

### Data Types and Storage

#### SQLite Data Type Mapping

| Schema Type | SQLite Storage | Go Type | JSON Type | Size |
|-------------|---------------|---------|-----------|------|
| INTEGER | INTEGER | int64 | number | 8 bytes |
| DATETIME | TEXT | time.Time | string | ~25 bytes |

#### Storage Calculations

**Per Record**:
- ID: 8 bytes
- Timestamp: ~25 bytes (ISO 8601 string)
- Filesize: 8 bytes
- SQLite overhead: ~8 bytes
- **Total per record**: ~49 bytes

**Storage Projections**:
- 1,000 records: ~49 KB
- 100,000 records: ~4.9 MB
- 1,000,000 records: ~49 MB
- 10,000,000 records: ~490 MB

## Indexes and Performance

### Index Analysis

#### idx_timestamp Performance

**Query Types Optimized**:
```sql
-- Time range queries (most common)
SELECT * FROM log_sizes WHERE timestamp >= ? AND timestamp < ?;

-- Recent data queries
SELECT * FROM log_sizes WHERE timestamp > ? ORDER BY timestamp DESC LIMIT ?;

-- Aggregation with time filtering
SELECT COUNT(*), SUM(filesize) FROM log_sizes WHERE timestamp >= ?;
```

**Performance Characteristics**:
- **Index Size**: ~40% of table size for timestamp index
- **Query Performance**: O(log n) for range queries
- **Insert Performance**: Minimal overhead for index maintenance
- **Update Performance**: No updates expected in this schema

#### Index Maintenance

**Automatic Maintenance**:
- SQLite automatically maintains indexes
- B-tree structure provides balanced performance
- No manual index rebuilding required

**Monitoring Index Health**:
```sql
-- Check index usage statistics
PRAGMA index_info(idx_timestamp);

-- Analyze query plans
EXPLAIN QUERY PLAN SELECT * FROM log_sizes WHERE timestamp >= '2025-09-15';
```

### Query Performance Optimization

#### Efficient Query Patterns

**Time Range Queries**:
```sql
-- GOOD: Uses index effectively
SELECT * FROM log_sizes 
WHERE timestamp >= '2025-09-15T00:00:00Z' 
  AND timestamp < '2025-09-16T00:00:00Z'
ORDER BY timestamp;

-- AVOID: Functions prevent index usage
SELECT * FROM log_sizes 
WHERE strftime('%Y-%m-%d', timestamp) = '2025-09-15';
```

**Aggregation Queries**:
```sql
-- GOOD: Efficient aggregation
SELECT 
    COUNT(*) as total_records,
    SUM(filesize) as total_size,
    AVG(filesize) as average_size
FROM log_sizes 
WHERE timestamp >= ?;

-- GOOD: Using subqueries for complex aggregations
SELECT 
    date(timestamp) as log_date,
    COUNT(*) as daily_count,
    SUM(filesize) as daily_size
FROM log_sizes 
WHERE timestamp >= '2025-09-01'
GROUP BY date(timestamp)
ORDER BY log_date;
```

#### Query Performance Targets

| Query Type | Target Time | Expected Volume |
|------------|-------------|-----------------|
| Single insert | <10ms | High frequency |
| Time range (1 day) | <100ms | Medium frequency |
| Full table aggregation | <500ms | Low frequency |
| Recent logs (100 records) | <50ms | High frequency |

## Query Patterns

### Data Insertion Patterns

#### Single Record Insertion
```sql
-- Primary insertion pattern
INSERT INTO log_sizes (timestamp, filesize) VALUES (?, ?);
```

**Go Implementation**:
```go
func (c *SQLiteController) InsertLogSize(filesize int64) error {
    _, err := c.db.Exec(
        `INSERT INTO log_sizes (timestamp, filesize) VALUES (?, ?)`,
        time.Now().UTC(), filesize)
    return err
}
```

#### Batch Insertion (Future Enhancement)
```sql
-- Batch insertion for improved performance
INSERT INTO log_sizes (timestamp, filesize) VALUES 
    (?, ?), (?, ?), (?, ?);
```

### Data Retrieval Patterns

#### All Records Retrieval
```sql
SELECT id, timestamp, filesize 
FROM log_sizes 
ORDER BY id;
```

#### Time Range Queries
```sql
-- Inclusive start, exclusive end
SELECT id, timestamp, filesize 
FROM log_sizes 
WHERE timestamp >= ? AND timestamp < ? 
ORDER BY timestamp;
```

#### Recent Records
```sql
-- Most recent N records
SELECT id, timestamp, filesize 
FROM log_sizes 
ORDER BY timestamp DESC 
LIMIT ?;
```

#### Statistical Aggregations
```sql
-- Summary statistics
SELECT 
    COUNT(*) as total_records,
    SUM(filesize) as total_size,
    AVG(filesize) as average_size,
    MIN(filesize) as min_size,
    MAX(filesize) as max_size,
    MAX(timestamp) as last_updated
FROM log_sizes;
```

#### Time Series Aggregations
```sql
-- Hourly aggregation
SELECT 
    strftime('%Y-%m-%d %H:00:00', timestamp) as hour_bucket,
    COUNT(*) as record_count,
    SUM(filesize) as total_size
FROM log_sizes 
WHERE timestamp >= ?
GROUP BY strftime('%Y-%m-%d %H', timestamp)
ORDER BY hour_bucket;
```

### Advanced Query Patterns

#### Size Distribution Analysis
```sql
-- Size range distribution
SELECT 
    CASE 
        WHEN filesize < 1024 THEN 'Small (<1KB)'
        WHEN filesize < 10240 THEN 'Medium (1-10KB)'
        WHEN filesize < 102400 THEN 'Large (10-100KB)'
        ELSE 'Very Large (>100KB)'
    END as size_category,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / (SELECT COUNT(*) FROM log_sizes), 2) as percentage
FROM log_sizes
GROUP BY size_category;
```

#### Growth Trend Analysis
```sql
-- Daily growth trends
SELECT 
    date(timestamp) as log_date,
    COUNT(*) as daily_records,
    SUM(filesize) as daily_size,
    AVG(filesize) as avg_size
FROM log_sizes 
WHERE timestamp >= date('now', '-30 days')
GROUP BY date(timestamp)
ORDER BY log_date;
```

## Data Migration

### Schema Versioning

#### Version Tracking Table
```sql
CREATE TABLE schema_version (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);
```

#### Migration Script Template
```sql
-- Migration: Add new column example
-- Version: 2
-- Date: 2025-09-15

BEGIN TRANSACTION;

-- Check current version
INSERT OR IGNORE INTO schema_version (version, description) 
VALUES (2, 'Add new column example');

-- Perform migration (example)
-- ALTER TABLE log_sizes ADD COLUMN new_field TEXT;

-- Update schema version
UPDATE schema_version SET applied_at = CURRENT_TIMESTAMP WHERE version = 2;

COMMIT;
```

### Data Export and Import

#### Export Data
```sql
-- Export to CSV format
.mode csv
.headers on
.output logpush_backup.csv
SELECT * FROM log_sizes;
.output stdout
```

#### Import Data
```sql
-- Import from CSV
.mode csv
.import logpush_backup.csv log_sizes_temp

-- Validate and merge
INSERT INTO log_sizes (timestamp, filesize)
SELECT timestamp, filesize FROM log_sizes_temp
WHERE timestamp IS NOT NULL AND filesize > 0;

DROP TABLE log_sizes_temp;
```

### Backup and Restore

#### Database Backup
```bash
# Full database backup
sqlite3 logpush.db ".backup logpush_backup_$(date +%Y%m%d).db"

# SQL dump backup
sqlite3 logpush.db ".dump" > logpush_backup_$(date +%Y%m%d).sql
```

#### Database Restore
```bash
# Restore from backup file
cp logpush_backup_20250915.db logpush.db

# Restore from SQL dump
sqlite3 logpush_new.db < logpush_backup_20250915.sql
```

## Maintenance and Optimization

### Database Maintenance Tasks

#### Regular Maintenance
```sql
-- Analyze database statistics (monthly)
ANALYZE;

-- Rebuild indexes if necessary (rarely needed)
REINDEX idx_timestamp;

-- Check database integrity (monthly)
PRAGMA integrity_check;

-- Vacuum database to reclaim space (after large deletions)
VACUUM;
```

#### Performance Monitoring
```sql
-- Check database size
SELECT page_count * page_size as size_bytes FROM pragma_page_count(), pragma_page_size();

-- Check index usage
EXPLAIN QUERY PLAN SELECT * FROM log_sizes WHERE timestamp >= '2025-09-15';

-- Check table statistics
SELECT * FROM sqlite_stat1 WHERE tbl = 'log_sizes';
```

### Data Retention Policies

#### Automatic Cleanup (Future Enhancement)
```sql
-- Delete records older than 1 year
DELETE FROM log_sizes 
WHERE timestamp < datetime('now', '-1 year');

-- Archive old data before deletion
CREATE TABLE log_sizes_archive AS 
SELECT * FROM log_sizes 
WHERE timestamp < datetime('now', '-1 year');
```

#### Partitioning Strategy (Future Enhancement)
```sql
-- Create monthly tables for better performance
CREATE TABLE log_sizes_202509 AS 
SELECT * FROM log_sizes 
WHERE timestamp >= '2025-09-01' AND timestamp < '2025-10-01';

-- Create view for unified access
CREATE VIEW log_sizes_current AS
SELECT * FROM log_sizes_202509
UNION ALL
SELECT * FROM log_sizes_202510;
```

### Performance Tuning

#### SQLite Configuration
```sql
-- Optimize for write performance
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = 10000;
PRAGMA temp_store = MEMORY;
```

#### Connection Settings
```go
// Go database connection optimization
db, err := sql.Open("sqlite3", "logpush.db?cache=shared&mode=rwc")
if err != nil {
    return nil, err
}

// Set connection pool limits
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(25)
db.SetConnMaxLifetime(5 * time.Minute)
```

---

*This schema documentation should be updated whenever database changes are made. All migrations should be tested thoroughly before applying to production data.*
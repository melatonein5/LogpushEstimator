// Package database provides SQLite-based data persistence for LogpushEstimator.
//
// This package implements a minimal SQLite database controller for storing
// and querying log size records with timestamps. It's designed specifically
// for the LogpushEstimator application to track Cloudflare log data volumes
// over time.
//
// # Usage
//
// Create a new database controller:
//
//	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
//	db, err := database.NewSQLiteController("logpush.db", logger)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer db.Close()
//
// Insert log size records:
//
//	err = db.InsertLogSize(1024) // Insert a 1KB log record
//	if err != nil {
//		log.Printf("Failed to insert log size: %v", err)
//	}
//
// Query records:
//
//	logs, err := db.GetAll()
//	if err != nil {
//		log.Printf("Failed to get logs: %v", err)
//	}
//
// # Database Schema
//
// The package maintains a single table 'log_sizes' with the following structure:
//
//	CREATE TABLE log_sizes (
//		id INTEGER PRIMARY KEY AUTOINCREMENT,
//		timestamp DATETIME NOT NULL,
//		filesize INTEGER NOT NULL
//	);
//
// An index on the timestamp column is automatically created for efficient
// time-range queries.
package database

import (
	"database/sql"
	"log/slog"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// LogSize represents a single log size record with timestamp.
// This struct maps directly to the log_sizes table in the database.
type LogSize struct {
	ID        int64     // Unique identifier (auto-increment primary key)
	Timestamp time.Time // When the log was recorded
	Filesize  int64     // Size of the log data in bytes
}

// SQLiteController provides database operations for log size tracking.
// It encapsulates SQLite database connections and provides methods for
// inserting and querying log size records with proper error handling
// and structured logging.
type SQLiteController struct {
	db     *sql.DB      // SQLite database connection
	logger *slog.Logger // Structured logger for database operations
}

// NewSQLiteController creates a new database controller and initializes the database.
// It opens or creates a SQLite database at the specified path, creates the required
// tables and indexes if they don't exist, and returns a configured controller.
//
// Parameters:
//   - path: Database file path. If empty, defaults to "logpush.db"
//   - logger: Logger for database operations. If nil, creates a default logger
//
// Returns:
//   - *SQLiteController: Configured database controller
//   - error: Any error encountered during initialization
//
// The function ensures the database schema is properly set up with:
//   - log_sizes table for storing log records
//   - timestamp index for efficient time-range queries
func NewSQLiteController(path string, logger *slog.Logger) (*SQLiteController, error) {
	if path == "" {
		path = "logpush.db"
	}
	if logger == nil {
		// Create a no-op logger if none provided
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	logger.Info("Opening SQLite database", "path", path)
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		logger.Error("Failed to open SQLite database", "error", err, "path", path)
		return nil, err
	}

	logger.Info("Creating log_sizes table if not exists")
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS log_sizes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		filesize INTEGER NOT NULL
	);`)
	if err != nil {
		logger.Error("Failed to create log_sizes table", "error", err)
		db.Close()
		return nil, err
	}

	logger.Info("Creating timestamp index if not exists")
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_log_sizes_timestamp ON log_sizes(timestamp);`)
	if err != nil {
		logger.Error("Failed to create timestamp index", "error", err)
		db.Close()
		return nil, err
	}

	logger.Info("SQLite database setup completed successfully")
	return &SQLiteController{db: db, logger: logger}, nil
}

// InsertLogSize inserts a new log size record with the current timestamp.
// This is the primary method for recording log data sizes as they are received.
//
// Parameters:
//   - filesize: Size of the log data in bytes (must be positive)
//
// Returns:
//   - error: Any error encountered during database insertion
//
// The function automatically uses the current time as the timestamp for the record.
func (c *SQLiteController) InsertLogSize(filesize int64) error {
	c.logger.Info("Inserting log size", "filesize", filesize)
	_, err := c.db.Exec(`INSERT INTO log_sizes (timestamp, filesize) VALUES (?, ?)`, time.Now(), filesize)
	if err != nil {
		c.logger.Error("Failed to insert log size", "error", err, "filesize", filesize)
		return err
	}
	c.logger.Info("Log size inserted successfully", "filesize", filesize)
	return nil
}

// QueryByTimeRange returns all log size records within a specified time range.
// This method is useful for generating reports and analytics for specific time periods.
//
// Parameters:
//   - start: Start time (inclusive) - records at or after this time are included
//   - end: End time (exclusive) - records before this time are included
//
// Returns:
//   - []LogSize: Slice of log size records ordered by timestamp
//   - error: Any error encountered during the query
//
// The results are automatically sorted by timestamp in ascending order.
func (c *SQLiteController) QueryByTimeRange(start, end time.Time) ([]LogSize, error) {
	c.logger.Info("Querying log sizes by time range", "start", start, "end", end)
	rows, err := c.db.Query(`SELECT id, timestamp, filesize FROM log_sizes WHERE timestamp >= ? AND timestamp < ? ORDER BY timestamp`, start, end)
	if err != nil {
		c.logger.Error("Failed to query log sizes by time range", "error", err, "start", start, "end", end)
		return nil, err
	}
	defer rows.Close()
	var out []LogSize
	for rows.Next() {
		var l LogSize
		err := rows.Scan(&l.ID, &l.Timestamp, &l.Filesize)
		if err != nil {
			c.logger.Error("Failed to scan log size row", "error", err)
			return nil, err
		}
		out = append(out, l)
	}
	c.logger.Info("Query completed successfully", "start", start, "end", end, "count", len(out))
	return out, nil
}

// GetAll returns all log size records from the database.
// This method retrieves every record in the log_sizes table, ordered by ID.
// Use with caution on large datasets as it loads all records into memory.
//
// Returns:
//   - []LogSize: Slice of all log size records ordered by ID
//   - error: Any error encountered during the query
//
// For large datasets, consider using QueryByTimeRange instead to limit results.
func (c *SQLiteController) GetAll() ([]LogSize, error) {
	c.logger.Info("Querying all log sizes")
	rows, err := c.db.Query(`SELECT id, timestamp, filesize FROM log_sizes ORDER BY id`)
	if err != nil {
		c.logger.Error("Failed to query all log sizes", "error", err)
		return nil, err
	}
	defer rows.Close()
	var out []LogSize
	for rows.Next() {
		var l LogSize
		err := rows.Scan(&l.ID, &l.Timestamp, &l.Filesize)
		if err != nil {
			c.logger.Error("Failed to scan log size row", "error", err)
			return nil, err
		}
		out = append(out, l)
	}
	c.logger.Info("Query all completed successfully", "count", len(out))
	return out, nil
}

// Close closes the database connection and releases associated resources.
// This method should be called when the controller is no longer needed,
// typically using defer after creating the controller.
//
// Returns:
//   - error: Any error encountered while closing the database connection
//
// Example usage:
//
//	db, err := NewSQLiteController("logpush.db", logger)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer db.Close() // Ensure cleanup
func (c *SQLiteController) Close() error {
	c.logger.Info("Closing database connection")
	err := c.db.Close()
	if err != nil {
		c.logger.Error("Failed to close database", "error", err)
	} else {
		c.logger.Info("Database connection closed successfully")
	}
	return err
}

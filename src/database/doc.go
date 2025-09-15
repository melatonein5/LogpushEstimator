// Package database provides SQLite-based data persistence for LogpushEstimator.
//
// This package offers a clean abstraction over SQLite database operations,
// specifically designed for storing and analyzing Cloudflare log size data.
// It handles database initialization, schema management, and provides
// type-safe methods for data insertion and retrieval.
//
// # Examples
//
// Basic usage for storing log data:
//
//	package main
//
//	import (
//		"log"
//		"log/slog"
//		"os"
//		"time"
//
//		"github.com/melatonein5/LogpushEstimator/src/database"
//	)
//
//	func main() {
//		// Create a logger
//		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
//
//		// Initialize database
//		db, err := database.NewSQLiteController("logpush.db", logger)
//		if err != nil {
//			log.Fatal("Failed to initialize database:", err)
//		}
//		defer db.Close()
//
//		// Insert log size data
//		err = db.InsertLogSize(2048) // 2KB log
//		if err != nil {
//			log.Printf("Failed to insert log size: %v", err)
//		}
//
//		// Query all records
//		logs, err := db.GetAll()
//		if err != nil {
//			log.Printf("Failed to query logs: %v", err)
//			return
//		}
//
//		log.Printf("Found %d log records", len(logs))
//		for _, logEntry := range logs {
//			log.Printf("Log ID: %d, Size: %d bytes, Time: %v",
//				logEntry.ID, logEntry.Filesize, logEntry.Timestamp)
//		}
//	}
//
// Time range queries for analytics:
//
//	// Query logs from the last 24 hours
//	end := time.Now()
//	start := end.Add(-24 * time.Hour)
//
//	recentLogs, err := db.QueryByTimeRange(start, end)
//	if err != nil {
//		log.Printf("Failed to query recent logs: %v", err)
//		return
//	}
//
//	// Calculate total size for the period
//	var totalSize int64
//	for _, logEntry := range recentLogs {
//		totalSize += logEntry.Filesize
//	}
//
//	log.Printf("Total log data in last 24h: %d bytes (%d records)",
//		totalSize, len(recentLogs))
//
// # Database Schema
//
// The package maintains a simple but effective schema optimized for time-series
// log data analysis:
//
//	Table: log_sizes
//	┌─────────────┬──────────────┬─────────────────────────────────┐
//	│ Column      │ Type         │ Description                     │
//	├─────────────┼──────────────┼─────────────────────────────────┤
//	│ id          │ INTEGER      │ Primary key (auto-increment)    │
//	│ timestamp   │ DATETIME     │ When the log was recorded       │
//	│ filesize    │ INTEGER      │ Size of log data in bytes       │
//	└─────────────┴──────────────┴─────────────────────────────────┘
//
//	Index: idx_timestamp on (timestamp)
//	- Optimizes time-range queries for analytics
//
// # Thread Safety
//
// The SQLiteController is safe for concurrent use. SQLite handles concurrent
// reads efficiently, and the controller's methods are designed to work correctly
// with Go's concurrent execution model.
//
// # Error Handling
//
// All database operations return descriptive errors that can be used for
// debugging and monitoring. The package uses structured logging to provide
// detailed operation context.
package database

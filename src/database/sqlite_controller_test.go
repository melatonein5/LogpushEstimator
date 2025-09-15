package database

import (
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"
)

func TestNewSQLiteController(t *testing.T) {
	// Test with temporary database
	tempFile := "test_logpush.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	controller, err := NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create SQLiteController: %v", err)
	}
	defer controller.Close()

	if controller.db == nil {
		t.Error("Database connection should not be nil")
	}

	if controller.logger == nil {
		t.Error("Logger should not be nil")
	}
}

func TestNewSQLiteControllerDefaultPath(t *testing.T) {
	// Test with default path
	defer os.Remove("logpush.db")

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	controller, err := NewSQLiteController("", logger)
	if err != nil {
		t.Fatalf("Failed to create SQLiteController with default path: %v", err)
	}
	defer controller.Close()

	if controller.db == nil {
		t.Error("Database connection should not be nil")
	}
}

func TestNewSQLiteControllerNilLogger(t *testing.T) {
	// Test with nil logger
	tempFile := "test_logpush_nil_logger.db"
	defer os.Remove(tempFile)

	controller, err := NewSQLiteController(tempFile, nil)
	if err != nil {
		t.Fatalf("Failed to create SQLiteController with nil logger: %v", err)
	}
	defer controller.Close()

	if controller.logger == nil {
		t.Error("Logger should not be nil even when passed nil")
	}
}

func TestInsertLogSize(t *testing.T) {
	tempFile := "test_insert.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	controller, err := NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create SQLiteController: %v", err)
	}
	defer controller.Close()

	// Test inserting a log size
	filesize := int64(1024)
	err = controller.InsertLogSize(filesize)
	if err != nil {
		t.Fatalf("Failed to insert log size: %v", err)
	}

	// Verify the insertion by querying all records
	logSizes, err := controller.GetAll()
	if err != nil {
		t.Fatalf("Failed to query log sizes: %v", err)
	}

	if len(logSizes) != 1 {
		t.Fatalf("Expected 1 log size, got %d", len(logSizes))
	}

	if logSizes[0].Filesize != filesize {
		t.Errorf("Expected filesize %d, got %d", filesize, logSizes[0].Filesize)
	}

	// Check that timestamp is recent (within 1 second)
	now := time.Now()
	timeDiff := now.Sub(logSizes[0].Timestamp)
	if timeDiff < 0 || timeDiff > time.Second {
		t.Errorf("Timestamp should be recent, got %v (diff: %v)", logSizes[0].Timestamp, timeDiff)
	}
}

func TestInsertLogSizeZero(t *testing.T) {
	tempFile := "test_insert_zero.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	controller, err := NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create SQLiteController: %v", err)
	}
	defer controller.Close()

	// Test inserting zero filesize (should still work)
	err = controller.InsertLogSize(0)
	if err != nil {
		t.Fatalf("Failed to insert zero log size: %v", err)
	}

	logSizes, err := controller.GetAll()
	if err != nil {
		t.Fatalf("Failed to query log sizes: %v", err)
	}

	if len(logSizes) != 1 {
		t.Fatalf("Expected 1 log size, got %d", len(logSizes))
	}

	if logSizes[0].Filesize != 0 {
		t.Errorf("Expected filesize 0, got %d", logSizes[0].Filesize)
	}
}

func TestGetAll(t *testing.T) {
	tempFile := "test_get_all.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	controller, err := NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create SQLiteController: %v", err)
	}
	defer controller.Close()

	// Insert multiple log sizes
	filesizes := []int64{100, 200, 300, 400, 500}
	for _, size := range filesizes {
		err = controller.InsertLogSize(size)
		if err != nil {
			t.Fatalf("Failed to insert log size %d: %v", size, err)
		}
		time.Sleep(1 * time.Millisecond) // Small delay to ensure different timestamps
	}

	// Query all
	logSizes, err := controller.GetAll()
	if err != nil {
		t.Fatalf("Failed to get all log sizes: %v", err)
	}

	if len(logSizes) != len(filesizes) {
		t.Fatalf("Expected %d log sizes, got %d", len(filesizes), len(logSizes))
	}

	// Verify sizes are returned in insertion order (ID ascending)
	for i, logSize := range logSizes {
		if logSize.Filesize != filesizes[i] {
			t.Errorf("Expected filesize %d at index %d, got %d", filesizes[i], i, logSize.Filesize)
		}
	}
}

func TestQueryByTimeRange(t *testing.T) {
	tempFile := "test_query_range.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	controller, err := NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create SQLiteController: %v", err)
	}
	defer controller.Close()

	// Insert some log sizes
	baseTime := time.Now().Add(-2 * time.Hour)

	// Insert logs at different times using direct SQL to control timestamps
	_, err = controller.db.Exec(`INSERT INTO log_sizes (timestamp, filesize) VALUES (?, ?)`,
		baseTime.Add(-30*time.Minute), 100)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	_, err = controller.db.Exec(`INSERT INTO log_sizes (timestamp, filesize) VALUES (?, ?)`,
		baseTime.Add(-15*time.Minute), 200)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	_, err = controller.db.Exec(`INSERT INTO log_sizes (timestamp, filesize) VALUES (?, ?)`,
		baseTime.Add(15*time.Minute), 300)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Query within a time range that should include only the middle record
	startTime := baseTime.Add(-20 * time.Minute)
	endTime := baseTime.Add(-10 * time.Minute)

	logSizes, err := controller.QueryByTimeRange(startTime, endTime)
	if err != nil {
		t.Fatalf("Failed to query log sizes by time range: %v", err)
	}

	if len(logSizes) != 1 {
		t.Fatalf("Expected 1 log size in time range, got %d", len(logSizes))
	}

	if logSizes[0].Filesize != 200 {
		t.Errorf("Expected filesize 200, got %d", logSizes[0].Filesize)
	}
}

func TestQueryByTimeRangeEmpty(t *testing.T) {
	tempFile := "test_query_range_empty.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	controller, err := NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create SQLiteController: %v", err)
	}
	defer controller.Close()

	// Query empty database
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	logSizes, err := controller.QueryByTimeRange(startTime, endTime)
	if err != nil {
		t.Fatalf("Failed to query empty database: %v", err)
	}

	if len(logSizes) != 0 {
		t.Fatalf("Expected 0 log sizes in empty database, got %d", len(logSizes))
	}
}

func TestClose(t *testing.T) {
	tempFile := "test_close.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	controller, err := NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create SQLiteController: %v", err)
	}

	// Close should not return an error
	err = controller.Close()
	if err != nil {
		t.Errorf("Close returned an error: %v", err)
	}

	// Second close should still not error (idempotent)
	err = controller.Close()
	if err != nil {
		t.Errorf("Second close returned an error: %v", err)
	}
}

func TestConcurrentInserts(t *testing.T) {
	tempFile := "test_concurrent.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	controller, err := NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create SQLiteController: %v", err)
	}
	defer controller.Close()

	// Test concurrent inserts
	numGoroutines := 10
	insertsPerGoroutine := 5

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*insertsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < insertsPerGoroutine; j++ {
				filesize := int64(goroutineID*100 + j)
				err := controller.InsertLogSize(filesize)
				if err != nil {
					errChan <- err
					return
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Check for errors
	close(errChan)
	for err := range errChan {
		if err != nil {
			t.Fatalf("Concurrent insert failed: %v", err)
		}
	}

	// Verify all inserts completed
	logSizes, err := controller.GetAll()
	if err != nil {
		t.Fatalf("Failed to query after concurrent inserts: %v", err)
	}

	expectedCount := numGoroutines * insertsPerGoroutine
	if len(logSizes) != expectedCount {
		t.Errorf("Expected %d log sizes after concurrent inserts, got %d", expectedCount, len(logSizes))
	}
}

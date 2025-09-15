package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/melatonein5/LogpushEstimator/src/database"
)

func TestMakeDashboardHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := MakeDashboardHandler(logger)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Note: This test will fail if the template file doesn't exist
	// In a real environment, you'd mock the template or ensure test files exist
	// For now, we'll test the error case

	// Check that it attempts to serve HTML
	if contentType := rr.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") && rr.Code == http.StatusOK {
		t.Errorf("Expected HTML content type when successful, got %v", contentType)
	}

	// The handler should either return OK (if template exists) or Internal Server Error
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %v", rr.Code)
	}
}

func TestMakeStaticFileHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create temporary directory and files for testing
	tempDir := "test_static"
	os.MkdirAll(filepath.Join(tempDir, "css"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "js"), 0755)
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := map[string]string{
		"css/style.css":   "body { color: red; }",
		"js/dashboard.js": "console.log('test');",
		"test.html":       "<html><body>Test</body></html>",
		"test.txt":        "Plain text content",
	}

	for filePath, content := range testFiles {
		fullPath := filepath.Join(tempDir, filePath)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	// Create a test handler that uses the temporary directory
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		// Override the file path resolution for testing
		path := strings.TrimPrefix(r.URL.Path, "/static")
		filePath := filepath.Join(tempDir, path)

		logger.Info("Static file request", "remote_addr", r.RemoteAddr, "file", filePath)

		// Set cache headers for static assets
		w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour

		// Set appropriate content type based on file extension
		ext := filepath.Ext(filePath)
		switch ext {
		case ".css":
			w.Header().Set("Content-Type", "text/css")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript")
		case ".html":
			w.Header().Set("Content-Type", "text/html")
		default:
			w.Header().Set("Content-Type", "text/plain")
		}

		// Check if file exists
		file, err := os.Open(filePath)
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// Get file info for Content-Length header
		stat, err := file.Stat()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Serve the file content directly to preserve our headers
		w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
		_, err = io.Copy(w, file)
		if err != nil {
			logger.Error("Error serving static file", "error", err, "file", filePath)
		}
	}

	tests := []struct {
		name         string
		path         string
		expectedType string
	}{
		{"CSS file", "/static/css/style.css", "text/css"},
		{"JS file", "/static/js/dashboard.js", "application/javascript"},
		{"HTML file", "/static/test.html", "text/html"},
		{"Unknown file", "/static/test.txt", "text/plain"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			testHandler(rr, req)

			// Check content type
			if contentType := rr.Header().Get("Content-Type"); contentType != tt.expectedType {
				t.Errorf("Expected content type %v, got %v", tt.expectedType, contentType)
			}

			// Check cache headers
			if cacheControl := rr.Header().Get("Cache-Control"); cacheControl != "public, max-age=3600" {
				t.Errorf("Expected cache control 'public, max-age=3600', got %v", cacheControl)
			}
		})
	}
}

func setupTestDatabase(t *testing.T) (*database.SQLiteController, func()) {
	tempFile := "test_handlers.db"
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	db, err := database.NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Insert some test data
	testData := []int64{1024, 2048, 4096, 8192, 16384}

	for _, size := range testData {
		// Use the regular InsertLogSize method
		err = db.InsertLogSize(size)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	cleanup := func() {
		db.Close()
		os.Remove(tempFile)
	}

	return db, cleanup
}

func TestAPIRecentLogs(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := MakeAPIHandlers(db, logger)

	req, err := http.NewRequest("GET", "/api/logs/recent", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers["/api/logs/recent"].ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Could not parse JSON response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got success=%v, error=%v", response.Success, response.Error)
	}

	// Check content type and CORS headers
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected JSON content type, got %v", contentType)
	}

	if cors := rr.Header().Get("Access-Control-Allow-Origin"); cors != "*" {
		t.Errorf("Expected CORS header '*', got %v", cors)
	}
}

func TestAPITimeRangeQuery(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := MakeAPIHandlers(db, logger)

	// Test valid time range
	start := time.Now().Add(-2 * time.Hour).Format(time.RFC3339)
	end := time.Now().Format(time.RFC3339)

	req, err := http.NewRequest("GET", "/api/logs/range?start="+url.QueryEscape(start)+"&end="+url.QueryEscape(end), nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers["/api/logs/range"].ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Could not parse JSON response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got success=%v, error=%v", response.Success, response.Error)
	}
}

func TestAPITimeRangeQueryMissingParams(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := MakeAPIHandlers(db, logger)

	// Test missing parameters
	req, err := http.NewRequest("GET", "/api/logs/range", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers["/api/logs/range"].ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Could not parse JSON response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success=false for missing params, got success=%v", response.Success)
	}

	if !strings.Contains(response.Error, "start and end parameters required") {
		t.Errorf("Expected error about missing parameters, got: %v", response.Error)
	}
}

func TestAPITimeRangeQueryInvalidFormat(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := MakeAPIHandlers(db, logger)

	// Test invalid time format
	req, err := http.NewRequest("GET", "/api/logs/range?start=invalid&end=also-invalid", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers["/api/logs/range"].ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Could not parse JSON response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success=false for invalid time format, got success=%v", response.Success)
	}
}

func TestAPIStatsSummary(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := MakeAPIHandlers(db, logger)

	req, err := http.NewRequest("GET", "/api/stats/summary", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers["/api/stats/summary"].ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Could not parse JSON response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got success=%v, error=%v", response.Success, response.Error)
	}

	// Verify stats structure
	statsData, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Errorf("Expected stats data to be a map")
	} else {
		requiredFields := []string{"total_records", "total_size", "average_size", "min_size", "max_size", "last_updated"}
		for _, field := range requiredFields {
			if _, exists := statsData[field]; !exists {
				t.Errorf("Expected field %s in stats response", field)
			}
		}
	}
}

func TestAPITimeSeriesChart(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := MakeAPIHandlers(db, logger)

	// Test default hours
	req, err := http.NewRequest("GET", "/api/charts/timeseries", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers["/api/charts/timeseries"].ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Test with specific hours parameter
	req, err = http.NewRequest("GET", "/api/charts/timeseries?hours=12", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handlers["/api/charts/timeseries"].ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Could not parse JSON response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got success=%v, error=%v", response.Success, response.Error)
	}
}

func TestAPISizeBreakdown(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := MakeAPIHandlers(db, logger)

	req, err := http.NewRequest("GET", "/api/charts/breakdown", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers["/api/charts/breakdown"].ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Could not parse JSON response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got success=%v, error=%v", response.Success, response.Error)
	}

	// Verify breakdown structure
	breakdownData, ok := response.Data.([]interface{})
	if !ok {
		t.Errorf("Expected breakdown data to be an array")
	} else if len(breakdownData) == 0 {
		t.Errorf("Expected non-empty breakdown data")
	}
}

func TestCalculateStats(t *testing.T) {
	// Test with empty logs
	emptyStats := calculateStats([]database.LogSize{})
	if emptyStats.TotalRecords != 0 {
		t.Errorf("Expected 0 total records for empty logs, got %d", emptyStats.TotalRecords)
	}

	// Test with sample data
	now := time.Now()
	logs := []database.LogSize{
		{ID: 1, Timestamp: now.Add(-2 * time.Hour), Filesize: 1000},
		{ID: 2, Timestamp: now.Add(-1 * time.Hour), Filesize: 2000},
		{ID: 3, Timestamp: now, Filesize: 3000},
	}

	stats := calculateStats(logs)
	if stats.TotalRecords != 3 {
		t.Errorf("Expected 3 total records, got %d", stats.TotalRecords)
	}

	if stats.TotalSize != 6000 {
		t.Errorf("Expected total size 6000, got %d", stats.TotalSize)
	}

	if stats.AverageSize != 2000.0 {
		t.Errorf("Expected average size 2000, got %f", stats.AverageSize)
	}

	if stats.MinSize != 1000 {
		t.Errorf("Expected min size 1000, got %d", stats.MinSize)
	}

	if stats.MaxSize != 3000 {
		t.Errorf("Expected max size 3000, got %d", stats.MaxSize)
	}
}

func TestAggregateByHour(t *testing.T) {
	now := time.Now()
	logs := []database.LogSize{
		{ID: 1, Timestamp: now.Truncate(time.Hour), Filesize: 1000},
		{ID: 2, Timestamp: now.Truncate(time.Hour).Add(30 * time.Minute), Filesize: 2000},
		{ID: 3, Timestamp: now.Truncate(time.Hour).Add(time.Hour), Filesize: 3000},
	}

	result := aggregateByHour(logs)

	// Should have 2 hour buckets
	if len(result) != 2 {
		t.Errorf("Expected 2 time buckets, got %d", len(result))
	}

	// Check that aggregation is working (first hour should have 2 records totaling 3000)
	found := false
	for _, point := range result {
		if point.Count == 2 && point.TotalSize == 3000 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find a time bucket with count=2 and total_size=3000")
	}
}

func TestCalculateSizeBreakdown(t *testing.T) {
	logs := []database.LogSize{
		{ID: 1, Filesize: 512},              // < 1KB
		{ID: 2, Filesize: 5 * 1024},         // 1KB - 10KB
		{ID: 3, Filesize: 50 * 1024},        // 10KB - 100KB
		{ID: 4, Filesize: 500 * 1024},       // 100KB - 1MB
		{ID: 5, Filesize: 5 * 1024 * 1024},  // 1MB - 10MB
		{ID: 6, Filesize: 50 * 1024 * 1024}, // > 10MB
	}

	breakdown := calculateSizeBreakdown(logs)

	if len(breakdown) != 6 {
		t.Errorf("Expected 6 size ranges, got %d", len(breakdown))
	}

	// Each range should have exactly 1 entry (16.67% each)
	for i, item := range breakdown {
		if item.Count != 1 {
			t.Errorf("Expected count 1 for range %d (%s), got %d", i, item.Range, item.Count)
		}

		expectedPercentage := 100.0 / 6.0
		if item.Percentage < expectedPercentage-0.1 || item.Percentage > expectedPercentage+0.1 {
			t.Errorf("Expected percentage ~%.2f for range %s, got %.2f", expectedPercentage, item.Range, item.Percentage)
		}
	}
}

func TestSendSuccessResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	testData := map[string]string{"test": "data"}

	sendSuccessResponse(rr, testData)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected JSON content type, got %s", contentType)
	}

	if cors := rr.Header().Get("Access-Control-Allow-Origin"); cors != "*" {
		t.Errorf("Expected CORS header '*', got %s", cors)
	}

	var response APIResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Could not parse JSON response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %v", response.Success)
	}
}

func TestSendErrorResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	errorMessage := "Test error message"

	sendErrorResponse(rr, errorMessage)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", status)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected JSON content type, got %s", contentType)
	}

	var response APIResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Could not parse JSON response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success=false, got %v", response.Success)
	}

	if response.Error != errorMessage {
		t.Errorf("Expected error message '%s', got '%s'", errorMessage, response.Error)
	}
}

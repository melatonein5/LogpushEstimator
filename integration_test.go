package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/melatonein5/LogpushEstimator/src/database"
	"github.com/melatonein5/LogpushEstimator/src/gui/handlers"
)

// Integration tests that test the complete application flow
func TestFullApplicationFlow(t *testing.T) {
	// Create temporary database for testing
	tempFile := "test_integration.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	db, err := database.NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create test servers
	ingestionServer := createIngestionServer(db)
	guiServer := createGUIServer(db)

	ingestionTestServer := httptest.NewServer(ingestionServer.Handler)
	defer ingestionTestServer.Close()

	guiTestServer := httptest.NewServer(guiServer.Handler)
	defer guiTestServer.Close()

	// Step 1: Check health endpoints
	t.Run("Health Check", func(t *testing.T) {
		resp, err := http.Get(ingestionTestServer.URL + "/health")
		if err != nil {
			t.Fatalf("Failed to call ingestion health endpoint: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Ingestion health endpoint returned status %d, expected %d", resp.StatusCode, http.StatusOK)
		}
	})

	// Step 2: Ingest some data
	t.Run("Data Ingestion", func(t *testing.T) {
		testData := []string{
			"Small log entry",
			strings.Repeat("Medium log entry ", 100),
			strings.Repeat("Large log entry ", 1000),
		}

		for i, data := range testData {
			resp, err := http.Post(ingestionTestServer.URL+"/ingest", "text/plain", bytes.NewBufferString(data))
			if err != nil {
				t.Fatalf("Failed to ingest data %d: %v", i, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Ingestion endpoint returned status %d for data %d, expected %d", resp.StatusCode, i, http.StatusOK)
			}

			// Small delay to ensure different timestamps
			time.Sleep(10 * time.Millisecond)
		}
	})

	// Step 3: Verify data via API endpoints
	t.Run("API Data Retrieval", func(t *testing.T) {
		// Test summary stats
		resp, err := http.Get(guiTestServer.URL + "/api/stats/summary")
		if err != nil {
			t.Fatalf("Failed to call stats summary: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Stats summary returned status %d, expected %d", resp.StatusCode, http.StatusOK)
		}

		var statsResponse handlers.APIResponse
		err = json.NewDecoder(resp.Body).Decode(&statsResponse)
		if err != nil {
			t.Errorf("Failed to decode stats response: %v", err)
		}

		if !statsResponse.Success {
			t.Errorf("Stats API returned success=false: %v", statsResponse.Error)
		}

		// Test recent logs
		resp, err = http.Get(guiTestServer.URL + "/api/logs/recent")
		if err != nil {
			t.Fatalf("Failed to call recent logs: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Recent logs returned status %d, expected %d", resp.StatusCode, http.StatusOK)
		}

		var logsResponse handlers.APIResponse
		err = json.NewDecoder(resp.Body).Decode(&logsResponse)
		if err != nil {
			t.Errorf("Failed to decode logs response: %v", err)
		}

		if !logsResponse.Success {
			t.Errorf("Logs API returned success=false: %v", logsResponse.Error)
		}

		// Test time series
		resp, err = http.Get(guiTestServer.URL + "/api/charts/timeseries?hours=24")
		if err != nil {
			t.Fatalf("Failed to call time series: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Time series returned status %d, expected %d", resp.StatusCode, http.StatusOK)
		}

		// Test size breakdown
		resp, err = http.Get(guiTestServer.URL + "/api/charts/breakdown")
		if err != nil {
			t.Fatalf("Failed to call size breakdown: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Size breakdown returned status %d, expected %d", resp.StatusCode, http.StatusOK)
		}
	})

	// Step 4: Test static file serving
	t.Run("Static File Serving", func(t *testing.T) {
		// Note: These will return 404 if files don't exist, but should have correct headers
		staticFiles := []string{
			"/static/css/style.css",
			"/static/js/dashboard.js",
		}

		for _, file := range staticFiles {
			resp, err := http.Get(guiTestServer.URL + file)
			if err != nil {
				t.Fatalf("Failed to request static file %s: %v", file, err)
			}
			defer resp.Body.Close()

			// Check cache headers are set (regardless of file existence)
			if cacheControl := resp.Header.Get("Cache-Control"); cacheControl != "public, max-age=3600" {
				t.Errorf("Expected cache control header for %s, got %s", file, cacheControl)
			}
		}
	})
}

func TestConcurrentIngestAndQuery(t *testing.T) {
	// Create temporary database for testing
	tempFile := "test_concurrent_integration.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	db, err := database.NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create test servers
	ingestionServer := createIngestionServer(db)
	guiServer := createGUIServer(db)

	ingestionTestServer := httptest.NewServer(ingestionServer.Handler)
	defer ingestionTestServer.Close()

	guiTestServer := httptest.NewServer(guiServer.Handler)
	defer guiTestServer.Close()

	// Start concurrent ingestion
	numIngesters := 5
	ingestionsPerGoroutine := 10
	var wg sync.WaitGroup
	errChan := make(chan error, numIngesters*ingestionsPerGoroutine)

	wg.Add(numIngesters)
	for i := 0; i < numIngesters; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < ingestionsPerGoroutine; j++ {
				data := strings.Repeat("x", 100*(goroutineID*10+j+1))
				resp, err := http.Post(ingestionTestServer.URL+"/ingest", "text/plain", bytes.NewBufferString(data))
				if err != nil {
					errChan <- err
					return
				}
				resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					errChan <- err
					return
				}
			}
		}(i)
	}

	// Concurrent API queries
	numQueriers := 3
	wg.Add(numQueriers)
	for i := 0; i < numQueriers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				resp, err := http.Get(guiTestServer.URL + "/api/stats/summary")
				if err != nil {
					errChan <- err
					return
				}
				resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					errChan <- err
					return
				}

				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	// Wait for operations to complete
	wg.Wait()

	// Check for errors
	select {
	case err := <-errChan:
		t.Fatalf("Concurrent operation failed: %v", err)
	default:
		// No errors, verify final state
	}

	// Verify final database state
	logs, err := db.GetAll()
	if err != nil {
		t.Fatalf("Failed to query final state: %v", err)
	}

	expectedCount := numIngesters * ingestionsPerGoroutine
	if len(logs) < expectedCount {
		t.Errorf("Expected at least %d log entries, got %d", expectedCount, len(logs))
	}
}

func TestAPITimeRangeIntegration(t *testing.T) {
	// Create temporary database for testing
	tempFile := "test_time_range_integration.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	db, err := database.NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create servers
	ingestionServer := createIngestionServer(db)
	guiServer := createGUIServer(db)

	// Insert test data by using the API
	ingestionTestServer := httptest.NewServer(ingestionServer.Handler)
	defer ingestionTestServer.Close()

	testSizes := []int{1000, 2000, 3000, 4000, 5000, 6000}

	for i, size := range testSizes {
		data := strings.Repeat("x", size)
		resp, err := http.Post(ingestionTestServer.URL+"/ingest", "text/plain", bytes.NewBufferString(data))
		if err != nil {
			t.Fatalf("Failed to ingest test data %d: %v", i, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to ingest test data %d, status: %d", i, resp.StatusCode)
		}

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Create GUI server
	guiTestServer := httptest.NewServer(guiServer.Handler)
	defer guiTestServer.Close()

	// Test time range query - get data from last hour to next hour (wider range)
	now := time.Now()
	startTime := now.Add(-1 * time.Hour)
	endTime := now.Add(1 * time.Hour)

	queryURL := guiTestServer.URL + "/api/logs/range?start=" + url.QueryEscape(startTime.Format(time.RFC3339)) + "&end=" + url.QueryEscape(endTime.Format(time.RFC3339))

	resp, err := http.Get(queryURL)
	if err != nil {
		t.Fatalf("Failed to call time range API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Time range API returned status %d, expected %d", resp.StatusCode, http.StatusOK)
	}

	var response handlers.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Errorf("API returned success=false: %v", response.Error)
	}

	// Check if response data is nil
	if response.Data == nil {
		t.Errorf("API returned nil data")
		return
	}

	// The response data should be a slice of records
	responseData, ok := response.Data.([]interface{})
	if !ok {
		t.Errorf("API response data is not a slice, got type: %T", response.Data)
		return
	}

	if len(responseData) != len(testSizes) {
		t.Errorf("Expected %d records in time range, got %d", len(testSizes), len(responseData))
	}
}

func TestErrorHandling(t *testing.T) {
	// Create temporary database for testing
	tempFile := "test_error_handling.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	db, err := database.NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create servers
	ingestionServer := createIngestionServer(db)
	guiServer := createGUIServer(db)

	ingestionTestServer := httptest.NewServer(ingestionServer.Handler)
	defer ingestionTestServer.Close()

	guiTestServer := httptest.NewServer(guiServer.Handler)
	defer guiTestServer.Close()

	// Test invalid HTTP methods
	t.Run("Invalid Methods", func(t *testing.T) {
		// GET request to ingest endpoint should fail
		resp, err := http.Get(ingestionTestServer.URL + "/ingest")
		if err != nil {
			t.Fatalf("Failed to make GET request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405 for GET on ingest endpoint, got %d", resp.StatusCode)
		}

		// PUT request to ingest endpoint should fail
		req, _ := http.NewRequest("PUT", ingestionTestServer.URL+"/ingest", strings.NewReader("test"))
		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make PUT request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405 for PUT on ingest endpoint, got %d", resp.StatusCode)
		}
	})

	// Test empty request body
	t.Run("Empty Request Body", func(t *testing.T) {
		resp, err := http.Post(ingestionTestServer.URL+"/ingest", "text/plain", strings.NewReader(""))
		if err != nil {
			t.Fatalf("Failed to post empty body: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400 for empty body, got %d", resp.StatusCode)
		}
	})

	// Test invalid API endpoints
	t.Run("Invalid API Endpoints", func(t *testing.T) {
		resp, err := http.Get(guiTestServer.URL + "/api/nonexistent")
		if err != nil {
			t.Fatalf("Failed to call nonexistent endpoint: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404 for nonexistent endpoint, got %d", resp.StatusCode)
		}
	})

	// Test malformed time range parameters
	t.Run("Malformed Time Parameters", func(t *testing.T) {
		resp, err := http.Get(guiTestServer.URL + "/api/logs/range?start=invalid&end=also-invalid")
		if err != nil {
			t.Fatalf("Failed to call API with invalid time: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected 500 for invalid time format, got %d", resp.StatusCode)
		}

		var response handlers.APIResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			t.Errorf("Failed to decode error response: %v", err)
		}

		if response.Success {
			t.Errorf("Expected success=false for invalid time format")
		}
	})
}

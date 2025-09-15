package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/melatonein5/LogpushEstimator/src/database"
)

func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)

	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the content type
	expected := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expected {
		t.Errorf("handler returned wrong content type: got %v want %v",
			contentType, expected)
	}

	// Check the response body
	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Could not parse JSON response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%v'", response["status"])
	}

	if response["service"] != "LogpushEstimator" {
		t.Errorf("Expected service 'LogpushEstimator', got '%v'", response["service"])
	}
}

func TestMakeIngestionHandler(t *testing.T) {
	// Create temporary database for testing
	tempFile := "test_ingestion.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	db, err := database.NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	handler := makeIngestionHandler(db)

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid POST request",
			method:         "POST",
			body:           "test log data",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "Invalid GET request",
			method:         "GET",
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
		{
			name:           "Empty POST request",
			method:         "POST",
			body:           "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Request body cannot be empty",
		},
		{
			name:           "Large POST request",
			method:         "POST",
			body:           strings.Repeat("x", 10000),
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, "/ingest", strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			if body := strings.TrimSpace(rr.Body.String()); body != tt.expectedBody {
				t.Errorf("handler returned unexpected body: got %v want %v",
					body, tt.expectedBody)
			}
		})
	}
}

func TestMakeIngestionHandlerDatabaseInteraction(t *testing.T) {
	// Create temporary database for testing
	tempFile := "test_ingestion_db.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	db, err := database.NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	handler := makeIngestionHandler(db)

	// Send a valid request
	testData := "This is test log data"
	req, err := http.NewRequest("POST", "/ingest", strings.NewReader(testData))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Check response
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Verify data was inserted into database
	logSizes, err := db.GetAll()
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if len(logSizes) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logSizes))
	}

	expectedSize := int64(len(testData))
	if logSizes[0].Filesize != expectedSize {
		t.Errorf("Expected filesize %d, got %d", expectedSize, logSizes[0].Filesize)
	}
}

func TestCreateIngestionServer(t *testing.T) {
	// Create temporary database for testing
	tempFile := "test_create_ingestion.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	db, err := database.NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	server := createIngestionServer(db)

	if server == nil {
		t.Error("createIngestionServer returned nil")
	}

	if server.Addr != ingestionPort {
		t.Errorf("Expected server address %s, got %s", ingestionPort, server.Addr)
	}

	if server.Handler == nil {
		t.Error("Server handler should not be nil")
	}
}

func TestCreateGUIServer(t *testing.T) {
	// Create temporary database for testing
	tempFile := "test_create_gui.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	db, err := database.NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	server := createGUIServer(db)

	if server == nil {
		t.Error("createGUIServer returned nil")
	}

	if server.Addr != guiPort {
		t.Errorf("Expected server address %s, got %s", guiPort, server.Addr)
	}

	if server.Handler == nil {
		t.Error("Server handler should not be nil")
	}
}

func TestIngestionHandlerWithRealRequests(t *testing.T) {
	// Create temporary database for testing
	tempFile := "test_real_requests.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	db, err := database.NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create test server
	server := createIngestionServer(db)
	testServer := httptest.NewServer(server.Handler)
	defer testServer.Close()

	// Test health endpoint
	resp, err := http.Get(testServer.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health endpoint returned status %d, expected %d", resp.StatusCode, http.StatusOK)
	}

	// Test ingestion endpoint
	testData := "Sample log data for testing"
	resp, err = http.Post(testServer.URL+"/ingest", "text/plain", bytes.NewBufferString(testData))
	if err != nil {
		t.Fatalf("Failed to call ingest endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Ingest endpoint returned status %d, expected %d", resp.StatusCode, http.StatusOK)
	}

	// Verify data was stored
	logSizes, err := db.GetAll()
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if len(logSizes) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logSizes))
	}

	expectedSize := int64(len(testData))
	if logSizes[0].Filesize != expectedSize {
		t.Errorf("Expected filesize %d, got %d", expectedSize, logSizes[0].Filesize)
	}
}

func TestIngestionHandlerConcurrency(t *testing.T) {
	// Create temporary database for testing
	tempFile := "test_concurrency.db"
	defer os.Remove(tempFile)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	db, err := database.NewSQLiteController(tempFile, logger)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	handler := makeIngestionHandler(db)

	// Test concurrent requests
	numRequests := 100
	var wg sync.WaitGroup
	errChan := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()

			testData := "test data " + string(rune(requestID))
			req, err := http.NewRequest("POST", "/ingest", strings.NewReader(testData))
			if err != nil {
				errChan <- err
				return
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				errChan <- fmt.Errorf("Request %d failed with status %d", requestID, rr.Code)
				return
			}
		}(i)
	}

	// Wait for all requests to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			t.Fatalf("Concurrent request failed: %v", err)
		}
	}

	// Verify all requests were processed
	logSizes, err := db.GetAll()
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if len(logSizes) < numRequests {
		t.Errorf("Expected at least %d log entries, got %d", numRequests, len(logSizes))
	}
}

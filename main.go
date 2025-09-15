// Package main implements LogpushEstimator, a Cloudflare log ingestion monitoring tool.
//
// LogpushEstimator is designed to help monitor and estimate data usage from Cloudflare's
// Logpush service by tracking log file sizes over time. It provides both an ingestion
// endpoint for receiving log data and a web-based dashboard for visualizing usage patterns.
//
// # Architecture
//
// The application consists of two main HTTP servers:
//
//   - Ingestion Server (port 8080): Receives log data via POST requests
//   - GUI Server (port 8081): Serves the web dashboard and API endpoints
//
// # Usage
//
// To start the LogpushEstimator:
//
//	go run main.go
//
// Or build and run:
//
//	go build -o logpush-estimator .
//	./logpush-estimator
//
// The application will start both servers and be ready to accept log data and serve
// the dashboard interface.
//
// # API Endpoints
//
// Ingestion Server (8080):
//   - POST /ingest - Accept log data for size tracking
//   - GET /health - Health check endpoint
//
// GUI Server (8081):
//   - GET / - Dashboard interface
//   - GET /api/stats/summary - Summary statistics
//   - GET /api/logs/recent - Recent log entries
//   - GET /api/logs/time-range - Time-filtered log data
//   - GET /api/charts/time-series - Time series chart data
//   - GET /api/charts/size-breakdown - Size breakdown chart data
//   - GET /static/* - Static assets (CSS, JS, images)
//
// # Data Storage
//
// LogpushEstimator uses SQLite for data persistence, storing log size records
// with timestamps for analysis and visualization.
package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/melatonein5/LogpushEstimator/src/database"
	"github.com/melatonein5/LogpushEstimator/src/gui/handlers"
)

// Default server configuration
var (
	// ingestionPort specifies the port for the log ingestion server
	ingestionPort = ":8080"
	// guiPort specifies the port for the web dashboard server
	guiPort = ":8081"
)

// slogger provides structured logging throughout the application
var slogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

// healthHandler provides a health check endpoint that returns service status.
// It responds with a JSON object containing the service status and name.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	slogger.Info("Health check request", "remote_addr", r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]string{
		"status":  "ok",
		"service": "LogpushEstimator",
	}
	json.NewEncoder(w).Encode(response)
}

// makeIngestionHandler creates an HTTP handler for log data ingestion.
// It accepts POST requests containing log data and stores the payload size
// along with a timestamp in the database for monitoring purposes.
//
// The handler validates the HTTP method (must be POST), reads the request body,
// measures its size, and stores this information in the database using the
// provided SQLiteController.
//
// Returns appropriate HTTP status codes:
//   - 200 OK: Successfully processed and stored the log data
//   - 400 Bad Request: Empty body or failed to read body
//   - 405 Method Not Allowed: Non-POST requests
//   - 500 Internal Server Error: Database insertion failures
func makeIngestionHandler(db *database.SQLiteController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slogger.Info("Ingestion request received",
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"content_length", r.ContentLength)

		if r.Method != http.MethodPost {
			slogger.Warn("Invalid HTTP method", "method", r.Method, "remote_addr", r.RemoteAddr)
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method not allowed"))
			return
		}

		// Read the entire request body to measure its size
		body, err := io.ReadAll(r.Body)
		if err != nil {
			slogger.Error("Failed to read request body", "error", err, "remote_addr", r.RemoteAddr)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Failed to read request body"))
			return
		}
		defer r.Body.Close()

		// Calculate the actual body size
		bodySize := int64(len(body))

		// Validate body size is positive (not empty)
		if bodySize <= 0 {
			slogger.Warn("Empty request body received", "body_size", bodySize, "remote_addr", r.RemoteAddr)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Request body cannot be empty"))
			return
		}

		// Insert the computed body size into database
		err = db.InsertLogSize(bodySize)
		if err != nil {
			slogger.Error("Failed to insert log size", "error", err, "body_size", bodySize, "remote_addr", r.RemoteAddr)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to write log size"))
			return
		}

		slogger.Info("Log size inserted successfully", "body_size", bodySize, "remote_addr", r.RemoteAddr)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}

// createIngestionServer creates and configures the HTTP server for log data ingestion.
// The server listens on the configured ingestion port and provides endpoints for
// receiving log data and health checks.
//
// Endpoints:
//   - POST /ingest: Accept log data for size tracking
//   - GET /health: Health check endpoint
func createIngestionServer(db *database.SQLiteController) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/ingest", makeIngestionHandler(db))
	mux.HandleFunc("/health", healthHandler)
	return &http.Server{
		Addr:    ingestionPort,
		Handler: mux,
	}
}

// createGUIServer creates and configures the HTTP server for the web dashboard.
// The server provides both the web interface and REST API endpoints for
// accessing stored log data and analytics.
//
// Endpoints:
//   - GET /: Main dashboard interface
//   - GET /dashboard: Alternative dashboard path
//   - GET /api/*: REST API endpoints for data access
//   - GET /static/*: Static assets (CSS, JS, images)
func createGUIServer(db *database.SQLiteController) *http.Server {
	mux := http.NewServeMux()

	// Dashboard routes (specific paths only)
	mux.HandleFunc("/dashboard", handlers.MakeDashboardHandler(slogger))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only serve dashboard for exact root path, otherwise 404
		if r.URL.Path == "/" {
			handlers.MakeDashboardHandler(slogger)(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	// API routes
	apiHandlers := handlers.MakeAPIHandlers(db, slogger)
	for path, handler := range apiHandlers {
		mux.HandleFunc(path, handler)
	}

	// Static file serving
	mux.HandleFunc("/static/", handlers.MakeStaticFileHandler(slogger))

	return &http.Server{
		Addr:    guiPort,
		Handler: mux,
	}
}

func main() {
	slogger.Info("Starting LogpushEstimator", "ingestion_port", ingestionPort, "gui_port", guiPort)

	db, err := database.NewSQLiteController("", slogger)
	if err != nil {
		slogger.Error("Failed to initialize SQLite database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slogger.Error("Failed to close database", "error", err)
		} else {
			slogger.Info("Database connection closed successfully")
		}
	}()

	slogger.Info("SQLite database initialized successfully", "path", "logpush.db")

	ingestionServer := createIngestionServer(db)
	guiServer := createGUIServer(db)

	slogger.Info("Starting HTTP servers")

	go func() {
		slogger.Info("Starting ingestion server", "port", ingestionPort)
		if err := ingestionServer.ListenAndServe(); err != nil {
			slogger.Error("Ingestion server failed", "error", err, "port", ingestionPort)
			os.Exit(1)
		}
	}()

	go func() {
		slogger.Info("Starting GUI server", "port", guiPort)
		if err := guiServer.ListenAndServe(); err != nil {
			slogger.Error("GUI server failed", "error", err, "port", guiPort)
			os.Exit(1)
		}
	}()

	slogger.Info("LogpushEstimator startup complete - servers running")
	select {}
}

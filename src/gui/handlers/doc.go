// Package handlers implements HTTP request handlers for the LogpushEstimator web application.
//
// This package provides a complete set of HTTP handlers for both the web dashboard
// interface and REST API endpoints. It bridges the gap between HTTP requests and
// database operations, offering both human-readable web interfaces and
// machine-readable API responses.
//
// # Handler Categories
//
// The package is organized into several handler categories:
//
//   - Dashboard Handlers: Serve HTML interfaces using templates
//   - API Handlers: Provide JSON REST endpoints for data access
//   - Static Handlers: Serve CSS, JavaScript, and other static assets
//
// # Examples
//
// Setting up a complete web server with all handlers:
//
//	package main
//
//	import (
//		"log"
//		"log/slog"
//		"net/http"
//		"os"
//
//		"github.com/melatonein5/LogpushEstimator/src/database"
//		"github.com/melatonein5/LogpushEstimator/src/gui/handlers"
//	)
//
//	func main() {
//		// Setup logger and database
//		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
//		db, err := database.NewSQLiteController("logpush.db", logger)
//		if err != nil {
//			log.Fatal(err)
//		}
//		defer db.Close()
//
//		// Create HTTP multiplexer
//		mux := http.NewServeMux()
//
//		// Add dashboard handler
//		mux.HandleFunc("/", handlers.MakeDashboardHandler(logger))
//
//		// Add all API handlers
//		apiHandlers := handlers.MakeAPIHandlers(db, logger)
//		for path, handler := range apiHandlers {
//			mux.HandleFunc(path, handler)
//		}
//
//		// Add static file handler
//		mux.HandleFunc("/static/", handlers.MakeStaticFileHandler(logger))
//
//		// Start server
//		log.Println("Server starting on :8081")
//		log.Fatal(http.ListenAndServe(":8081", mux))
//	}
//
// Using individual API handlers:
//
//	// Get summary statistics
//	resp, err := http.Get("http://localhost:8081/api/stats/summary")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer resp.Body.Close()
//
//	var apiResponse handlers.APIResponse
//	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
//		log.Fatal(err)
//	}
//
//	if apiResponse.Success {
//		stats := apiResponse.Data.(map[string]interface{})
//		fmt.Printf("Total records: %.0f\n", stats["total_records"])
//		fmt.Printf("Total size: %.0f bytes\n", stats["total_size"])
//	}
//
// Custom handler integration:
//
//	// Create a custom handler that uses the same response format
//	func customHandler(db *database.SQLiteController, logger *slog.Logger) http.HandlerFunc {
//		return func(w http.ResponseWriter, r *http.Request) {
//			logger.Info("Custom API request", "remote_addr", r.RemoteAddr)
//
//			// Your custom logic here
//			data := map[string]interface{}{
//				"message": "Custom endpoint response",
//				"timestamp": time.Now().Format(time.RFC3339),
//			}
//
//			// Use the package's response helpers
//			w.Header().Set("Content-Type", "application/json")
//			response := handlers.APIResponse{Success: true, Data: data}
//			json.NewEncoder(w).Encode(response)
//		}
//	}
//
// # API Response Format
//
// All API endpoints follow a consistent response structure:
//
//	// Successful response
//	{
//		"success": true,
//		"data": {
//			// Endpoint-specific data
//		}
//	}
//
//	// Error response
//	{
//		"success": false,
//		"error": "Error message description"
//	}
//
// # Static File Organization
//
// The static file handler expects assets to be organized as:
//
//	src/gui/static/
//	├── css/
//	│   └── style.css
//	├── js/
//	│   └── dashboard.js
//	└── images/
//	    └── logo.png
//
// Files are served with appropriate MIME types and caching headers.
//
// # Template System
//
// Dashboard handlers use Go's html/template package with templates
// stored in:
//
//	src/gui/templates/
//	└── dashboard.html
//
// Templates have access to request context and can include dynamic data.
//
// # CORS Support
//
// API handlers include CORS headers for development environments,
// allowing cross-origin requests from frontend applications.
//
// # Error Handling
//
// The package provides consistent error handling across all endpoints:
//
//   - HTTP status codes reflect the type of error
//   - Error messages are descriptive but don't expose internal details
//   - All errors are logged with structured context
//   - Client responses use the standard APIResponse format
package handlers

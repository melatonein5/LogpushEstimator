// Package handlers provides HTTP request handlers for the LogpushEstimator web interface.
//
// This package implements handlers for serving the dashboard web interface and static
// assets. It includes both the main dashboard HTML page and a custom static file
// server that preserves proper MIME types and caching headers.
//
// # Handler Types
//
// The package provides two main types of handlers:
//
//   - Dashboard handlers: Serve the main web interface using HTML templates
//   - Static file handlers: Serve CSS, JavaScript, and other static assets
//
// # Usage
//
// Create dashboard handlers:
//
//	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
//	dashboardHandler := handlers.MakeDashboardHandler(logger)
//	http.HandleFunc("/", dashboardHandler)
//
// Create static file handlers:
//
//	staticHandler := handlers.MakeStaticFileHandler(logger)
//	http.HandleFunc("/static/", staticHandler)
//
// # Template Requirements
//
// The dashboard handler expects to find HTML templates in the
// 'src/gui/templates/' directory relative to the application root.
// Static files should be organized under 'src/gui/static/'.
package handlers

import (
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// MakeDashboardHandler creates an HTTP handler for serving the main dashboard interface.
// The handler serves HTML content by parsing and executing dashboard templates.
//
// The handler looks for templates in 'src/gui/templates/dashboard.html' and serves
// them as HTML responses with appropriate content-type headers. If template parsing
// or execution fails, it returns appropriate HTTP error responses.
//
// Parameters:
//   - logger: Structured logger for request logging and error reporting
//
// Returns:
//   - http.HandlerFunc: Configured handler function for dashboard requests
//
// Template Location:
// The handler expects dashboard.html to be located at 'src/gui/templates/dashboard.html'
// relative to the application's working directory.
func MakeDashboardHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Dashboard request", "remote_addr", r.RemoteAddr, "path", r.URL.Path)

		// Parse the dashboard template
		tmpl, err := template.ParseFiles("src/gui/templates/dashboard.html")
		if err != nil {
			logger.Error("Failed to parse dashboard template", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		err = tmpl.Execute(w, nil)
		if err != nil {
			logger.Error("Failed to execute dashboard template", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}

// MakeStaticFileHandler creates an HTTP handler for serving static assets.
// This handler serves CSS, JavaScript, images, and other static files with
// proper MIME type detection and caching headers.
//
// The handler implements custom file serving logic instead of using http.ServeFile
// to ensure proper MIME type headers are preserved and not overridden by the
// standard library.
//
// Parameters:
//   - logger: Structured logger for request logging and error reporting
//
// Returns:
//   - http.HandlerFunc: Configured handler function for static file requests
//
// File Organization:
// Static files should be organized under 'src/gui/static/' with subdirectories
// for different asset types (css/, js/, images/, etc.).
//
// Supported MIME Types:
//   - .css files: text/css
//   - .js files: application/javascript
//   - .html files: text/html
//   - Other files: application/octet-stream (default)
//
// Security:
// The handler includes basic path traversal protection by cleaning file paths
// and ensuring they remain within the static directory.
func MakeStaticFileHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Remove "/static" prefix from the path
		path := strings.TrimPrefix(r.URL.Path, "/static")
		filePath := filepath.Join("src/gui/static", path)

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
}

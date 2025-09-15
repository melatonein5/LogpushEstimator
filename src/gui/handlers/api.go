// Package handlers provides REST API endpoints for the LogpushEstimator web application.
//
// This file implements HTTP handlers for serving log size data through RESTful APIs.
// The handlers provide various endpoints for retrieving statistical summaries,
// time-series data, and filtered log records for the web dashboard.
//
// # API Endpoints
//
// The package provides the following API endpoints:
//
//   - /api/stats/summary: Summary statistics (total records, sizes, averages)
//   - /api/logs/recent: Recent log entries (configurable limit)
//   - /api/logs/time-range: Time-filtered log data with query parameters
//   - /api/charts/time-series: Hourly aggregated data for time-series charts
//   - /api/charts/size-breakdown: Size distribution data for charts
//
// # Response Format
//
// All API responses follow a consistent JSON structure:
//
//	{
//		"success": true,
//		"data": {...},
//		"error": null
//	}
//
// Error responses include an error message and set success to false.
//
// # Usage
//
// Create API handlers:
//
//	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
//	db, _ := database.NewSQLiteController("logpush.db", logger)
//	apiHandlers := handlers.MakeAPIHandlers(db, logger)
//
//	for path, handler := range apiHandlers {
//		http.HandleFunc(path, handler)
//	}
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/melatonein5/LogpushEstimator/src/database"
)

// APIResponse wraps all API responses in a consistent format.
// This structure ensures uniform response handling across all API endpoints.
type APIResponse struct {
	Success bool        `json:"success"`         // Indicates if the request was successful
	Data    interface{} `json:"data,omitempty"`  // Response data (present on success)
	Error   string      `json:"error,omitempty"` // Error message (present on failure)
}

// LogSizeStats represents summary statistics for log size data.
// This structure provides comprehensive metrics about stored log records.
type LogSizeStats struct {
	TotalRecords int64   `json:"total_records"` // Total number of log records
	TotalSize    int64   `json:"total_size"`    // Sum of all log sizes in bytes
	AverageSize  float64 `json:"average_size"`  // Average log size in bytes
	MinSize      int64   `json:"min_size"`      // Smallest log size in bytes
	MaxSize      int64   `json:"max_size"`      // Largest log size in bytes
	LastUpdated  string  `json:"last_updated"`  // ISO timestamp of most recent record
}

// TimeSeriesPoint represents a single data point for time-series charts.
// This structure aggregates log data by time period for visualization.
type TimeSeriesPoint struct {
	Timestamp string `json:"timestamp"`  // ISO timestamp for the data point
	Count     int    `json:"count"`      // Number of log records in this time period
	TotalSize int64  `json:"total_size"` // Sum of log sizes in this time period
}

// SizeBreakdown represents file size distribution data for charts.
// SizeBreakdown represents file size distribution data for charts.
// This structure categorizes log records by size ranges for analytics.
type SizeBreakdown struct {
	Range      string  `json:"range"`      // Size range description (e.g., "1KB-10KB")
	Count      int     `json:"count"`      // Number of records in this size range
	Percentage float64 `json:"percentage"` // Percentage of total records
}

// MakeAPIHandlers creates and configures all API endpoint handlers.
// This function returns a map of URL paths to their corresponding HTTP handlers,
// providing a centralized way to register all API endpoints.
//
// Parameters:
//   - db: Database controller for data access
//   - logger: Structured logger for request logging
//
// Returns:
//   - map[string]http.HandlerFunc: Map of API paths to handler functions
//
// The returned map contains handlers for:
//   - /api/stats/summary: Statistical summary of all log data
//   - /api/logs/recent: Recent log entries (with optional limit parameter)
//   - /api/logs/time-range: Time-filtered log data (requires start/end parameters)
//   - /api/charts/time-series: Hourly aggregated data for charts
//   - /api/charts/size-breakdown: Size distribution analysis
func MakeAPIHandlers(db *database.SQLiteController, logger *slog.Logger) map[string]http.HandlerFunc {
	handlers := make(map[string]http.HandlerFunc)

	// Recent logs endpoint (last 24 hours)
	handlers["/api/logs/recent"] = func(w http.ResponseWriter, r *http.Request) {
		logger.Info("API request: recent logs", "remote_addr", r.RemoteAddr)

		end := time.Now()
		start := end.Add(-24 * time.Hour)

		logs, err := db.QueryByTimeRange(start, end)
		if err != nil {
			logger.Error("Failed to query recent logs", "error", err)
			sendErrorResponse(w, "Failed to fetch recent logs")
			return
		}

		sendSuccessResponse(w, logs)
	}

	// Time range query endpoint
	handlers["/api/logs/range"] = func(w http.ResponseWriter, r *http.Request) {
		logger.Info("API request: time range query", "remote_addr", r.RemoteAddr)

		startStr := r.URL.Query().Get("start")
		endStr := r.URL.Query().Get("end")

		if startStr == "" || endStr == "" {
			sendErrorResponse(w, "start and end parameters required")
			return
		}

		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			sendErrorResponse(w, "Invalid start time format (use RFC3339)")
			return
		}

		end, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			sendErrorResponse(w, "Invalid end time format (use RFC3339)")
			return
		}

		logs, err := db.QueryByTimeRange(start, end)
		if err != nil {
			logger.Error("Failed to query logs by range", "error", err, "start", start, "end", end)
			sendErrorResponse(w, "Failed to fetch logs")
			return
		}

		sendSuccessResponse(w, logs)
	}

	// Summary statistics endpoint
	handlers["/api/stats/summary"] = func(w http.ResponseWriter, r *http.Request) {
		logger.Info("API request: summary stats", "remote_addr", r.RemoteAddr)

		logs, err := db.GetAll()
		if err != nil {
			logger.Error("Failed to get all logs for stats", "error", err)
			sendErrorResponse(w, "Failed to fetch statistics")
			return
		}

		stats := calculateStats(logs)
		sendSuccessResponse(w, stats)
	}

	// Time series data for charts (hourly aggregation)
	handlers["/api/charts/timeseries"] = func(w http.ResponseWriter, r *http.Request) {
		logger.Info("API request: time series data", "remote_addr", r.RemoteAddr)

		hoursStr := r.URL.Query().Get("hours")
		hours := 24 // default to 24 hours
		if hoursStr != "" {
			if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 {
				hours = h
			}
		}

		end := time.Now()
		start := end.Add(-time.Duration(hours) * time.Hour)

		logs, err := db.QueryByTimeRange(start, end)
		if err != nil {
			logger.Error("Failed to query logs for time series", "error", err)
			sendErrorResponse(w, "Failed to fetch time series data")
			return
		}

		timeSeries := aggregateByHour(logs)
		sendSuccessResponse(w, timeSeries)
	}

	// Size breakdown for distribution charts
	handlers["/api/charts/breakdown"] = func(w http.ResponseWriter, r *http.Request) {
		logger.Info("API request: size breakdown", "remote_addr", r.RemoteAddr)

		logs, err := db.GetAll()
		if err != nil {
			logger.Error("Failed to get logs for breakdown", "error", err)
			sendErrorResponse(w, "Failed to fetch breakdown data")
			return
		}

		breakdown := calculateSizeBreakdown(logs)
		sendSuccessResponse(w, breakdown)
	}

	return handlers
}

// sendSuccessResponse sends a successful API response with the provided data.
// It sets appropriate headers including CORS headers for development and
// formats the response using the standard APIResponse structure.
//
// Parameters:
//   - w: HTTP response writer
//   - data: Data to include in the response
func sendSuccessResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Enable CORS for local development
	response := APIResponse{Success: true, Data: data}
	json.NewEncoder(w).Encode(response)
}

// sendErrorResponse sends an error API response with the provided message.
// It sets appropriate headers and HTTP status codes for error conditions.
//
// Parameters:
//   - w: HTTP response writer
//   - message: Error message to include in the response
func sendErrorResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusInternalServerError)
	response := APIResponse{Success: false, Error: message}
	json.NewEncoder(w).Encode(response)
}

// calculateStats computes summary statistics from a slice of log size records.
// This function analyzes the provided data to generate comprehensive metrics
// including totals, averages, min/max values, and timestamps.
//
// Parameters:
//   - logs: Slice of log size records to analyze
//
// Returns:
//   - LogSizeStats: Calculated statistics structure
//
// The function handles edge cases such as empty datasets and automatically
// determines the most recent record timestamp.
func calculateStats(logs []database.LogSize) LogSizeStats {
	if len(logs) == 0 {
		return LogSizeStats{}
	}

	var total int64
	min := logs[0].Filesize
	max := logs[0].Filesize
	var lastUpdated time.Time

	for _, log := range logs {
		total += log.Filesize
		if log.Filesize < min {
			min = log.Filesize
		}
		if log.Filesize > max {
			max = log.Filesize
		}
		if log.Timestamp.After(lastUpdated) {
			lastUpdated = log.Timestamp
		}
	}

	avg := float64(total) / float64(len(logs))

	return LogSizeStats{
		TotalRecords: int64(len(logs)),
		TotalSize:    total,
		AverageSize:  avg,
		MinSize:      min,
		MaxSize:      max,
		LastUpdated:  lastUpdated.Format(time.RFC3339),
	}
}

func aggregateByHour(logs []database.LogSize) []TimeSeriesPoint {
	hourMap := make(map[string]struct {
		Count     int
		TotalSize int64
	})

	for _, log := range logs {
		hourKey := log.Timestamp.Truncate(time.Hour).Format("2006-01-02T15:04:05Z07:00")
		data := hourMap[hourKey]
		data.Count++
		data.TotalSize += log.Filesize
		hourMap[hourKey] = data
	}

	var result []TimeSeriesPoint
	for timestamp, data := range hourMap {
		result = append(result, TimeSeriesPoint{
			Timestamp: timestamp,
			Count:     data.Count,
			TotalSize: data.TotalSize,
		})
	}

	return result
}

func calculateSizeBreakdown(logs []database.LogSize) []SizeBreakdown {
	ranges := []struct {
		Name string
		Min  int64
		Max  int64
	}{
		{"< 1KB", 0, 1024},
		{"1KB - 10KB", 1024, 10 * 1024},
		{"10KB - 100KB", 10 * 1024, 100 * 1024},
		{"100KB - 1MB", 100 * 1024, 1024 * 1024},
		{"1MB - 10MB", 1024 * 1024, 10 * 1024 * 1024},
		{"> 10MB", 10 * 1024 * 1024, int64(^uint64(0) >> 1)}, // max int64
	}

	rangeCounts := make([]int, len(ranges))
	total := len(logs)

	for _, log := range logs {
		for i, r := range ranges {
			if log.Filesize >= r.Min && log.Filesize < r.Max {
				rangeCounts[i]++
				break
			}
		}
	}

	var result []SizeBreakdown
	for i, r := range ranges {
		percentage := 0.0
		if total > 0 {
			percentage = float64(rangeCounts[i]) / float64(total) * 100
		}
		result = append(result, SizeBreakdown{
			Range:      r.Name,
			Count:      rangeCounts[i],
			Percentage: percentage,
		})
	}

	return result
}

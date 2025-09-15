# LogpushEstimator Documentation

Welcome to the comprehensive documentation for LogpushEstimator, a Cloudflare log ingestion monitoring tool designed to help analyze and estimate data usage from Cloudflare's Logpush service.

## 📚 Documentation Structure

### Core Documentation
- **[Architecture Overview](design/architecture.md)** - System architecture and component overview
- **[System Requirements](design/requirements.md)** - Functional and non-functional requirements
- **[Design Specifications](design/design-specs.md)** - Detailed design specifications and patterns
- **[Database Schema](design/database-schema.md)** - Database design and data models

### API Documentation
- **[API Reference](api/api-reference.md)** - Complete API endpoint documentation with examples and SDKs

### Development
- **[Development Guide](development/development-guide.md)** - Local development environment setup, coding standards, and contribution guidelines

### Deployment
- **[Deployment Guide](deployment/deployment-guide.md)** - Production deployment instructions, configuration, and monitoring

## 🚀 Quick Start

1. **Installation**: See [Development Guide](development/development-guide.md)
2. **Configuration**: Review [Deployment Guide](deployment/deployment-guide.md)
3. **API Usage**: Check [API Reference](api/api-reference.md)
4. **Deployment**: Follow [Deployment Guide](deployment/deployment.md)

## 🏗️ Architecture Overview

LogpushEstimator is built with a dual-server architecture:

```
┌─────────────────────────────────────────────────────────────┐
│                    LogpushEstimator                         │
├─────────────────────────────────────────────────────────────┤
│  Ingestion Server (8080)    │    GUI Server (8081)          │
│  ┌─────────────────────┐    │    ┌─────────────────────┐     │
│  │ POST /ingest        │    │    │ GET /dashboard      │     │
│  │ GET /health         │    │    │ GET /api/*          │     │
│  └─────────────────────┘    │    │ GET /static/*       │     │
│                             │    └─────────────────────┘     │
├─────────────────────────────────────────────────────────────┤
│                    SQLite Database                          │
│                   (logpush.db)                              │
└─────────────────────────────────────────────────────────────┘
```

## 📊 Key Features

- **Real-time Log Ingestion**: HTTP endpoint for receiving Cloudflare log data
- **Web Dashboard**: Interactive interface for visualizing log analytics
- **REST API**: Comprehensive API for accessing stored data and statistics
- **SQLite Storage**: Lightweight, serverless database for data persistence
- **Time-Series Analytics**: Track log volume trends over time
- **Size Distribution Analysis**: Understand log size patterns and distributions

## 🔧 Technology Stack

- **Language**: Go 1.21+
- **Database**: SQLite3 with CGO
- **Web Framework**: Go standard library (net/http)
- **Logging**: Structured logging with slog
- **Testing**: Go built-in testing package with httptest
- **Frontend**: HTML, CSS, JavaScript (vanilla)

## 📈 Use Cases

1. **Cloudflare Log Analysis**: Monitor and analyze Cloudflare Logpush data volumes
2. **Capacity Planning**: Estimate storage requirements for log data
3. **Performance Monitoring**: Track log ingestion rates and patterns
4. **Cost Estimation**: Calculate data transfer and storage costs
5. **Operational Insights**: Understand log generation patterns across time

## 🔗 Related Resources

- [Cloudflare Logpush Documentation](https://developers.cloudflare.com/logs/get-started/enable-destinations/)
- [Go Programming Language](https://golang.org/)
- [SQLite Documentation](https://sqlite.org/docs.html)

## 📝 Documentation Conventions

- **Code blocks**: Use syntax highlighting for Go, JSON, bash, etc.
- **API examples**: Include both request and response examples
- **Configuration**: Show both default and example values
- **Diagrams**: Use ASCII art or mermaid syntax for visual representations
- **Links**: Use relative links for internal documentation

## 💡 Getting Help

- Check the [Deployment Guide](deployment/deployment-guide.md#troubleshooting) for common issues
- Review [API Reference](api/api-reference.md#error-codes) for API-specific problems  
- See [Development Guide](development/development-guide.md#troubleshooting) for development environment issues

---

*Last updated: September 15, 2025*
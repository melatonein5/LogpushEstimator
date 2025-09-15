# Deployment Guide

## Table of Contents
1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Build Configuration](#build-configuration)
4. [Local Development](#local-development)
5. [Production Deployment](#production-deployment)
6. [Docker Deployment](#docker-deployment)
7. [Cloud Deployment](#cloud-deployment)
8. [Security Configuration](#security-configuration)
9. [Monitoring and Logging](#monitoring-and-logging)
10. [Troubleshooting](#troubleshooting)

## Overview

LogpushEstimator is a Go application that consists of two HTTP servers:
- **Ingestion Server** (Port 8080): Handles log data ingestion
- **GUI Server** (Port 8081): Provides web dashboard and API endpoints

The application uses SQLite for data persistence and can be deployed in various environments from local development to production cloud infrastructure.

## Prerequisites

### System Requirements

**Minimum Requirements**:
- CPU: 1 core
- RAM: 512 MB
- Storage: 1 GB free space
- Operating System: Linux, macOS, or Windows

**Recommended for Production**:
- CPU: 2+ cores
- RAM: 2+ GB
- Storage: 10+ GB free space
- Operating System: Linux (Ubuntu 20.04+ or CentOS 8+)

### Software Dependencies

**Required**:
- Go 1.21 or later
- SQLite3 (included with Go sqlite driver)

**Optional**:
- Docker (for containerized deployment)
- nginx (for reverse proxy in production)
- systemd (for service management on Linux)

### Network Requirements

**Firewall Configuration**:
```bash
# Allow ingestion traffic
sudo ufw allow 8080/tcp

# Allow GUI/API traffic  
sudo ufw allow 8081/tcp

# Optional: Allow SSH for management
sudo ufw allow 22/tcp
```

**Port Requirements**:
- Port 8080: Ingestion server (must be accessible by log sources)
- Port 8081: GUI/API server (must be accessible by users/dashboards)

## Build Configuration

### Environment Variables

The application supports the following environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PATH` | `./logs.db` | SQLite database file path |
| `INGESTION_PORT` | `8080` | Port for ingestion server |
| `GUI_PORT` | `8081` | Port for GUI server |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `STATIC_DIR` | `./src/gui/static` | Static files directory |
| `TEMPLATES_DIR` | `./src/gui/templates` | Templates directory |

### Build Configuration File

Create a `.env` file for configuration:

```bash
# .env file
DB_PATH=/var/lib/logpush-estimator/logs.db
INGESTION_PORT=8080
GUI_PORT=8081
LOG_LEVEL=info
STATIC_DIR=/opt/logpush-estimator/static
TEMPLATES_DIR=/opt/logpush-estimator/templates
```

### Build Scripts

Create a `scripts/build.sh` file:

```bash
#!/bin/bash
set -e

echo "Building LogpushEstimator..."

# Clean previous builds
rm -rf build/

# Create build directory
mkdir -p build/{linux,windows,macos}

# Build for Linux (production)
echo "Building for Linux..."
GOOS=linux GOARCH=amd64 go build -o build/linux/logpush-estimator .

# Build for Windows
echo "Building for Windows..."
GOOS=windows GOARCH=amd64 go build -o build/windows/logpush-estimator.exe .

# Build for macOS
echo "Building for macOS..."
GOOS=darwin GOARCH=amd64 go build -o build/macos/logpush-estimator .

# Copy static files
echo "Copying static files..."
cp -r src/gui/static build/linux/
cp -r src/gui/templates build/linux/
cp -r src/gui/static build/windows/
cp -r src/gui/templates build/windows/
cp -r src/gui/static build/macos/
cp -r src/gui/templates build/macos/

echo "Build complete!"
```

Make the script executable:
```bash
chmod +x scripts/build.sh
./scripts/build.sh
```

## Local Development

### Quick Start

1. **Clone and Setup**:
   ```bash
   git clone <repository>
   cd LogpushEstimator
   go mod download
   ```

2. **Run Application**:
   ```bash
   go run main.go
   ```

3. **Verify Services**:
   ```bash
   # Check ingestion server
   curl http://localhost:8080/health
   
   # Check GUI server
   curl http://localhost:8081/api/stats/summary
   
   # Open dashboard
   open http://localhost:8081
   ```

### Development Configuration

Create a development configuration file `config/development.env`:

```bash
DB_PATH=./data/dev-logs.db
LOG_LEVEL=debug
INGESTION_PORT=8080
GUI_PORT=8081
```

Load configuration:
```bash
export $(cat config/development.env | xargs)
go run main.go
```

### Testing in Development

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific test package
go test ./src/database

# Run with coverage
go test -cover ./...
```

## Production Deployment

### System Setup

1. **Create Application User**:
   ```bash
   sudo useradd -r -s /bin/false logpush-estimator
   sudo mkdir -p /opt/logpush-estimator
   sudo mkdir -p /var/lib/logpush-estimator
   sudo mkdir -p /var/log/logpush-estimator
   ```

2. **Set Permissions**:
   ```bash
   sudo chown -R logpush-estimator:logpush-estimator /opt/logpush-estimator
   sudo chown -R logpush-estimator:logpush-estimator /var/lib/logpush-estimator
   sudo chown -R logpush-estimator:logpush-estimator /var/log/logpush-estimator
   ```

### Application Deployment

1. **Copy Application Files**:
   ```bash
   sudo cp build/linux/logpush-estimator /opt/logpush-estimator/
   sudo cp -r build/linux/static /opt/logpush-estimator/
   sudo cp -r build/linux/templates /opt/logpush-estimator/
   sudo chmod +x /opt/logpush-estimator/logpush-estimator
   ```

2. **Create Configuration**:
   ```bash
   sudo tee /opt/logpush-estimator/config.env << EOF
   DB_PATH=/var/lib/logpush-estimator/logs.db
   INGESTION_PORT=8080
   GUI_PORT=8081
   LOG_LEVEL=info
   STATIC_DIR=/opt/logpush-estimator/static
   TEMPLATES_DIR=/opt/logpush-estimator/templates
   EOF
   ```

### Systemd Service

Create `/etc/systemd/system/logpush-estimator.service`:

```ini
[Unit]
Description=LogpushEstimator Service
After=network.target
Wants=network.target

[Service]
Type=simple
User=logpush-estimator
Group=logpush-estimator
WorkingDirectory=/opt/logpush-estimator
EnvironmentFile=/opt/logpush-estimator/config.env
ExecStart=/opt/logpush-estimator/logpush-estimator
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=logpush-estimator

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/logpush-estimator

[Install]
WantedBy=multi-user.target
```

Enable and start the service:
```bash
sudo systemctl daemon-reload
sudo systemctl enable logpush-estimator
sudo systemctl start logpush-estimator
sudo systemctl status logpush-estimator
```

### Reverse Proxy Configuration

Create nginx configuration `/etc/nginx/sites-available/logpush-estimator`:

```nginx
upstream logpush_ingestion {
    server localhost:8080;
}

upstream logpush_gui {
    server localhost:8081;
}

# Ingestion endpoint
server {
    listen 80;
    server_name ingestion.yourdomain.com;

    location / {
        proxy_pass http://logpush_ingestion;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Increase body size for large logs
        client_max_body_size 100M;
    }
}

# GUI/API endpoint
server {
    listen 80;
    server_name dashboard.yourdomain.com;

    location / {
        proxy_pass http://logpush_gui;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Enable the configuration:
```bash
sudo ln -s /etc/nginx/sites-available/logpush-estimator /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## Docker Deployment

### Dockerfile

Create a `Dockerfile`:

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o logpush-estimator .

# Production stage
FROM alpine:latest

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN adduser -D -s /bin/sh logpush

WORKDIR /app

# Copy binary and static files
COPY --from=builder /app/logpush-estimator .
COPY --from=builder /app/src/gui/static ./static
COPY --from=builder /app/src/gui/templates ./templates

# Create data directory
RUN mkdir -p /app/data && chown logpush:logpush /app/data

USER logpush

# Environment variables
ENV DB_PATH=/app/data/logs.db
ENV STATIC_DIR=/app/static
ENV TEMPLATES_DIR=/app/templates
ENV INGESTION_PORT=8080
ENV GUI_PORT=8081

# Expose ports
EXPOSE 8080 8081

CMD ["./logpush-estimator"]
```

### Docker Compose

Create a `docker-compose.yml`:

```yaml
version: '3.8'

services:
  logpush-estimator:
    build: .
    ports:
      - "8080:8080"  # Ingestion port
      - "8081:8081"  # GUI port
    volumes:
      - logpush_data:/app/data
    environment:
      - DB_PATH=/app/data/logs.db
      - LOG_LEVEL=info
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/ssl/certs:ro
    depends_on:
      - logpush-estimator
    restart: unless-stopped

volumes:
  logpush_data:
```

### Build and Deploy

```bash
# Build and start
docker-compose up -d

# View logs
docker-compose logs -f logpush-estimator

# Check status
docker-compose ps

# Update application
docker-compose pull
docker-compose up -d
```

## Cloud Deployment

### AWS Deployment

#### EC2 Deployment

1. **Launch EC2 Instance**:
   ```bash
   # t3.small or larger recommended
   # Amazon Linux 2 or Ubuntu 20.04
   # Security group: ports 22, 80, 443, 8080, 8081
   ```

2. **Install Dependencies**:
   ```bash
   # Amazon Linux 2
   sudo yum update -y
   sudo yum install -y git
   
   # Install Go
   wget https://golang.org/dl/go1.21.0.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
   echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
   source ~/.bashrc
   ```

3. **Deploy Application**:
   ```bash
   git clone <repository>
   cd LogpushEstimator
   go build -o logpush-estimator .
   
   # Follow production deployment steps above
   ```

#### ECS Deployment

Create `ecs-task-definition.json`:

```json
{
  "family": "logpush-estimator",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "executionRoleArn": "arn:aws:iam::ACCOUNT:role/ecsTaskExecutionRole",
  "containerDefinitions": [
    {
      "name": "logpush-estimator",
      "image": "your-registry/logpush-estimator:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        },
        {
          "containerPort": 8081,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "DB_PATH",
          "value": "/app/data/logs.db"
        },
        {
          "name": "LOG_LEVEL",
          "value": "info"
        }
      ],
      "mountPoints": [
        {
          "sourceVolume": "data",
          "containerPath": "/app/data"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/logpush-estimator",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ],
  "volumes": [
    {
      "name": "data",
      "efsVolumeConfiguration": {
        "fileSystemId": "fs-XXXXXXXX"
      }
    }
  ]
}
```

### Google Cloud Platform

#### Cloud Run Deployment

1. **Build and Push Container**:
   ```bash
   gcloud builds submit --tag gcr.io/PROJECT-ID/logpush-estimator
   ```

2. **Deploy to Cloud Run**:
   ```bash
   gcloud run deploy logpush-estimator \
     --image gcr.io/PROJECT-ID/logpush-estimator \
     --platform managed \
     --port 8080 \
     --allow-unauthenticated \
     --memory 1Gi \
     --cpu 1
   ```

#### Compute Engine

Similar to AWS EC2 deployment with GCP-specific commands:

```bash
# Create instance
gcloud compute instances create logpush-estimator \
  --zone=us-central1-a \
  --machine-type=e2-medium \
  --image-family=ubuntu-2004-lts \
  --image-project=ubuntu-os-cloud \
  --boot-disk-size=20GB
```

### Azure Deployment

#### Container Instances

```bash
az container create \
  --resource-group myResourceGroup \
  --name logpush-estimator \
  --image your-registry/logpush-estimator:latest \
  --ports 8080 8081 \
  --cpu 1 \
  --memory 2 \
  --environment-variables \
    DB_PATH=/app/data/logs.db \
    LOG_LEVEL=info
```

## Security Configuration

### TLS/SSL Configuration

1. **Generate SSL Certificate**:
   ```bash
   # Using Let's Encrypt
   sudo certbot --nginx -d yourdomain.com
   ```

2. **Update nginx Configuration**:
   ```nginx
   server {
       listen 443 ssl http2;
       server_name yourdomain.com;
       
       ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
       ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;
       
       # Security headers
       add_header X-Frame-Options DENY;
       add_header X-Content-Type-Options nosniff;
       add_header X-XSS-Protection "1; mode=block";
       
       location / {
           proxy_pass http://localhost:8081;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
       }
   }
   ```

### Firewall Configuration

```bash
# UFW (Ubuntu/Debian)
sudo ufw enable
sudo ufw allow ssh
sudo ufw allow 'Nginx Full'
sudo ufw deny 8080  # Block direct access to ingestion
sudo ufw deny 8081  # Block direct access to GUI

# Only allow through nginx
sudo ufw allow from 127.0.0.1 to any port 8080
sudo ufw allow from 127.0.0.1 to any port 8081
```

### Application Security

1. **Rate Limiting** (nginx):
   ```nginx
   http {
       limit_req_zone $binary_remote_addr zone=ingestion:10m rate=10r/s;
       
       server {
           location /ingest {
               limit_req zone=ingestion burst=20 nodelay;
               proxy_pass http://localhost:8080;
           }
       }
   }
   ```

2. **Request Size Limits**:
   ```nginx
   client_max_body_size 10M;  # Limit log size
   ```

3. **IP Whitelisting**:
   ```nginx
   location /ingest {
       allow 192.168.1.0/24;
       allow 10.0.0.0/8;
       deny all;
       proxy_pass http://localhost:8080;
   }
   ```

## Monitoring and Logging

### Application Logging

The application logs to stdout/stderr by default. Configure log aggregation:

1. **systemd Journal**:
   ```bash
   journalctl -u logpush-estimator -f
   journalctl -u logpush-estimator --since "1 hour ago"
   ```

2. **File Logging** (modify systemd service):
   ```ini
   [Service]
   StandardOutput=append:/var/log/logpush-estimator/app.log
   StandardError=append:/var/log/logpush-estimator/error.log
   ```

### Health Monitoring

1. **Basic Health Check Script**:
   ```bash
   #!/bin/bash
   # /opt/logpush-estimator/health-check.sh
   
   INGESTION_URL="http://localhost:8080/health"
   GUI_URL="http://localhost:8081/api/stats/summary"
   
   # Check ingestion service
   if ! curl -f -s $INGESTION_URL > /dev/null; then
       echo "Ingestion service is down"
       exit 1
   fi
   
   # Check GUI service  
   if ! curl -f -s $GUI_URL > /dev/null; then
       echo "GUI service is down"
       exit 1
   fi
   
   echo "All services healthy"
   exit 0
   ```

2. **Cron Health Check**:
   ```bash
   # Add to crontab
   */5 * * * * /opt/logpush-estimator/health-check.sh || echo "LogpushEstimator health check failed" | mail -s "Service Alert" admin@yourdomain.com
   ```

### Prometheus Monitoring

1. **Add Metrics Endpoint** (future enhancement):
   ```go
   // Add to main.go
   import "github.com/prometheus/client_golang/prometheus/promhttp"
   
   http.Handle("/metrics", promhttp.Handler())
   ```

2. **Prometheus Configuration**:
   ```yaml
   scrape_configs:
     - job_name: 'logpush-estimator'
       static_configs:
         - targets: ['localhost:8082']
   ```

### Log Rotation

```bash
# /etc/logrotate.d/logpush-estimator
/var/log/logpush-estimator/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 644 logpush-estimator logpush-estimator
    postrotate
        systemctl reload logpush-estimator
    endscript
}
```

## Troubleshooting

### Common Issues

#### Service Won't Start

1. **Check Service Status**:
   ```bash
   sudo systemctl status logpush-estimator
   journalctl -u logpush-estimator --no-pager
   ```

2. **Common Causes**:
   - Missing binary: Ensure `/opt/logpush-estimator/logpush-estimator` exists
   - Permission issues: Check file ownership and permissions
   - Port conflicts: Ensure ports 8080/8081 are available
   - Database issues: Check database file permissions

#### Database Issues

1. **Permission Errors**:
   ```bash
   sudo chown logpush-estimator:logpush-estimator /var/lib/logpush-estimator/logs.db
   sudo chmod 644 /var/lib/logpush-estimator/logs.db
   ```

2. **Database Corruption**:
   ```bash
   sqlite3 /var/lib/logpush-estimator/logs.db "PRAGMA integrity_check;"
   ```

#### Memory Issues

1. **Monitor Memory Usage**:
   ```bash
   ps aux | grep logpush-estimator
   sudo systemctl status logpush-estimator
   ```

2. **Increase Memory Limits** (systemd):
   ```ini
   [Service]
   MemoryMax=2G
   ```

#### Network Issues

1. **Check Port Availability**:
   ```bash
   sudo netstat -tlnp | grep :8080
   sudo netstat -tlnp | grep :8081
   ```

2. **Test Connectivity**:
   ```bash
   curl -v http://localhost:8080/health
   curl -v http://localhost:8081/api/stats/summary
   ```

#### Performance Issues

1. **Database Optimization**:
   ```bash
   sqlite3 /var/lib/logpush-estimator/logs.db "VACUUM;"
   sqlite3 /var/lib/logpush-estimator/logs.db "ANALYZE;"
   ```

2. **Monitor System Resources**:
   ```bash
   top -p $(pgrep logpush-estimator)
   iotop -p $(pgrep logpush-estimator)
   ```

### Debug Mode

Enable debug logging:

```bash
# Update config.env
LOG_LEVEL=debug

# Restart service
sudo systemctl restart logpush-estimator

# Monitor logs
journalctl -u logpush-estimator -f
```

### Support and Recovery

#### Backup and Recovery

1. **Database Backup**:
   ```bash
   # Daily backup script
   #!/bin/bash
   DATE=$(date +%Y%m%d_%H%M%S)
   cp /var/lib/logpush-estimator/logs.db \
      /var/backups/logpush-estimator-$DATE.db
   ```

2. **Configuration Backup**:
   ```bash
   tar -czf /var/backups/logpush-config-$(date +%Y%m%d).tar.gz \
       /opt/logpush-estimator/config.env \
       /etc/systemd/system/logpush-estimator.service
   ```

#### Emergency Recovery

1. **Service Recovery**:
   ```bash
   # Stop service
   sudo systemctl stop logpush-estimator
   
   # Restore from backup
   sudo cp /var/backups/logpush-estimator-YYYYMMDD.db \
           /var/lib/logpush-estimator/logs.db
   
   # Fix permissions
   sudo chown logpush-estimator:logpush-estimator /var/lib/logpush-estimator/logs.db
   
   # Start service
   sudo systemctl start logpush-estimator
   ```

2. **Complete Reinstall**:
   ```bash
   # Remove existing installation
   sudo systemctl stop logpush-estimator
   sudo systemctl disable logpush-estimator
   sudo rm -rf /opt/logpush-estimator
   
   # Reinstall following deployment guide
   ```

For additional support, check the application logs and refer to the [troubleshooting section](../README.md#troubleshooting) in the main documentation.

---

*This deployment guide is current as of September 15, 2025. For the most up-to-date deployment information, always refer to the official documentation and release notes.*
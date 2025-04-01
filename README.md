# Ronnin API

A Go-based API service that creates Jira tickets from HTTP requests. Built with Gin framework and includes Swagger documentation, Prometheus metrics, and structured logging.

## Features

- RESTful API endpoints using Gin framework
- Automatic Swagger documentation
- Prometheus metrics
- Structured logging with Zap
- Graceful shutdown
- CORS support
- Environment-based configuration
- Jira integration
- Health check endpoint
- Request validation

## Prerequisites

- Go 1.19 or later
- Jira account and API token
- Git

## Installation

1. Clone the repository:
```bash
git clone https://github.com/parvez-capri/ronnin.git
cd ronnin
```

2. Install dependencies:
```bash
go mod download
```

3. Install required tools:
```bash
# Install swag for Swagger documentation
go install github.com/swaggo/swag/cmd/swag@latest

# Install Air for hot reload (optional)
go install github.com/air-verse/air@latest
```

## Configuration

1. Create a `.env` file in the project root:
```env
# Server Configuration
PORT=8080
ENV=development
LOG_LEVEL=info

# CORS Configuration
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

# Jira Configuration
JIRA_URL=https://your-jira-instance.atlassian.net
JIRA_USERNAME=your-jira-email@example.com
JIRA_API_TOKEN=your-jira-api-token
JIRA_PROJECT_KEY=YOUR_PROJECT
SUPPORT_TEAM_MEMBERS=member1,member2
```

## Running the Application

### Development Mode

Using Air (with hot reload):
```bash
air
```

Standard way:
```bash
go run cmd/api/main.go
```

### Production Mode
```bash
ENV=production go run cmd/api/main.go
```

## API Documentation

After starting the server, visit:
- Swagger UI: http://localhost:8080/swagger/index.html
- Swagger JSON: http://localhost:8080/swagger/doc.json

## API Endpoints

### Health Check
```bash
curl http://localhost:8080/health
```

### Create Ticket
```bash
curl -X POST \
  http://localhost:8080/create-ticket \
  -H 'Content-Type: application/json' \
  -d '{
    "url": "https://example.com/api/endpoint",
    "payload": {
      "key": "example payload"
    },
    "response": {
      "status": "500",
      "body": "Internal Server Error"
    },
    "requestHeaders": {
      "Authorization": "Bearer xxx",
      "Content-Type": "application/json"
    },
    "imageS3URL": "https://your-bucket.s3.amazonaws.com/screenshot.png"
  }'
```

### Metrics
```bash
curl http://localhost:8080/metrics
```

## Project Structure
- `cmd/`: Application entry points
- `internal/`: Private application code
- `pkg/`: Shared utilities
- `config/`: Configuration management

## Dependencies
- Chi Router
- Viper (Configuration)
- Zap (Logging)

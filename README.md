# Ronnin API

A Go-based API service for reporting issues that creates Jira tickets with screenshot uploads. Features MongoDB persistence, AWS S3 integration for image uploads with presigned URLs, and smart formatting for Jira tickets. Built with Gin framework and includes Swagger documentation, Prometheus metrics, and structured logging.

## Features

- RESTful API endpoint for reporting issues with file uploads
- MongoDB persistence for ticket data
- AWS S3 integration for file uploads with presigned URLs
- Jira ticket creation with smart formatting
- Automatic Swagger documentation
- Prometheus metrics
- Structured logging with Zap
- Graceful shutdown
- CORS support
- Environment-based configuration
- Health check endpoint
- Request validation
- Smart truncation for Jira ticket descriptions with fallback to comments
- Docker support for containerized deployment

## Prerequisites

- Go 1.19 or later
- Jira account and API token
- MongoDB (local or remote)
- AWS S3 bucket (for file uploads)
- Git
- Docker and Docker Compose (for containerized deployment)

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
DEFAULT_PRIORITY=Medium

# AWS S3 Configuration
AWS_S3_ACCESS_KEY=your-access-key
AWS_S3_SECRET_KEY=your-secret-key
AWS_S3_REGION=us-east-1
AWS_S3_BUCKET_NAME=your-bucket-name
AWS_S3_BASE_URL=https://your-bucket.s3.amazonaws.com

# MongoDB Configuration
MONGO_URI=mongodb://localhost:27017
MONGO_DB=ronnin
MONGO_COLLECTION=tickets
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

### Docker Deployment

The application can be deployed using Docker and Docker Compose:

1. Build and start all services:
```bash
docker-compose up -d
```

This will start:
- The Ronnin API on port 8080
- MongoDB on port 27017
- MinIO (S3-compatible storage) on ports 9000 (API) and 9001 (Console)

2. View logs:
```bash
docker-compose logs -f api
```

3. Stop all services:
```bash
docker-compose down
```

4. To rebuild the API after making changes:
```bash
docker-compose build api
docker-compose up -d api
```

#### Accessing MinIO Console
MinIO provides an S3-compatible interface for development:
- Console URL: http://localhost:9001
- Username: minio
- Password: minio123

## API Documentation

After starting the server, visit:
- Swagger UI: http://localhost:8080/swagger/index.html
- Swagger JSON: http://localhost:8080/swagger/doc.json

## API Endpoints

### Health Check
```bash
curl http://localhost:8080/health
```

### Report Issue with File Upload
```bash
curl -X POST \
  http://localhost:8080/report-issue \
  -H 'Content-Type: multipart/form-data' \
  -F 'issue=Login Error' \
  -F 'description=Cannot log in with valid credentials' \
  -F 'userEmail=user@example.com' \
  -F 'leadId=12345' \
  -F 'product=Website' \
  -F 'pageUrl=https://example.com/login' \
  -F 'failedNetworkCalls=[{"url":"https://api.example.com/login","method":"POST","status":401}]' \
  -F 'image0=@/path/to/screenshot.png'
```

### Retrieve All Tickets
```bash
curl http://localhost:8080/tickets
```

### Retrieve Specific Ticket
```bash
curl http://localhost:8080/tickets/PROJ-123
```

### Metrics
```bash
curl http://localhost:8080/metrics
```

## Project Structure
- `cmd/`: Application entry points
  - `api/`: API server
- `internal/`: Private application code
  - `config/`: Configuration management
  - `handlers/`: HTTP handlers
  - `models/`: Data models
  - `services/`: Business logic
    - `jira.go`: Jira ticket creation service
    - `s3.go`: AWS S3 file upload service
    - `mongo.go`: MongoDB persistence service
  - `errors/`: Error handling utilities
- `pkg/`: Shared utilities
  - `logger/`: Logging setup
- `docs/`: Swagger documentation

## Database Schema

### MongoDB Collection: tickets

The application stores ticket data in MongoDB with a flattened structure:

| Field                  | Type         | Description                             |
|------------------------|--------------|-----------------------------------------|
| _id                    | ObjectId     | MongoDB document ID                      |
| ticket_id              | string       | Jira ticket ID (e.g., PROJ-123)         |
| status                 | string       | Ticket status                           |
| assigned_to            | string       | User the ticket is assigned to          |
| jira_link              | string       | Link to the ticket in Jira              |
| created_at             | datetime     | Ticket creation timestamp               |
| issue                  | string       | Issue title                             |
| description            | string       | Issue description                       |
| user_email             | string       | Reporter's email                        |
| lead_id                | string       | Lead/customer ID                        |
| product                | string       | Product name                            |
| page_url               | string       | URL where the issue occurred            |
| image_url              | string       | S3 presigned URL for screenshot (valid for 7 days) |
| failed_network_calls_json | string    | JSON string of network call data        |
| payload_json           | string       | JSON string of request payload          |
| response_json          | string       | JSON string of response data            |
| request_headers_json   | string       | JSON string of request headers          |

## Features Details

### S3 Image Upload
- Upload screenshots to S3 when reporting issues
- Generates 7-day presigned URLs for secure access
- URLs are embedded in Jira tickets and stored in MongoDB

### MongoDB Persistence
- Stores all ticket data in a flattened structure
- Supports querying by Jira ticket ID
- Complex data structures are serialized to JSON strings

### Jira Integration
- Creates well-formatted tickets with collapsible sections
- Handles large data with smart truncation
- Falls back to comments when description exceeds Jira's limit
- Randomly assigns tickets to team members
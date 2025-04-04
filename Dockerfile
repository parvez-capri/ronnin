# Build stage
FROM golang:1.19-alpine AS builder

# Install git for fetching Go dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Generate Swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@latest
RUN swag init -g cmd/api/main.go

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/ronnin ./cmd/api

# Final stage
FROM alpine:3.17

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/ronnin .
# Copy .env file - for environments where you want to use the container's .env
# Comment this out if you're mounting an external .env file
COPY --from=builder /app/.env .

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./ronnin"] 
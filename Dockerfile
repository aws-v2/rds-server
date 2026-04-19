# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o rds-server ./cmd/api

# Final stage
FROM alpine:latest

WORKDIR /app

# Install certificates
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/rds-server .
COPY --from=builder /app/docs ./docs

# Copy migrations (critical for database synchronization)
COPY --from=builder /app/internal/infrastructure/database/migrations ./internal/infrastructure/database/migrations

# Expose the port
EXPOSE 8087

# Run the binary
CMD ["./rds-server"]

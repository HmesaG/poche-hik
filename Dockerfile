# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
# We use CGO_ENABLED=0 for a static binary (modernc.org/sqlite is pure Go)
RUN CGO_ENABLED=0 GOOS=linux go build -o ponches-server ./cmd/server/main.go

# Stage 2: Final image
FROM alpine:latest

WORKDIR /app

# Install ca-certificates and tzdata
RUN apk add --no-cache ca-certificates tzdata

# Copy the binary from the builder stage
COPY --from=builder /app/ponches-server .

# Copy the web assets
COPY --from=builder /app/web ./web

# Create a data directory for the SQLite database
RUN mkdir -p /app/data

# Expose the port (SERVER_PORT env var should match this or be mapped)
EXPOSE 8080

# Run the server
CMD ["./ponches-server"]

FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git to fetch dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod file for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o main ./cmd/server

# Final stage - Alpine image
FROM alpine:latest

# Install CA certificates (for HTTPS requests)
RUN apk --no-cache add ca-certificates

# Create a non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -S appuser -u 1001 -G appgroup

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy .env file if it exists (optional)
COPY --from=builder /app/cmd/server/.env ./ 

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port (adjust as needed)
EXPOSE 8080

# Run the binary
CMD ["./main"]
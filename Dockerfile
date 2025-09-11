# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build all binaries
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o producer ./cmd/producer
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o consumer ./cmd/consumer
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api ./cmd/api

# Final stage
FROM scratch

# Copy ca-certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binaries
COPY --from=builder /app/producer /producer
COPY --from=builder /app/consumer /consumer
COPY --from=builder /app/api /api

# Expose ports (will be overridden by docker-compose)
EXPOSE 8080 8081 8082

# Default command (will be overridden by docker-compose)
CMD ["/producer"]

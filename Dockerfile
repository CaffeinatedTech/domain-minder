# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o domain-minder ./cmd/server/

# Run stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/domain-minder .
COPY --from=builder /app/.env.example .

# Create data directory
RUN mkdir -p data

# Environment variables (can be overridden)
ENV DB_PATH=/app/data/domain_minder.db
ENV PORT=9000

# Expose port
EXPOSE 9000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9000/health || exit 1

# Run application
ENTRYPOINT ["./domain-minder"]

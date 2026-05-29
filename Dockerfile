# Build stage
FROM golang:1.26-alpine AS builder

# Install build dependencies for SQLite3
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies with caching
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled for SQLite3
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -installsuffix cgo -ldflags="-w -s" -o discord-bot-moderation cmd/app/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/discord-bot-moderation .

# Create the database if it doesn't exist
RUN mkdir -p /app/data

# Run the binary
CMD ["./discord-bot-moderation"]

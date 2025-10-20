# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies for SQLite3
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled for SQLite3
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o discord-bot-moderation cmd/app/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/discord-bot-moderation .

# Create directory for SQLite database
RUN mkdir -p /data

# Run the binary
CMD ["./discord-bot-moderation"]

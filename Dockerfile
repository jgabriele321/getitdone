FROM golang:1.24.2-alpine AS builder

# Set the working directory
WORKDIR /build

# Copy the Go module files first
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -tags netgo -ldflags '-s -w' -o app ./cmd/bot

# Start from a fresh Alpine image
FROM alpine:latest

# Install necessary packages
RUN apk --no-cache add ca-certificates tzdata

# Set environment variables
ENV TZ=UTC
ENV PORT=8080

# Create a non-root user
RUN adduser -D -s /bin/sh appuser

# Set the working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/app .

# Copy the environment example file
COPY env.example .env.example

# Set ownership
RUN chown -R appuser:appuser /app

# Use the non-root user
USER appuser

# Expose the port
EXPOSE ${PORT}

# Run the application
CMD ["./app"]
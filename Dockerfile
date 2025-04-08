# Build stage
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application with version information
ARG VERSION=0.1.0-dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Build with version information
RUN CGO_ENABLED=0 GOOS=linux go build \
  -ldflags "-X github.com/sammcj/mcp-package-version/pkg/version.Version=${VERSION} -X github.com/sammcj/mcp-package-version/pkg/version.Commit=${COMMIT} -X github.com/sammcj/mcp-package-version/pkg/version.BuildDate=${BUILD_DATE}" \
  -o mcp-package-version .

# Final stage
FROM alpine:latest

# Set working directory
WORKDIR /app

# Install CA certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/mcp-package-version .

# Expose port
EXPOSE 18080

# Run the application with SSE transport by default
CMD ["./mcp-package-version", "--transport", "sse", "--port", "18080", "--base-url", "http://0.0.0.0"]

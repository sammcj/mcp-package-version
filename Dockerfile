# Build stage
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies (make and git for versioning)
RUN apk add --no-cache make git

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code (including Makefile)
COPY . .

# Build the application using the Makefile
# CGO_ENABLED=0 and GOOS=linux ensure a static Linux binary for the final stage
RUN CGO_ENABLED=0 GOOS=linux make build

# Final stage
FROM alpine:latest

# The base url is where you want to point your clients at (don't include the /sse endpoint)
ARG BASE_URL="http://mcp-package-version"
ARG PORT="18080"
ENV BASE_URL=${BASE_URL}
ENV PORT=${PORT}

# Set default log level (can be overridden with -e LOG_LEVEL=debug)
ENV LOG_LEVEL=info

# Set working directory
WORKDIR /app

# Install CA certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage (using the path from Makefile)
COPY --from=builder /app/bin/mcp-package-version .

# Expose port
EXPOSE ${PORT}

# Run the application with SSE transport by default, using shell form for variable substitution
CMD ./mcp-package-version --transport sse --port ${PORT} --base-url ${BASE_URL}

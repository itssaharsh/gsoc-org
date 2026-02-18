# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy dependency files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# CGO_ENABLED=0 ensures a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o gsoc-app main.go

# Stage 2: Create the final lightweight image
FROM alpine:latest

WORKDIR /root/

# Install certificates for HTTPS requests to the GSoC API
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/gsoc-app .

# Copy templates directory
COPY --from=builder /app/templates ./templates

# Expose the application port
EXPOSE 8080

# Run the binary
CMD ["./gsoc-app"]
FROM golang:1.23.4-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o s3-bucket-browser ./cmd

# Use a smaller image for the final build
FROM alpine:latest

WORKDIR /app

# Install CA certificates for HTTPS connections
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/s3-bucket-browser .

# Expose the port
EXPOSE 8080

# Run the application
CMD ["./s3-bucket-browser"]
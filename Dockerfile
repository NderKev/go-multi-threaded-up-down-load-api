# Build Stage
FROM golang:1.18-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project directory to /app
COPY . .

# Build the Go application
RUN go build -o main .

# Runtime Stage
FROM alpine:latest

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Expose the port that the Go application will use
EXPOSE 8989

# Run the Go binary
CMD ["./main"]

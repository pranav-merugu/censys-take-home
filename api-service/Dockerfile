# Build stage
FROM golang:1.24.4-alpine AS builder

WORKDIR /app

# Copy go mod files from project root
COPY go.mod go.sum ./
RUN go mod download

# Copy entire project (needed for proto package and imports)
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api-service-bin ./api-service/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/api-service-bin ./api-service

# Expose HTTP port
EXPOSE 8080

# Run the service
CMD ["./api-service"]

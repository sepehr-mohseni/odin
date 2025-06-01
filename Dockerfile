FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /odin cmd/odin/main.go

# Use a minimal image for the final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /odin .

# Copy templates and static files
COPY --from=builder /app/pkg/admin/templates/ ./pkg/admin/templates/
COPY --from=builder /app/config/default_config.yaml ./config/config.yaml

# Create volume mount points
VOLUME ["/app/config"]

# Expose the ports
EXPOSE 8080
EXPOSE 8081

# Command to run
CMD ["./odin"]

# --------------------------
# 1. Build stage
# --------------------------
FROM golang:1.24-alpine AS builder

# Enable CGO to 0 for fully static binary
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build static binary
RUN go build -ldflags="-s -w" -o docker-logger main.go

# --------------------------
# 2. Minimal runtime stage
# --------------------------
FROM alpine:3.21

WORKDIR /app

COPY --from=builder /app/docker-logger /app/docker-logger

VOLUME ["/var/run/docker.sock", "/app/logs"]

ENV LOG_DIR=/app/logs
ENV MAX_SIZE_MB=10
ENV MAX_BACKUPS=5
ENV MAX_AGE_DAYS=31
ENV TARGET_CONTAINERS=""

ENTRYPOINT ["/app/docker-logger"]
# Build stage
FROM golang:1.21-bullseye AS builder

WORKDIR /build
COPY cmd/script/ ./
RUN go mod tidy
RUN go mod download

# Build snapshot program (excluding generate_encryption.go)
RUN go build -o snapshot snapshot.go logger.go

# Build key generation program (standalone)
RUN go build -o generate_encryption generate_encryption.go

# Runtime stage
FROM ubuntu:22.04

# Update packages and install dependencies
RUN apt-get update && apt-get install -y \
    cron \
    rsync \
    util-linux \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

# Create snapshots and keys directories
RUN mkdir -p /app/snapshots /app/keys

# Copy Go binaries from builder stage
COPY --from=builder /build/snapshot /app/snapshot
COPY --from=builder /build/generate_encryption /app/generate_encryption

# Copy scripts
COPY cronjob/cronjob.sh /app/cronjob.sh

# Make scripts executable
RUN chmod +x /app/snapshot /app/generate_encryption /app/cronjob.sh

# Copy cron configuration
COPY crontab /etc/cron.d/snapshot-cron

# Set proper permissions for cron file
RUN chmod 0644 /etc/cron.d/snapshot-cron

# Apply cron configuration
RUN crontab /etc/cron.d/snapshot-cron

# Create log file for cron
RUN touch /var/log/cron.log

WORKDIR /app

# Entry point
CMD ["/app/cronjob.sh"]

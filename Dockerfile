# Build stage
FROM golang:1.24-bullseye AS builder

WORKDIR /build

# Build snapshot program
COPY cmd/script/ ./script/
WORKDIR /build/script
RUN go mod tidy && go mod download
RUN go build -o snapshot snapshot.go logger.go crypto.go

# Build key generation program  
WORKDIR /build
COPY cmd/generate/ ./generate/
WORKDIR /build/generate
RUN go mod tidy && go mod download
RUN go build -o generate_encryption generate_encryption.go

# Build test program
WORKDIR /build  
COPY cmd/test/ ./test/
WORKDIR /build/test
RUN ls -la && cat go.mod  # Debug output
RUN go mod tidy && go mod download
RUN head -5 decrypt.go  # Show first few lines
RUN go build -v -o simple_decrypt_test .


# Runtime stage
FROM debian:bullseye

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
COPY --from=builder /build/script/snapshot /app/snapshot
COPY --from=builder /build/generate/generate_encryption /app/generate_encryption
COPY --from=builder /build/test/simple_decrypt_test /app/simple_decrypt_test

# Copy source files for runtime compilation
COPY cmd/script/ /app/cmd/script/

# Copy scripts
COPY cronjob/cronjob.sh /app/cronjob.sh

# Make scripts executable
RUN chmod +x /app/snapshot /app/generate_encryption /app/simple_decrypt_test /app/cronjob.sh

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

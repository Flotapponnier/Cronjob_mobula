# Build stage
FROM golang:1.24-bullseye AS builder

WORKDIR /build

# Build snapshot program
COPY cmd/script/ ./script/
WORKDIR /build/script
RUN go mod tidy && go mod download
RUN go build -o snapshot snapshot.go logger.go encryption_snapshot.go save_architectured_snapshot.go upload_cloud.go

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
RUN go build -v -o decrypt .


# Runtime stage
FROM debian:bullseye

# Set timezone to Europe/Paris
ENV TZ=Europe/Paris
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# Update packages and install dependencies including Google Cloud SDK and disk imaging tools
RUN apt-get update && apt-get install -y \
    cron \
    rsync \
    util-linux \
    ca-certificates \
    tzdata \
    curl \
    python3 \
    python3-pip \
    e2fsprogs \
    parted \
    genisoimage \
    isolinux \
    syslinux-common \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

# Install Google Cloud SDK manually
RUN curl -sSL https://sdk.cloud.google.com > /tmp/gcpsdk-install.sh && \
    bash /tmp/gcpsdk-install.sh --disable-prompts --install-dir=/opt && \
    rm /tmp/gcpsdk-install.sh
ENV PATH="/opt/google-cloud-sdk/bin:$PATH"

# Create disk_images and keys directories
RUN mkdir -p /app/disk_images /app/keys

# Copy Go binaries from builder stage
COPY --from=builder /build/script/snapshot /app/snapshot
COPY --from=builder /build/generate/generate_encryption /app/generate_encryption
COPY --from=builder /build/test/decrypt /app/decrypt

# Copy source files for runtime compilation
COPY cmd/script/ /app/cmd/script/

# Copy scripts and configuration
COPY cronjob/cronjob.sh /app/cronjob.sh
COPY .env /app/.env
COPY mobulacronjson.json /app/keys/mobulacronjson.json

# Make scripts executable
RUN chmod +x /app/snapshot /app/generate_encryption /app/decrypt /app/cronjob.sh

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

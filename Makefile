.PHONY: build up down stop destroy clean logs shell minio minio-down

# Docker settings
IMAGE_NAME := snapshot-cron
CONTAINER_NAME := snapshot-container

# Build the Docker image
build:
	@echo "Building Docker image..."
	docker build -t $(IMAGE_NAME) .

# Start the container
up: build
	@echo "Starting snapshot container..."
	@docker network create mobula-network 2>/dev/null || true
	@if docker ps -a --format "table {{.Names}}" | grep -q "^$(CONTAINER_NAME)$$"; then \
		echo "Container $(CONTAINER_NAME) already exists. Starting it..."; \
		docker start $(CONTAINER_NAME); \
	else \
		echo "Creating new container..."; \
		docker run -d --privileged --memory=16g --network mobula-network --name $(CONTAINER_NAME) $(IMAGE_NAME); \
	fi
	@echo "Container started. Use 'make logs' to view output."

# Stop the container (without removing)
down:
	@echo "Stopping container..."
	-docker stop $(CONTAINER_NAME)

# Stop the container (alias for down)
stop: down

# Destroy the container (with confirmation)
destroy:
	@echo "⚠️  WARNING: This will permanently delete the container!"
	@echo -n "Are you sure you want to destroy the container? [y/N]: "; \
	read answer; \
	if [ "$$answer" = "y" ] || [ "$$answer" = "Y" ]; then \
		echo "Stopping and removing container..."; \
		docker stop $(CONTAINER_NAME) 2>/dev/null || true; \
		docker rm $(CONTAINER_NAME) 2>/dev/null || true; \
		echo "Container destroyed."; \
	else \
		echo "Operation cancelled."; \
	fi

# View container logs
logs:
	docker logs -f $(CONTAINER_NAME)

# Get shell access to container
shell:
	docker exec -it $(CONTAINER_NAME) /bin/bash

# Clean up (remove container and image)
clean: destroy
	@echo "Cleaning up Docker image..."
	-docker rmi $(IMAGE_NAME)

# Show snapshot files
snapshots:
	docker exec $(CONTAINER_NAME) ls -la /app/snapshots

# Generate encryption keys and send shares
generate:
	@echo "Generating encryption keys and sending email shares..."
	docker exec -it $(CONTAINER_NAME) /app/generate_encryption

# Comprehensive encryption tests
test:
	@echo "🧪 Running comprehensive encryption tests..."
	@echo ""
	@echo "1. Checking master key exists:"
	@docker exec $(CONTAINER_NAME) cat /app/keys/master.key
	@echo ""
	@echo "2. Checking key info file:"
	@docker exec $(CONTAINER_NAME) cat /app/keys/key_info.json
	@echo ""
	@echo "3. Checking if snapshots exist:"
	@docker exec $(CONTAINER_NAME) find /app/snapshots -name "*.encrypted" | head -3 || echo "No snapshots found yet"
	@echo ""
	@echo "4. Running 'hello world!' decryption test:"
	@echo "   You'll be asked for key shares. If successful, you should see 'hello world!'"
	@echo ""
	@docker exec -it $(CONTAINER_NAME) /app/decrypt

# Interactive snapshot decryption
decrypt:
	@echo "🔓 Interactive Snapshot Decryption"
	@echo "=================================="
	@echo "This will decrypt a snapshot file."
	@echo ""
	@docker exec -it $(CONTAINER_NAME) /app/decrypt snapshot

# Start MinIO for testing
minio:
	@echo "Starting MinIO S3 server..."
	docker-compose -f docker-compose.minio.yml up -d
	@echo ""
	@echo "✅ MinIO started!"
	@echo ""
	@echo "📋 Next steps:"
	@echo "  1. Open http://localhost:9001 in your browser"
	@echo "  2. Login: minioadmin / minioadmin123"
	@echo "  3. Create a bucket named: mobula-backups"
	@echo "  4. Run: make minio-config"
	@echo ""

# Stop MinIO
minio-down:
	@echo "Stopping MinIO..."
	docker-compose -f docker-compose.minio.yml down

# Configure .env for MinIO
minio-config:
	@echo "Configuring .env for MinIO..."
	@sed -i.bak 's|S3_ENABLED=.*|S3_ENABLED=true|' .env
	@sed -i.bak 's|S3_ENDPOINT=.*|S3_ENDPOINT=http://minio-test:9000|' .env
	@sed -i.bak 's|S3_REGION=.*|S3_REGION=us-east-1|' .env
	@sed -i.bak 's|S3_ACCESS_KEY_ID=.*|S3_ACCESS_KEY_ID=minioadmin|' .env
	@sed -i.bak 's|S3_SECRET_ACCESS_KEY=.*|S3_SECRET_ACCESS_KEY=minioadmin123|' .env
	@sed -i.bak 's|S3_BUCKET_NAME=.*|S3_BUCKET_NAME=mobula-backups|' .env
	@rm .env.bak
	@echo "✅ .env configured for MinIO!"
	@echo ""
	@echo "You can now run: make up"

# Configure .env for OVH
ovh-config:
	@echo "Configuring .env for OVH S3..."
	@sed -i.bak 's|S3_ENABLED=.*|S3_ENABLED=false|' .env
	@sed -i.bak 's|S3_ENDPOINT=.*|S3_ENDPOINT=https://s3.gra.io.cloud.ovh.net|' .env
	@sed -i.bak 's|S3_REGION=.*|S3_REGION=gra|' .env
	@sed -i.bak 's|S3_ACCESS_KEY_ID=.*|S3_ACCESS_KEY_ID=your-access-key-id|' .env
	@sed -i.bak 's|S3_SECRET_ACCESS_KEY=.*|S3_SECRET_ACCESS_KEY=your-secret-access-key|' .env
	@sed -i.bak 's|S3_BUCKET_NAME=.*|S3_BUCKET_NAME=your-bucket-name|' .env
	@rm .env.bak
	@echo "✅ .env configured for OVH S3!"
	@echo ""
	@echo "⚠️  Don't forget to update your credentials in .env"

# Show help
help:
	@echo "Available commands:"
	@echo "  build        - Build the Docker image"
	@echo "  up           - Build and start the container"
	@echo "  down         - Stop the container (without removing)"
	@echo "  stop         - Stop the container (alias for down)"
	@echo "  destroy      - Stop and remove the container (with confirmation)"
	@echo "  logs         - View container logs"
	@echo "  shell        - Get shell access to container"
	@echo "  snapshots    - List snapshot files"
	@echo "  generate     - Generate encryption keys and send shares"
	@echo "  test         - Comprehensive encryption test + interactive decryption"
	@echo "  decrypt      - Interactive snapshot decryption with decompression"
	@echo "  clean        - Remove container and image (calls destroy)"
	@echo "  minio        - Start MinIO S3 server for testing"
	@echo "  minio-down   - Stop MinIO S3 server"
	@echo "  minio-config - Configure .env for MinIO"
	@echo "  ovh-config   - Configure .env for OVH S3"
	@echo "  help         - Show this help message"
	@echo ""
	@echo "Note: 'down' and 'stop' only stop the container."
	@echo "      Use 'destroy' to permanently remove it."
	@echo "      Use 'clean' to remove both container and image."

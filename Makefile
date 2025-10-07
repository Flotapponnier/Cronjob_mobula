.PHONY: build up down clean logs shell

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
	docker run -d --name $(CONTAINER_NAME) $(IMAGE_NAME)
	@echo "Container started. Use 'make logs' to view output."

# Stop and remove the container
down:
	@echo "Stopping and removing container..."
	-docker stop $(CONTAINER_NAME)
	-docker rm $(CONTAINER_NAME)

# View container logs
logs:
	docker logs -f $(CONTAINER_NAME)

# Get shell access to container
shell:
	docker exec -it $(CONTAINER_NAME) /bin/bash

# Clean up (remove container and image)
clean: down
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

# Interactive snapshot decryption with decompression
decrypt:
	@echo "🔓 Interactive Snapshot Decryption"
	@echo "=================================="
	@echo "This will decrypt a snapshot and optionally decompress it."
	@echo ""
	docker exec -it $(CONTAINER_NAME) /app/decrypt snapshot

# Show help
help:
	@echo "Available commands:"
	@echo "  build     - Build the Docker image"
	@echo "  up        - Build and start the container"
	@echo "  down      - Stop and remove the container"
	@echo "  logs      - View container logs"
	@echo "  shell     - Get shell access to container"
	@echo "  snapshots - List snapshot files"
	@echo "  generate  - Generate encryption keys and send shares"
	@echo "  test      - Comprehensive encryption test + interactive decryption"
	@echo "  decrypt   - Interactive snapshot decryption with decompression"
	@echo "  clean     - Remove container and image"
	@echo "  help      - Show this help message"

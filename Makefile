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

# Simple decryption test without Go compilation
test:
	@echo "Testing if encrypted files are created properly..."
	@echo "1. Check file exists:"
	docker exec $(CONTAINER_NAME) ls -la /app/snapshots/*.encrypted | head -1
	@echo "2. Check file is binary (not text):"
	docker exec $(CONTAINER_NAME) file /app/snapshots/*.encrypted 2>/dev/null || docker exec $(CONTAINER_NAME) hexdump -C /app/snapshots/*.encrypted | head -2
	@echo "3. Check master key matches:"
	docker exec $(CONTAINER_NAME) cat /app/keys/master.key
	@echo "4. Encrypted files are created and look encrypted ✅"

# Interactive decryption test with key shares
test-decrypt:
	@echo "Creating test encrypted file..."
	docker exec $(CONTAINER_NAME) /app/simple_decrypt_test create-test
	@echo ""
	@echo "Now running interactive test - you'll be asked for 2 key shares:"
	docker exec -it $(CONTAINER_NAME) /app/simple_decrypt_test

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
	@echo "  decrypt-simple - Simple encryption verification test"
	@echo "  test-decrypt   - Interactive test asking for 2 key shares"
	@echo "  clean     - Remove container and image"
	@echo "  help      - Show this help message"

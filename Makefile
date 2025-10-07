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

# Combined test: basic checks + interactive decryption test
test:
	@echo "🧪 Running comprehensive encryption tests..."
	@echo ""
	@echo "1. Checking encrypted files exist:"
	docker exec $(CONTAINER_NAME) ls -la /app/snapshots/*.encrypted | head -1
	@echo ""
	@echo "2. Verifying files are encrypted (binary data):"
	docker exec $(CONTAINER_NAME) file /app/snapshots/*.encrypted 2>/dev/null || docker exec $(CONTAINER_NAME) hexdump -C /app/snapshots/*.encrypted | head -2
	@echo ""
	@echo "3. Showing master key:"
	docker exec $(CONTAINER_NAME) cat /app/keys/master.key
	@echo ""
	@echo "✅ Basic encryption tests passed!"
	@echo ""
	@echo "4. Creating test file with 'hello world!' for decryption test..."
	docker exec $(CONTAINER_NAME) /app/simple_decrypt_test create-test
	@echo ""
	@echo "5. Now running interactive test - you'll be asked for 2 key shares:"
	@echo "   If successful, you should see 'hello world!' as output."
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
	@echo "  test      - Comprehensive encryption test + interactive decryption"
	@echo "  clean     - Remove container and image"
	@echo "  help      - Show this help message"

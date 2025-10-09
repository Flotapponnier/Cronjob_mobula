# Mobula - Encrypted Snapshot System

A secure, automated snapshot system that creates encrypted backups and optionally uploads them to Google Cloud Storage. The system uses Shamir's Secret Sharing for key management, ensuring that no single point of failure can compromise your encrypted data.

## Quick Start

### 1. Build and Start
```bash
make up
```
This builds the Docker image and starts the container. The system will begin taking snapshots every 5 minutes.

### 2. Generate Encryption Keys
```bash
make generate
```
Generate encryption keys using Shamir's Secret Sharing. You'll receive key shares that must be stored securely and separately.

### 3. View Logs
```bash
make logs
```
Monitor the system's activity and snapshot creation in real-time.

## Makefile Commands

### Core Operations
- **`make build`** - Build the Docker image
- **`make up`** - Build and start the container (or restart if already exists)
- **`make down`** / **`make stop`** - Stop the container (preserves data)
- **`make logs`** - View real-time container logs (Ctrl+C to exit)

### Container Management
- **`make destroy`** - Stop and permanently remove the container (asks for confirmation)
- **`make clean`** - Remove both container and Docker image (calls destroy first)
- **`make shell`** - Get interactive shell access to the running container

### Key Management & Testing
- **`make generate`** - Generate new encryption keys and Shamir shares
- **`make test`** - Run comprehensive encryption tests
- **`make decrypt`** - Interactive snapshot decryption tool

### Utilities
- **`make snapshots`** - List current snapshot files
- **`make help`** - Show all available commands

## Container Lifecycle

### Starting Fresh
```bash
make up          # Creates and starts new container
make generate    # Generate encryption keys
make logs        # Monitor activity
```

### Daily Operations
```bash
make down        # Stop container (keeps data)
make up          # Restart container (same data)
make logs        # Check logs
```

### Complete Reset
```bash
make destroy     # Remove container (asks confirmation)
make clean       # Remove container + image
make up          # Start completely fresh
```

## Important Notes

### Data Persistence
- **`make down`** only **stops** the container - all keys, snapshots, and configuration are preserved
- **`make up`** will restart the existing container with all data intact
- **`make destroy`** permanently deletes everything - use with caution!

### Key Management
- Always run `make generate` after `make destroy` to create new encryption keys
- Store the generated key shares securely and separately
- The test file is automatically cleaned when new keys are generated

### Monitoring
- Use `make logs` to monitor snapshot creation and cloud uploads
- Snapshots are organized in `/app/snapshots/DD/MM/HH/` structure
- Failed operations are logged in red, successful ones in green

## Configuration (.env file)

### Shamir Secret Sharing Configuration
```bash
SHAMIR_TOTAL_SHARES=3    # Total number of key shares to generate
SHAMIR_THRESHOLD=3       # Minimum shares needed to decrypt
```

**Why Shamir's Secret Sharing?**
We use [HashiCorp Vault's Shamir implementation](https://github.com/hashicorp/vault/shamir) for cryptographic key splitting. This algorithm divides the master encryption key into multiple shares where:
- **Security**: No single share can decrypt data alone
- **Redundancy**: You can lose some shares and still recover data
- **Distributed Trust**: Different people/systems can hold different shares
- **Industry Standard**: Used by HashiCorp Vault for production security

### Encryption & Storage Paths
```bash
KEY_DIR=/app/keys                    # Directory for encryption keys
KEY_FILENAME=master.key              # Master key filename
SNAPSHOT_DIR=/app/snapshots          # Where encrypted snapshots are stored
SNAPSHOT_SOURCE_PATH=/app            # Source directory to backup
TEST_FILE=/app/test_hello.encrypted  # Test file for encryption validation
```

### Data Retention
```bash
DAY_RETENTION=7    # Remove snapshots older than N days (0 = keep forever)
```

### Google Cloud Storage (Optional)
```bash
GCP_ENABLED=true                                    # Enable cloud uploads
GCP_PROJECT_ID=your-project-id                     # GCP project
GCP_BUCKET_NAME=your-bucket-name                   # Storage bucket
GCP_SERVICE_ACCOUNT_KEY_FILE=service-account.json  # Auth credentials
GCP_BUCKET_PREFIX=snapshots                        # Folder prefix in bucket
```

## Encryption Technology

### Why AES-GCM?
- **AES-256**: Industry-standard symmetric encryption
- **GCM Mode**: Provides both encryption and authentication
- **Tamper Detection**: Any modification to encrypted data is detected
- **Performance**: Fast encryption/decryption for large files

### Security Architecture
1. **Random Master Key**: 256-bit key generated with crypto/rand
2. **Shamir Splitting**: Master key split into configurable shares
3. **AES-GCM Encryption**: Snapshots encrypted with authenticated encryption
4. **Secure Storage**: Encrypted snapshots can be stored anywhere safely

## Encryption Testing

### test_encryption/ Folder
A standalone testing tool to verify your encryption strength and recover your snapshots:

![Encryption Test Tool](images/tester.png)

#### Manual Decryption Test (Option 1)
1. Copy your `.encrypted` snapshot files to the `test_encryption/` folder
2. Run `cd test_encryption && go run main.go`
3. Choose **Option 1** - Manual decryption with your 3 key shares
4. Enter the filename of your encrypted snapshot
5. Enter your 3 key shares when prompted

**Result**: The tool will:
- Decrypt your snapshot using the 3 key shares
- Save the decrypted data as `filename_decrypted.iso.gz`
- Automatically decompress it to `filename_decrypted.iso`
- Create a bootable ISO ready for use in VMs

#### Accessing Your Backup Data
Once decrypted, you can extract and browse your backup:

```bash
# Extract ISO contents (recommended for macOS)
brew install p7zip
7z x filename_decrypted.iso -o./extracted_backup/

# Browse your backup files
ls -la ./extracted_backup/

# Check snapshot creation info
cat ./extracted_backup/snapshot_info/disk_image_info.txt
```

The `disk_image_info.txt` file contains:
- Exact timestamp when the snapshot was encrypted
- Source filesystem information
- Encryption method details
- Recovery instructions

#### Brute Force Security Test (Option 2)
- **Concurrent Attack Simulation**: Tries millions of random key combinations
- **Proves Uncrackability**: Demonstrates that your encryption is mathematically secure
- **Performance Metrics**: Shows attack rate and estimated time to crack (infinity)

#### Other Options
- **Option 3**: Refresh file list to see available encrypted snapshots
- **Option 4**: Test ISO contents and get mounting instructions
- **Option 5**: Exit the tool

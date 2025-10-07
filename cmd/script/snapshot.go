package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	snapshotDir = "/app/snapshots"
	keyFile     = "/app/keys/master.key"
)

func main() {
	// Check if encryption key exists
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		logError("No encryption key found. Use 'make generate' to create a key and start")
		return
	}

	// Load the master key
	masterKey, err := loadMasterKey()
	if err != nil {
		logError("Failed to load master key: %v", err)
		return
	}

	// Create architectured snapshot path and name
	snapshotPath, snapshotName, err := createArchitecturedSnapshot()
	if err != nil {
		logError("Failed to create snapshot architecture: %v", err)
		return
	}
	
	logInfo("Starting encrypted snapshot %s", snapshotName)
	
	// Create temporary directory for this snapshot (before encryption)
	tempSnapshotPath := snapshotPath + "_temp"
	if err := os.MkdirAll(tempSnapshotPath, 0755); err != nil {
		logError("Failed to create temp snapshot path: %v", err)
		return
	}
	
	// Copy important files using rsync
	backupPath := filepath.Join(tempSnapshotPath, "app_backup")
	if err := copyFiles(backupPath); err != nil {
		logError("Failed to copy files: %v", err)
		return
	}
	
	// Add metadata
	if err := createMetadata(tempSnapshotPath); err != nil {
		logError("Failed to create metadata: %v", err)
		return
	}
	
	// Encrypt the snapshot
	encryptedPath := snapshotPath + ".encrypted"
	if err := encryptSnapshot(tempSnapshotPath, encryptedPath, masterKey); err != nil {
		logError("Failed to encrypt snapshot: %v", err)
		return
	}
	
	// Remove the temporary unencrypted directory
	if err := os.RemoveAll(tempSnapshotPath); err != nil {
		logError("Failed to remove temp snapshot: %v", err)
	}
	
	// Show snapshot statistics
	getSnapshotStats()
	
	logInfo("Encrypted snapshot %s has been saved: %s", snapshotName, encryptedPath)
}

func copyFiles(backupPath string) error {
	// Ensure backup directory exists
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}
	
	// Use rsync with more robust flags
	cmd := exec.Command("rsync", "-a", "--exclude=snapshots", "--ignore-errors", "/app/", backupPath+"/")
	
	// Run command and capture output for better error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		logError("Rsync failed with output: %s", string(output))
		return err
	}
	
	return nil
}

func createMetadata(snapshotPath string) error {
	metadataPath := filepath.Join(snapshotPath, "metadata.txt")
	file, err := os.Create(metadataPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	hostname, _ := os.Hostname()
	
	fmt.Fprintf(file, "Snapshot created on: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(file, "Hostname: %s\n", hostname)
	
	// Get uptime
	if uptimeBytes, err := os.ReadFile("/proc/uptime"); err == nil {
		fmt.Fprintf(file, "Uptime: %s", string(uptimeBytes))
	}
	
	return nil
}



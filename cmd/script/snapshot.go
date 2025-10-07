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
)

func main() {
	timestamp := time.Now().Format("20060102_150405")
	snapshotName := fmt.Sprintf("disk_snapshot_%s", timestamp)
	
	// Create snapshot directory if it doesn't exist
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		logError("Failed to create snapshot directory: %v", err)
		return
	}
	
	logInfo("Starting snapshot %s", snapshotName)
	
	// Create directory for this snapshot
	snapshotPath := filepath.Join(snapshotDir, snapshotName)
	if err := os.MkdirAll(snapshotPath, 0755); err != nil {
		logError("Failed to create snapshot path: %v", err)
		return
	}
	
	// Copy important files using rsync
	backupPath := filepath.Join(snapshotPath, "app_backup")
	if err := copyFiles(backupPath); err != nil {
		logError("Failed to copy files: %v", err)
		return
	}
	
	// Add metadata
	if err := createMetadata(snapshotPath); err != nil {
		logError("Failed to create metadata: %v", err)
		return
	}
	
	logInfo("Snapshot %s has been generated in the folder %s", snapshotName, snapshotPath)
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



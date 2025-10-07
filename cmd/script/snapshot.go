package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	
	// Check and apply retention policy
	checkRetentionPolicy()
	
	logInfo("Encrypted snapshot %s has been saved: %s", snapshotName, encryptedPath)
}

func copyFiles(backupPath string) error {
	// Get source path from config
	sourcePath := getSourcePath()
	
	// Ensure backup directory exists
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}
	
	// Use rsync with more robust flags
	cmd := exec.Command("rsync", "-a", "--exclude=snapshots", "--ignore-errors", sourcePath+"/", backupPath+"/")
	
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

// getSourcePath reads the source path from .env file
func getSourcePath() string {
	defaultPath := "/app"
	
	envFile := "/app/.env"
	file, err := os.Open(envFile)
	if err != nil {
		return defaultPath
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		if key == "SNAPSHOT_SOURCE_PATH" && value != "" {
			return value
		}
	}
	
	return defaultPath
}

// checkRetentionPolicy checks if cleanup is needed based on DAY_RETENTION setting
func checkRetentionPolicy() {
	retentionDays := getRetentionDays()
	if retentionDays <= 0 {
		return // No cleanup if retention is 0 or negative
	}
	
	// Check if we need to clean up old snapshots
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	logInfo("🗑️ Checking retention policy: removing snapshots older than %d days", retentionDays)
	
	removed := 0
	totalSize := int64(0)
	
	// Walk through all snapshots and remove old ones
	err := filepath.Walk(snapshotDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}
		
		// Only process encrypted snapshot files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".encrypted") {
			if info.ModTime().Before(cutoffTime) {
				totalSize += info.Size()
				if err := os.Remove(path); err != nil {
					logError("Failed to remove old snapshot %s: %v", path, err)
				} else {
					removed++
					logInfo("🗑️ Removed old snapshot: %s", filepath.Base(path))
				}
			}
		}
		
		return nil
	})
	
	if err != nil {
		logError("Failed to check retention policy: %v", err)
		return
	}
	
	if removed > 0 {
		logInfo("✅ Retention cleanup complete: removed %d snapshots (%.2f MB freed)", removed, float64(totalSize)/1024/1024)
		
		// Remove empty directories after cleanup
		removeEmptyDirs(snapshotDir)
	}
}

// getRetentionDays reads retention policy from .env file
func getRetentionDays() int {
	defaultRetention := 0 // No cleanup by default
	
	envFile := "/app/.env"
	file, err := os.Open(envFile)
	if err != nil {
		return defaultRetention
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		if key == "DAY_RETENTION" {
			if days, err := strconv.Atoi(value); err == nil && days >= 0 {
				return days
			}
		}
	}
	
	return defaultRetention
}

// removeEmptyDirs removes empty directories recursively
func removeEmptyDirs(root string) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == root {
			return err
		}
		
		if info.IsDir() {
			// Try to remove if empty
			if err := os.Remove(path); err == nil {
				logInfo("🗂️ Removed empty directory: %s", strings.TrimPrefix(path, root+"/"))
			}
		}
		
		return nil
	})
}



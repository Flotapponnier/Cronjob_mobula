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

var (
	snapshotDir string
	keyFile     string
)

func main() {
	loadConfig()

	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		logError("No encryption key found. Use 'make generate' to create a key and start")
		return
	}

	masterKey, err := loadMasterKey()
	if err != nil {
		logError("Failed to load master key: %v", err)
		return
	}

	snapshotPath, snapshotName, err := createArchitecturedSnapshot()
	if err != nil {
		logError("Failed to create snapshot architecture: %v", err)
		return
	}

	logInfo("Starting encrypted snapshot %s", snapshotName)

	tempSnapshotPath := snapshotPath + "_temp"
	if err := os.MkdirAll(tempSnapshotPath, 0755); err != nil {
		logError("Failed to create temp snapshot path: %v", err)
		return
	}

	backupPath := filepath.Join(tempSnapshotPath, "app_backup")
	if err := copyFiles(backupPath); err != nil {
		logError("Failed to copy files: %v", err)
		return
	}

	if err := createMetadata(tempSnapshotPath); err != nil {
		logError("Failed to create metadata: %v", err)
		return
	}

	encryptedPath := snapshotPath + ".encrypted"
	if err := encryptSnapshot(tempSnapshotPath, encryptedPath, masterKey); err != nil {
		logError("Failed to encrypt snapshot: %v", err)
		return
	}

	if err := os.RemoveAll(tempSnapshotPath); err != nil {
		logError("Failed to remove temp snapshot: %v", err)
	}

	logSectionStart("📊 Snapshot Organization Stats")
	getSnapshotStatsContent()
	logSectionEnd()

	checkRetentionPolicy()

	uploadToCloud(encryptedPath, snapshotName)

	logInfo("Encrypted snapshot %s has been saved: %s", snapshotName, encryptedPath)
}

func loadConfig() {
	snapshotDir = "/app/snapshots"
	keyDir := "/app/keys"
	keyFilename := "master.key"

	envFile := "/app/.env"
	file, err := os.Open(envFile)
	if err != nil {
		logInfo("No .env file found, using default paths")
		keyFile = filepath.Join(keyDir, keyFilename)
		return
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

		switch key {
		case "SNAPSHOT_DIR":
			if value != "" {
				snapshotDir = value
			}
		case "KEY_DIR":
			if value != "" {
				keyDir = value
			}
		case "KEY_FILENAME":
			if value != "" {
				keyFilename = value
			}
		}
	}

	keyFile = filepath.Join(keyDir, keyFilename)
}

func copyFiles(backupPath string) error {
	sourcePath := getSourcePath()

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	cmd := exec.Command("rsync", "-a", "--exclude=snapshots", "--ignore-errors", sourcePath+"/", backupPath+"/")

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

	if uptimeBytes, err := os.ReadFile("/proc/uptime"); err == nil {
		fmt.Fprintf(file, "Uptime: %s", string(uptimeBytes))
	}

	return nil
}

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

func checkRetentionPolicy() {
	retentionDays := getRetentionDays()
	if retentionDays <= 0 {
		return
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	logInfo("🗑️ Checking retention policy: removing snapshots older than %d days", retentionDays)

	removed := 0
	totalSize := int64(0)

	err := filepath.Walk(snapshotDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

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

		removeEmptyDirs(snapshotDir)
	}
}

func getRetentionDays() int {
	defaultRetention := 0

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

func removeEmptyDirs(root string) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == root {
			return err
		}

		if info.IsDir() {
			if err := os.Remove(path); err == nil {
				logInfo("🗂️ Removed empty directory: %s", strings.TrimPrefix(path, root+"/"))
			}
		}

		return nil
	})
}

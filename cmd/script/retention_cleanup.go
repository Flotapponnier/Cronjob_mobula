package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRetentionDays = 0
)

func checkRetentionPolicy() {
	retentionDays := getRetentionDays()
	if retentionDays <= 0 {
		return
	}

	logInfo("🗑️ Checking retention policy: removing disk images older than %d days", retentionDays)

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	removed := 0
	var totalSize int64

	err := filepath.Walk(diskImageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".encrypted") {
			if info.ModTime().Before(cutoffTime) {
				totalSize += info.Size()
				if err := os.Remove(path); err != nil {
					logError("Failed to remove old disk image %s: %v", path, err)
				} else {
					removed++
					logInfo("🗑️ Removed old disk image: %s", filepath.Base(path))
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
		logInfo("✅ Retention cleanup complete: removed %d disk images (%.2f MB freed)", removed, float64(totalSize)/1024/1024)

		removeEmptyDirs(diskImageDir)
	}
}

func getRetentionDays() int {
	defaultRetention := defaultRetentionDays

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

		if key == "DAY_RETENTION" && value != "" {
			if days, err := strconv.Atoi(value); err == nil && days >= 0 {
				return days
			}
		}
	}

	return defaultRetention
}

func removeEmptyDirs(root string) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() || path == root {
			return nil
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}

		if len(entries) == 0 {
			if err := os.Remove(path); err == nil {
				logInfo("🗑️ Removed empty directory: %s", path)
			}
		}

		return nil
	})
}


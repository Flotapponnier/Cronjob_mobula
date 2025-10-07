package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func createArchitecturedSnapshot() (string, string, error) {
	now := time.Now()

	year := fmt.Sprintf("%04d", now.Year())
	day := fmt.Sprintf("%02d", now.Day())
	month := fmt.Sprintf("%02d", int(now.Month()))
	hour := fmt.Sprintf("%02d", now.Hour())

	snapshotDirPath := filepath.Join(snapshotDir, year, day, month, hour)
	if err := os.MkdirAll(snapshotDirPath, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create snapshot directory structure: %v", err)
	}

	timestamp := now.Format("02012006_1504")
	snapshotName := fmt.Sprintf("snapshot_%s", timestamp)
	snapshotPath := filepath.Join(snapshotDirPath, snapshotName)

	logInfo("Created snapshot directory structure: %s", snapshotDirPath)
	logInfo("Snapshot will be saved as: %s", snapshotName)

	return snapshotPath, snapshotName, nil
}

func getSnapshotStatsContent() {
	yearFolders := 0
	dayFolders := 0
	hourFolders := 0
	totalSnapshots := 0

	err := filepath.Walk(snapshotDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() && path != snapshotDir {
			relPath, _ := filepath.Rel(snapshotDir, path)
			pathParts := filepath.SplitList(filepath.ToSlash(relPath))

			switch len(pathParts) {
			case 1:
				yearFolders++
			case 2:
				dayFolders++
			case 3:
			case 4:
				hourFolders++
			}
		} else if !info.IsDir() && filepath.Ext(info.Name()) == ".encrypted" {
			totalSnapshots++
		}

		return nil
	})

	if err != nil {
		logError("Failed to calculate stats: %v", err)
		return
	}

	logInfo("📅 Years with snapshots: %d", yearFolders)
	logInfo("📁 Days with snapshots: %d", dayFolders)
	logInfo("⏰ Hour folders: %d", hourFolders)
	logInfo("📸 Total encrypted snapshots: %d", totalSnapshots)
}

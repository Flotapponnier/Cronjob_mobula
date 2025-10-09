package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	diskImageBaseName = "disk_image"
)

func createArchitecturedDiskImage() (string, string, error) {
	return createArchitecturedDiskImageWithTime(time.Now())
}

func createArchitecturedDiskImageWithTime(now time.Time) (string, string, error) {

	year := fmt.Sprintf("%04d", now.Year())
	day := fmt.Sprintf("%02d", now.Day())
	month := fmt.Sprintf("%02d", int(now.Month()))
	hour := fmt.Sprintf("%02d", now.Hour())

	diskImageDirPath := filepath.Join(diskImageDir, year, day, month, hour)
	if err := os.MkdirAll(diskImageDirPath, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create disk image directory structure: %v", err)
	}

	timestamp := now.Format("02012006_1504")
	diskImageName := fmt.Sprintf("%s_%s", diskImageBaseName, timestamp)
	diskImagePath := filepath.Join(diskImageDirPath, diskImageName)

	logInfo("Created disk image directory structure: %s", diskImageDirPath)
	logInfo("Disk image will be saved as: %s", diskImageName)

	return diskImagePath, diskImageName, nil
}

func getDiskImageStatsContent() {
	yearFolders := 0
	dayFolders := 0
	hourFolders := 0
	totalDiskImages := 0

	err := filepath.Walk(diskImageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() && path != diskImageDir {
			relPath, _ := filepath.Rel(diskImageDir, path)
			pathParts := strings.Split(filepath.ToSlash(relPath), "/")

			switch len(pathParts) {
			case 1:
				yearFolders++
			case 2:
				dayFolders++
			case 3:
			case 4:
				hourFolders++
			}
		} else if !info.IsDir() && strings.HasSuffix(info.Name(), ".encrypted") {
			totalDiskImages++
		}

		return nil
	})

	if err != nil {
		logError("Failed to calculate stats: %v", err)
		return
	}

	logInfo("📅 Years with disk images: %d", yearFolders)
	logInfo("📁 Days with disk images: %d", dayFolders)
	logInfo("⏰ Hour folders: %d", hourFolders)
	logInfo("💽 Total encrypted disk images: %d", totalDiskImages)
}

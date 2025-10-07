package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	baseSnapshotDir = "/app/snapshots"
)

// createArchitecturedSnapshot creates snapshot with organized folder structure
// Structure: /app/snapshots/DD/MM/HH/snapshot_DDMMYYYY_HHMMSS.encrypted
func createArchitecturedSnapshot() (string, string, error) {
	now := time.Now()
	
	// Create folder structure: DD/MM/HH
	day := fmt.Sprintf("%02d", now.Day())
	month := fmt.Sprintf("%02d", int(now.Month()))
	hour := fmt.Sprintf("%02d", now.Hour())
	
	// Create the full directory path
	snapshotDir := filepath.Join(baseSnapshotDir, day, month, hour)
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create snapshot directory structure: %v", err)
	}
	
	// Generate snapshot name with date and time (HHMM format)
	timestamp := now.Format("02012006_1504") // DDMMYYYY_HHMM
	snapshotName := fmt.Sprintf("snapshot_%s", timestamp)
	snapshotPath := filepath.Join(snapshotDir, snapshotName)
	
	logInfo("Created snapshot directory structure: %s", snapshotDir)
	logInfo("Snapshot will be saved as: %s", snapshotName)
	
	return snapshotPath, snapshotName, nil
}

// getSnapshotStats returns statistics about the current snapshot organization
func getSnapshotStats() {
	logInfo("📊 Snapshot Organization Stats:")
	getSnapshotStatsContent()
}

// getSnapshotStatsContent shows the statistics without the header
func getSnapshotStatsContent() {
	// Walk through the base directory and count folders
	dayFolders := 0
	hourFolders := 0
	totalSnapshots := 0
	
	err := filepath.Walk(baseSnapshotDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}
		
		if info.IsDir() && path != baseSnapshotDir {
			relPath, _ := filepath.Rel(baseSnapshotDir, path)
			pathParts := filepath.SplitList(filepath.ToSlash(relPath))
			
			switch len(pathParts) {
			case 1: // Day folder (DD)
				dayFolders++
			case 2: // Month folder (DD/MM) - not counted separately  
			case 3: // Hour folder (DD/MM/HH)
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
	
	logInfo("📁 Days with snapshots: %d", dayFolders)
	logInfo("⏰ Hour folders: %d", hourFolders)
	logInfo("📸 Total encrypted snapshots: %d", totalSnapshots)
}


// listSnapshotsByDate lists all snapshots organized by date
func listSnapshotsByDate() {
	logInfo("📅 Snapshots organized by date:")
	
	err := filepath.Walk(baseSnapshotDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		if info.IsDir() && path != baseSnapshotDir {
			relPath, _ := filepath.Rel(baseSnapshotDir, path)
			pathParts := filepath.SplitList(filepath.ToSlash(relPath))
			
			if len(pathParts) == 3 { // This is an hour folder (DD/MM/HH)
				// Count snapshots in this hour folder
				pattern := filepath.Join(path, "*.encrypted")
				matches, _ := filepath.Glob(pattern)
				
				if len(matches) > 0 {
					day, month, hour := pathParts[0], pathParts[1], pathParts[2]
					logInfo("📂 %s/%s at %s:00 - %d snapshots", day, month, hour, len(matches))
					
					// Show individual snapshots
					for _, match := range matches {
						filename := filepath.Base(match)
						stat, _ := os.Stat(match)
						sizeKB := stat.Size() / 1024
						logInfo("    📄 %s (%d KB)", filename, sizeKB)
					}
				}
			}
		}
		
		return nil
	})
	
	if err != nil {
		logError("Failed to list snapshots: %v", err)
	}
}
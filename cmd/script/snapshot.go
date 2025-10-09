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
	diskImageDir string
	keyFile      string
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

	diskImagePath, diskImageName, err := createArchitecturedDiskImage()
	if err != nil {
		logError("Failed to create disk image path: %v", err)
		return
	}

	logInfo("Starting encrypted OS disk image %s", diskImageName)

	isoPath := diskImagePath + ".iso.gz"
	if err := createCompressedISO(isoPath); err != nil {
		logError("Failed to create compressed ISO: %v", err)
		return
	}

	encryptedDiskPath := diskImagePath + ".encrypted"
	if err := encryptDiskImage(isoPath, encryptedDiskPath, masterKey); err != nil {
		logError("Failed to encrypt ISO: %v", err)
		return
	}

	if err := os.Remove(isoPath); err != nil {
		logError("Failed to remove ISO: %v", err)
	}

	logSectionStart("💽 Disk Image Stats")
	getDiskImageStatsContent()
	logSectionEnd()

	checkRetentionPolicy()

	uploadToCloud(encryptedDiskPath, diskImageName)

	if err := updateSnapshotInfoFile(diskImageName, encryptedDiskPath); err != nil {
		logError("Failed to update snapshot info file: %v", err)
	}

	logInfo("Encrypted disk image %s has been saved: %s", diskImageName, encryptedDiskPath)
}

func loadConfig() {
	diskImageDir = "/app/disk_images"
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
		case "DISK_IMAGE_DIR":
			if value != "" {
				diskImageDir = value
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

func createCompressedISO(isoPath string) error {
	logInfo("Creating compressed ISO from filesystem...")

	tempISO := "/tmp/temp.iso"
	tempDir := "/tmp/iso_content"
	
	// Create temp directory
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// Copy filesystem to temp directory
	cmd := exec.Command("rsync", "-aHAXx", 
		"--exclude=/proc/*", "--exclude=/sys/*", "--exclude=/dev/*",
		"--exclude=/tmp/*", "--exclude=/var/tmp/*", "--exclude=/run/*",
		"--exclude=/mnt/*", "--exclude=/media/*", "--exclude=/lost+found",
		"--exclude=/app/disk_images/*",
		"/", tempDir+"/")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy filesystem: %v", err)
	}

	// Create snapshot info file in the ISO
	infoDir := filepath.Join(tempDir, "snapshot_info")
	if err := os.MkdirAll(infoDir, 0755); err != nil {
		return err
	}
	
	infoFile := filepath.Join(infoDir, "disk_image_info.txt")
	if file, err := os.Create(infoFile); err == nil {
		fmt.Fprintf(file, "Last Snapshot Information\n")
		fmt.Fprintf(file, "========================\n")
		fmt.Fprintf(file, "Disk image created: %s\n", time.Now().Format(time.RFC3339))
		fmt.Fprintf(file, "Source: Container OS filesystem\n")
		fmt.Fprintf(file, "Type: Compressed ISO (gzip)\n")
		fmt.Fprintf(file, "Encryption: AES-256-GCM with Shamir Secret Sharing\n")
		fmt.Fprintf(file, "\nTo restore:\n")
		fmt.Fprintf(file, "1. Decrypt with 3 key shares\n")
		fmt.Fprintf(file, "2. Decompress with gunzip\n")
		fmt.Fprintf(file, "3. Mount ISO or use in VM\n")
		file.Close()
	}

	// Create bootloader directory structure
	bootDir := filepath.Join(tempDir, "isolinux")
	if err := os.MkdirAll(bootDir, 0755); err != nil {
		return err
	}

	// Copy isolinux files for booting
	cmd = exec.Command("cp", "/usr/lib/ISOLINUX/isolinux.bin", bootDir)
	cmd.Run()
	cmd = exec.Command("cp", "/usr/lib/syslinux/modules/bios/ldlinux.c32", bootDir)
	cmd.Run()

	// Create isolinux.cfg for boot menu
	cfgContent := `DEFAULT linux
LABEL linux
  KERNEL /boot/vmlinuz
  APPEND root=/dev/sr0 ro
`
	os.WriteFile(filepath.Join(bootDir, "isolinux.cfg"), []byte(cfgContent), 0644)

	// Create ISO with better compatibility
	cmd = exec.Command("genisoimage", "-o", tempISO, "-R", "-J", "-joliet-long", tempDir)
	if err := cmd.Run(); err != nil {
		// Fallback to simple format
		cmd = exec.Command("genisoimage", "-o", tempISO, "-R", tempDir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create ISO: %v", err)
		}
	}

	// Compress ISO
	cmd = exec.Command("gzip", "-c", tempISO)
	outFile, err := os.Create(isoPath)
	if err != nil {
		return err
	}
	defer outFile.Close()
	
	cmd.Stdout = outFile
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to compress ISO: %v", err)
	}

	os.Remove(tempISO)
	logInfo("Compressed ISO created successfully")
	return nil
}



func createMetadata(metadataPath string) error {
	file, err := os.Create(metadataPath)
	if err != nil {
		return err
	}
	defer file.Close()

	hostname, _ := os.Hostname()

	fmt.Fprintf(file, "Disk image created on: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(file, "Hostname: %s\n", hostname)

	if uptimeBytes, err := os.ReadFile("/proc/uptime"); err == nil {
		fmt.Fprintf(file, "Uptime: %s", string(uptimeBytes))
	}

	return nil
}

func checkRetentionPolicy() {
	retentionDays := getRetentionDays()
	if retentionDays <= 0 {
		return
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	logInfo("🗑️ Checking retention policy: removing disk images older than %d days", retentionDays)

	removed := 0
	totalSize := int64(0)

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

func encryptDiskImage(diskPath, encryptedPath string, key []byte) error {
	logInfo("Encrypting disk image...")

	if err := encryptFile(diskPath, encryptedPath, key); err != nil {
		return fmt.Errorf("failed to encrypt disk image: %v", err)
	}

	logInfo("Disk image encrypted successfully")
	return nil
}

func updateSnapshotInfoFile(diskImageName, encryptedDiskPath string) error {
	infoFilePath := "/app/last_snapshot_info.txt"
	
	file, err := os.Create(infoFilePath)
	if err != nil {
		return fmt.Errorf("failed to create snapshot info file: %v", err)
	}
	defer file.Close()

	now := time.Now()
	hostname, _ := os.Hostname()

	// Get file size
	info, err := os.Stat(encryptedDiskPath)
	var fileSize int64 = 0
	if err == nil {
		fileSize = info.Size()
	}

	fmt.Fprintf(file, "Last Snapshot Information\n")
	fmt.Fprintf(file, "========================\n\n")
	fmt.Fprintf(file, "Snapshot Name: %s\n", diskImageName)
	fmt.Fprintf(file, "Creation Date: %s\n", now.Format("2006-01-02"))
	fmt.Fprintf(file, "Creation Time: %s\n", now.Format("15:04:05"))
	fmt.Fprintf(file, "Full Timestamp: %s\n", now.Format(time.RFC3339))
	fmt.Fprintf(file, "Hostname: %s\n", hostname)
	fmt.Fprintf(file, "File Path: %s\n", encryptedDiskPath)
	fmt.Fprintf(file, "File Size: %.2f MB\n", float64(fileSize)/1024/1024)
	fmt.Fprintf(file, "Encryption: AES-256-GCM\n")
	fmt.Fprintf(file, "\nSnapshot Type: Full OS Disk Image\n")
	fmt.Fprintf(file, "Next Snapshot: %s (estimated)\n", now.Add(time.Minute).Format("15:04:05"))

	logInfo("Updated snapshot info file: %s", infoFilePath)
	return nil
}
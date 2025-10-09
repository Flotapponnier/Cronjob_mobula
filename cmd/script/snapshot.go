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

var (
	// System paths
	tempMountPoint    string
	tempBootMount     string
	tempISODir        string
	tempISOFile       string
	infoFileName      string
	snapshotInfoDir   string
	diskImageInfoFile string
	
	// System tools
	mkfsExt4Path     string
	genisoimagePath  string
	isolinuxLibPath  string
	syslinuxLibPath  string
	
	// Exclusions
	excludePatterns  []string
)

func loadConfig() {
	diskImageDir = "/app/disk_images"
	keyDir := "/app/keys"
	keyFilename := "master.key"
	
	// Default system paths
	tempMountPoint = "/tmp/disk_mount"
	tempBootMount = "/tmp/boot_mount"
	tempISODir = "/tmp/iso_content"
	tempISOFile = "/tmp/temp.iso"
	infoFileName = "last_snapshot_info.txt"
	snapshotInfoDir = "snapshot_info"
	diskImageInfoFile = "disk_image_info.txt"
	
	// Default system tools
	mkfsExt4Path = "/sbin/mkfs.ext4"
	genisoimagePath = "genisoimage"
	isolinuxLibPath = "/usr/lib/ISOLINUX"
	syslinuxLibPath = "/usr/lib/syslinux/modules/bios"
	
	// Default exclusions
	excludePatterns = []string{
		"--exclude=/proc/*", "--exclude=/sys/*", "--exclude=/dev/*",
		"--exclude=/tmp/*", "--exclude=/var/tmp/*", "--exclude=/run/*",
		"--exclude=/mnt/*", "--exclude=/media/*", "--exclude=/lost+found",
		"--exclude=/app/disk_images/*",
	}

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
			if value != "" { diskImageDir = value }
		case "KEY_DIR":
			if value != "" { keyDir = value }
		case "KEY_FILENAME":
			if value != "" { keyFilename = value }
		// System paths
		case "TEMP_MOUNT_POINT":
			if value != "" { tempMountPoint = value }
		case "TEMP_BOOT_MOUNT":
			if value != "" { tempBootMount = value }
		case "TEMP_ISO_DIR":
			if value != "" { tempISODir = value }
		case "TEMP_ISO_FILE":
			if value != "" { tempISOFile = value }
		case "INFO_FILE_NAME":
			if value != "" { infoFileName = value }
		case "SNAPSHOT_INFO_DIR":
			if value != "" { snapshotInfoDir = value }
		case "DISK_IMAGE_INFO_FILE":
			if value != "" { diskImageInfoFile = value }
		// System tools
		case "MKFS_EXT4_PATH":
			if value != "" { mkfsExt4Path = value }
		case "GENISOIMAGE_PATH":
			if value != "" { genisoimagePath = value }
		case "ISOLINUX_LIB_PATH":
			if value != "" { isolinuxLibPath = value }
		case "SYSLINUX_LIB_PATH":
			if value != "" { syslinuxLibPath = value }
		// Exclusions (rebuild the array if any exclusion is set)
		case "EXCLUDE_PROC", "EXCLUDE_SYS", "EXCLUDE_DEV", "EXCLUDE_TMP", 
			 "EXCLUDE_VAR_TMP", "EXCLUDE_RUN", "EXCLUDE_MNT", "EXCLUDE_MEDIA", "EXCLUDE_LOST_FOUND":
			if value != "" {
				updateExclusionPattern(key, value)
			}
		}
	}

	keyFile = filepath.Join(keyDir, keyFilename)
}

func updateExclusionPattern(key, value string) {
	// Find and update the specific exclusion pattern
	exclusionMap := map[string]int{
		"EXCLUDE_PROC": 0, "EXCLUDE_SYS": 1, "EXCLUDE_DEV": 2,
		"EXCLUDE_TMP": 3, "EXCLUDE_VAR_TMP": 4, "EXCLUDE_RUN": 5,
		"EXCLUDE_MNT": 6, "EXCLUDE_MEDIA": 7, "EXCLUDE_LOST_FOUND": 8,
	}
	
	if idx, exists := exclusionMap[key]; exists && idx < len(excludePatterns) {
		excludePatterns[idx] = "--exclude=" + value
	}
}

func createCompressedISO(isoPath string) error {
	logInfo("Creating compressed ISO from filesystem...")

	// Create temp directory
	if err := os.MkdirAll(tempISODir, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(tempISODir)

	// Copy filesystem to temp directory using configured exclusions
	args := append([]string{"rsync", "-aHAXx"}, excludePatterns...)
	args = append(args, "/", tempISODir+"/")
	cmd := exec.Command(args[0], args[1:]...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy filesystem: %v", err)
	}

	// Create snapshot info file in the ISO
	infoDir := filepath.Join(tempISODir, snapshotInfoDir)
	if err := os.MkdirAll(infoDir, 0755); err != nil {
		return err
	}
	
	infoFile := filepath.Join(infoDir, diskImageInfoFile)
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
	bootDir := filepath.Join(tempISODir, "isolinux")
	if err := os.MkdirAll(bootDir, 0755); err != nil {
		return err
	}

	// Copy isolinux files for booting using configured paths
	cmd = exec.Command("cp", filepath.Join(isolinuxLibPath, "isolinux.bin"), bootDir)
	cmd.Run()
	cmd = exec.Command("cp", filepath.Join(syslinuxLibPath, "ldlinux.c32"), bootDir)
	cmd.Run()

	// Create isolinux.cfg for boot menu
	cfgContent := `DEFAULT linux
LABEL linux
  KERNEL /boot/vmlinuz
  APPEND root=/dev/sr0 ro
`
	os.WriteFile(filepath.Join(bootDir, "isolinux.cfg"), []byte(cfgContent), 0644)

	// Create ISO with better compatibility using configured tool
	cmd = exec.Command(genisoimagePath, "-o", tempISOFile, "-R", "-J", "-joliet-long", tempISODir)
	if err := cmd.Run(); err != nil {
		// Fallback to simple format
		cmd = exec.Command(genisoimagePath, "-o", tempISOFile, "-R", tempISODir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create ISO: %v", err)
		}
	}

	// Compress ISO
	cmd = exec.Command("gzip", "-c", tempISOFile)
	outFile, err := os.Create(isoPath)
	if err != nil {
		return err
	}
	defer outFile.Close()
	
	cmd.Stdout = outFile
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to compress ISO: %v", err)
	}

	os.Remove(tempISOFile)
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
	infoFilePath := "/app/" + infoFileName
	
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
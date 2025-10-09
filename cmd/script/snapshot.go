package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Global configuration variables
var (
	diskImageDir string
	keyFile      string

	// System paths
	tempMountPoint    string
	tempBootMount     string
	tempISODir        string
	tempISOFile       string
	infoFileName      string
	snapshotInfoDir   string
	diskImageInfoFile string

	// System tools
	mkfsExt4Path    string
	genisoimagePath string
	isolinuxLibPath string
	syslinuxLibPath string

	// Exclusions
	excludePatterns []string
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

	// Use single timestamp for consistency
	now := time.Now()
	diskImagePath, diskImageName, err := createArchitecturedDiskImageWithTime(now)
	if err != nil {
		logError("Failed to create disk image path: %v", err)
		return
	}

	logInfo("Starting encrypted OS disk image %s", diskImageName)

	isoPath := diskImagePath + ".iso.gz"
	if err := createCompressedISOWithTime(isoPath, now); err != nil {
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
		// System paths
		case "TEMP_MOUNT_POINT":
			if value != "" {
				tempMountPoint = value
			}
		case "TEMP_BOOT_MOUNT":
			if value != "" {
				tempBootMount = value
			}
		case "TEMP_ISO_DIR":
			if value != "" {
				tempISODir = value
			}
		case "TEMP_ISO_FILE":
			if value != "" {
				tempISOFile = value
			}
		case "INFO_FILE_NAME":
			if value != "" {
				infoFileName = value
			}
		case "SNAPSHOT_INFO_DIR":
			if value != "" {
				snapshotInfoDir = value
			}
		case "DISK_IMAGE_INFO_FILE":
			if value != "" {
				diskImageInfoFile = value
			}
		// System tools
		case "MKFS_EXT4_PATH":
			if value != "" {
				mkfsExt4Path = value
			}
		case "GENISOIMAGE_PATH":
			if value != "" {
				genisoimagePath = value
			}
		case "ISOLINUX_LIB_PATH":
			if value != "" {
				isolinuxLibPath = value
			}
		case "SYSLINUX_LIB_PATH":
			if value != "" {
				syslinuxLibPath = value
			}
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
	return createCompressedISOWithTime(isoPath, time.Now())
}

func createCompressedISOWithTime(isoPath string, now time.Time) error {
	logInfo("Creating compressed ISO from filesystem...")

	if err := os.MkdirAll(tempISODir, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(tempISODir)

	args := append([]string{"rsync", "-aHAXx"}, excludePatterns...)
	args = append(args, "/", tempISODir+"/")
	cmd := exec.Command(args[0], args[1:]...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy filesystem: %v", err)
	}

	infoDir := filepath.Join(tempISODir, snapshotInfoDir)
	if err := os.MkdirAll(infoDir, 0755); err != nil {
		return err
	}

	infoFile := filepath.Join(infoDir, diskImageInfoFile)
	if file, err := os.Create(infoFile); err == nil {
		fmt.Fprintf(file, "Last Snapshot Information\n")
		fmt.Fprintf(file, "========================\n")
		fmt.Fprintf(file, "Disk image created: %s\n", now.Format(time.RFC3339))
		fmt.Fprintf(file, "Source: Container OS filesystem\n")
		fmt.Fprintf(file, "Type: Compressed ISO (gzip)\n")
		fmt.Fprintf(file, "Encryption: AES-256-GCM with Shamir Secret Sharing\n")
		fmt.Fprintf(file, "\nTo restore:\n")
		fmt.Fprintf(file, "1. Decrypt with 3 key shares\n")
		fmt.Fprintf(file, "2. Decompress with gunzip\n")
		fmt.Fprintf(file, "3. Mount ISO or use in VM\n")
		file.Close()
	}

	bootDir := filepath.Join(tempISODir, "isolinux")
	if err := os.MkdirAll(bootDir, 0755); err != nil {
		return err
	}

	cmd = exec.Command("cp", filepath.Join(isolinuxLibPath, "isolinux.bin"), bootDir)
	cmd.Run()
	cmd = exec.Command("cp", filepath.Join(syslinuxLibPath, "ldlinux.c32"), bootDir)
	cmd.Run()

	cfgContent := `DEFAULT linux
LABEL linux
  KERNEL /boot/vmlinuz
  APPEND root=/dev/sr0 ro
`
	os.WriteFile(filepath.Join(bootDir, "isolinux.cfg"), []byte(cfgContent), 0644)

	cmd = exec.Command(genisoimagePath, "-o", tempISOFile, "-R", "-J", "-joliet-long", tempISODir)
	if err := cmd.Run(); err != nil {
		cmd = exec.Command(genisoimagePath, "-o", tempISOFile, "-R", tempISODir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create ISO: %v", err)
		}
	}

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

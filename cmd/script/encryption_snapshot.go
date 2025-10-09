package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	keyLengthBytes = 32 // AES-256 key length
)

func loadMasterKey() ([]byte, error) {
	keyHex, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read master key: %v", err)
	}

	keyStr := strings.TrimSpace(string(keyHex))
	key, err := hex.DecodeString(keyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode master key: %v", err)
	}

	if len(key) != keyLengthBytes {
		return nil, fmt.Errorf("invalid key length: expected %d bytes, got %d", keyLengthBytes, len(key))
	}

	return key, nil
}

func encryptFile(srcFile, dstFile string, key []byte) error {
	plaintext, err := os.ReadFile(srcFile)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return os.WriteFile(dstFile, ciphertext, 0600)
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

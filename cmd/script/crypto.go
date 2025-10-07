package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// loadMasterKey loads the master encryption key from file
func loadMasterKey() ([]byte, error) {
	keyHex, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read master key: %v", err)
	}

	// Convert hex string to bytes
	keyStr := strings.TrimSpace(string(keyHex))
	key, err := hex.DecodeString(keyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode master key: %v", err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key length: expected 32 bytes, got %d", len(key))
	}

	return key, nil
}

// encryptSnapshot compresses and encrypts a snapshot directory
func encryptSnapshot(snapshotPath, encryptedPath string, key []byte) error {
	// Create compressed tar archive
	tarPath := snapshotPath + ".tar.gz"
	if err := createTarGz(snapshotPath, tarPath); err != nil {
		return fmt.Errorf("failed to create tar archive: %v", err)
	}
	defer os.Remove(tarPath) // Clean up tar file

	// Encrypt the tar file
	if err := encryptFile(tarPath, encryptedPath, key); err != nil {
		return fmt.Errorf("failed to encrypt file: %v", err)
	}

	return nil
}

// createTarGz creates a compressed tar archive
func createTarGz(srcDir, dstFile string) error {
	file, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// Update header name to be relative to srcDir
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			return err
		}

		return nil
	})
}

// encryptFile encrypts a file using AES-GCM
func encryptFile(srcFile, dstFile string, key []byte) error {
	// Read source file
	plaintext, err := os.ReadFile(srcFile)
	if err != nil {
		return err
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// Write encrypted file
	return os.WriteFile(dstFile, ciphertext, 0600)
}
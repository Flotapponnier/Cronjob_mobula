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

	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key length: expected 32 bytes, got %d", len(key))
	}

	return key, nil
}

func encryptSnapshot(snapshotPath, encryptedPath string, key []byte) error {
	tarPath := snapshotPath + ".tar.gz"
	if err := createTarGz(snapshotPath, tarPath); err != nil {
		return fmt.Errorf("failed to create tar archive: %v", err)
	}
	defer os.Remove(tarPath)

	if err := encryptFile(tarPath, encryptedPath, key); err != nil {
		return fmt.Errorf("failed to encrypt file: %v", err)
	}

	return nil
}

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


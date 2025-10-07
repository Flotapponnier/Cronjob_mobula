package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/vault/shamir"
)

func main() {
	fmt.Println("🔓 Simple Decryption Test")
	fmt.Println("=========================")
	
	if len(os.Args) < 2 {
		// Interactive mode - ask for key shares
		runInteractiveTest()
	} else if os.Args[1] == "create-test" {
		// Create test encrypted file
		createTestFile()
	} else {
		fmt.Println("Usage:")
		fmt.Println("  simple_decrypt_test                 # Interactive test")
		fmt.Println("  simple_decrypt_test create-test     # Create test file")
	}
}

func runInteractiveTest() {
	fmt.Println("This will decrypt a snapshot using 2 key shares.")
	fmt.Println()
	
	// Get snapshot file path
	fmt.Print("Enter snapshot file path: ")
	var filePath string
	fmt.Scanln(&filePath)
	filePath = strings.TrimSpace(filePath)
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("❌ File %s not found.\n", filePath)
		return
	}
	
	// Get first key share
	fmt.Print("Enter KEY SHARE #1: ")
	var share1 string
	fmt.Scanln(&share1)
	share1 = strings.TrimSpace(share1)
	
	// Get second key share  
	fmt.Print("Enter KEY SHARE #2: ")
	var share2 string
	fmt.Scanln(&share2)
	share2 = strings.TrimSpace(share2)
	
	fmt.Println()
	fmt.Printf("🔐 Attempting to reconstruct master key from shares...\n")
	
	// Convert hex shares to bytes
	shareBytes1, err := hex.DecodeString(share1)
	if err != nil {
		fmt.Printf("❌ Invalid hex in share 1: %v\n", err)
		return
	}
	
	shareBytes2, err := hex.DecodeString(share2)
	if err != nil {
		fmt.Printf("❌ Invalid hex in share 2: %v\n", err)
		return
	}
	
	// Reconstruct master key
	shares := [][]byte{shareBytes1, shareBytes2}
	masterKey, err := shamir.Combine(shares)
	if err != nil {
		fmt.Printf("❌ Failed to reconstruct key: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Master key reconstructed: %s\n", hex.EncodeToString(masterKey))
	
	// Try to decrypt the specified file
	fmt.Printf("🔓 Decrypting snapshot: %s\n", filePath)
	
	decryptedData, err := decryptFile(filePath, masterKey)
	if err != nil {
		fmt.Printf("❌ Decryption failed: %v\n", err)
		return
	}
	
	fmt.Printf("✅ SUCCESS! Decrypted snapshot size: %d bytes\n", len(decryptedData))
	fmt.Println("💾 Snapshot decrypted successfully - it's a tar.gz archive")
	fmt.Println()
	
	// Ask if user wants to decompress and show contents
	fmt.Print("Do you want me to decompress it and show the inside? (y/n): ")
	var response string
	fmt.Scanln(&response)
	response = strings.TrimSpace(strings.ToLower(response))
	
	if response == "y" || response == "yes" {
		if err := decompressAndShow(decryptedData, filePath); err != nil {
			fmt.Printf("❌ Failed to decompress: %v\n", err)
		}
	}
	
	fmt.Println()
	fmt.Println("🎉 Your Shamir Secret Sharing system works perfectly!")
}

func createTestFile() {
	fmt.Println("📝 Creating test encrypted file...")
	
	// Read the master key
	keyHex, err := os.ReadFile("/app/keys/master.key")
	if err != nil {
		fmt.Printf("❌ Cannot read master key: %v\n", err)
		return
	}
	
	keyStr := strings.TrimSpace(string(keyHex))
	masterKey, err := hex.DecodeString(keyStr)
	if err != nil {
		fmt.Printf("❌ Cannot decode master key: %v\n", err)
		return
	}
	
	// Encrypt "hello world!" with the master key
	plaintext := []byte("hello world!")
	
	encryptedData, err := encryptData(plaintext, masterKey)
	if err != nil {
		fmt.Printf("❌ Encryption failed: %v\n", err)
		return
	}
	
	// Save encrypted file
	testFile := "/app/test_hello.encrypted"
	if err := os.WriteFile(testFile, encryptedData, 0600); err != nil {
		fmt.Printf("❌ Failed to save test file: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Test file created: %s\n", testFile)
	fmt.Printf("📝 Contains encrypted: \"hello world!\"\n")
	fmt.Printf("🔑 Encrypted with master key: %s\n", hex.EncodeToString(masterKey))
	fmt.Println()
	fmt.Println("Now run without arguments to test decryption!")
}

// encryptData encrypts data using AES-GCM
func encryptData(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptFile decrypts a file using AES-GCM
func decryptFile(filename string, key []byte) ([]byte, error) {
	ciphertext, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	
	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]
	
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	
	return plaintext, nil
}

// decompressAndShow decompresses the tar.gz data and extracts it to a folder
func decompressAndShow(data []byte, originalPath string) error {
	fmt.Println("📂 Decompressing and extracting snapshot...")
	
	// Create decrypted folder with timestamp
	timestamp := time.Now().Format("20060102_150405")
	baseName := filepath.Base(originalPath)
	baseName = strings.TrimSuffix(baseName, ".encrypted")
	decryptedDir := fmt.Sprintf("/app/decrypted/%s_%s", baseName, timestamp)
	
	if err := os.MkdirAll(decryptedDir, 0755); err != nil {
		return fmt.Errorf("failed to create decrypted directory: %v", err)
	}
	
	// Decompress gzip
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()
	
	// Extract tar archive
	tarReader := tar.NewReader(gzReader)
	fileCount := 0
	totalSize := int64(0)
	
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar reading error: %v", err)
		}
		
		targetPath := filepath.Join(decryptedDir, header.Name)
		
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", targetPath, err)
			}
		case tar.TypeReg:
			fileCount++
			totalSize += header.Size
			
			// Create parent directory if needed
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %v", targetPath, err)
			}
			
			// Create file
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %v", targetPath, err)
			}
			
			// Copy file content
			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return fmt.Errorf("failed to copy file content for %s: %v", targetPath, err)
			}
			file.Close()
		}
	}
	
	fmt.Printf("✅ Extracted to: %s\n", decryptedDir)
	fmt.Printf("📊 Files extracted: %d\n", fileCount)
	fmt.Printf("📏 Total size: %d bytes (%.2f MB)\n", totalSize, float64(totalSize)/1024/1024)
	fmt.Println()
	
	// Show directory tree
	fmt.Println("📁 Directory structure:")
	cmd := exec.Command("tree", decryptedDir, "-L", "3")
	if output, err := cmd.Output(); err == nil {
		fmt.Print(string(output))
	} else {
		// Fallback to ls if tree is not available
		cmd = exec.Command("ls", "-la", decryptedDir)
		if output, err := cmd.Output(); err == nil {
			fmt.Print(string(output))
		}
	}
	
	// Show metadata if exists
	metadataPath := filepath.Join(decryptedDir, "metadata.txt")
	if _, err := os.Stat(metadataPath); err == nil {
		fmt.Println()
		fmt.Println("📄 Snapshot metadata:")
		if content, err := os.ReadFile(metadataPath); err == nil {
			fmt.Print(string(content))
		}
	}
	
	return nil
}
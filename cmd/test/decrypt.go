package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

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
	fmt.Println("This test will ask for 2 key shares and decrypt a test file.")
	fmt.Println()
	
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
	
	// Try to decrypt test file
	testFile := "/app/test_hello.encrypted"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		fmt.Printf("❌ Test file %s not found. Run with 'create-test' first.\n", testFile)
		return
	}
	
	fmt.Printf("🔓 Decrypting test file: %s\n", testFile)
	
	decryptedData, err := decryptFile(testFile, masterKey)
	if err != nil {
		fmt.Printf("❌ Decryption failed: %v\n", err)
		return
	}
	
	fmt.Printf("✅ SUCCESS! Decrypted content: \"%s\"\n", string(decryptedData))
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
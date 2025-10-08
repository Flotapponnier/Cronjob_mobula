package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/vault/shamir"
)

const (
	ColorReset = "\033[0m"
	ColorGreen = "\033[32m"
	ColorRed   = "\033[31m"
)

type KeyInfo struct {
	MasterKeyHex   string    `json:"master_key_hex"`
	GeneratedAt    time.Time `json:"generated_at"`
	TotalShares    int       `json:"total_shares"`
	RequiredShares int       `json:"required_shares"`
}

func main() {
	fmt.Println("🔓 Decryption Tool")
	fmt.Println("==================")

	if len(os.Args) < 2 {
		runSimpleTest()
	} else if os.Args[1] == "create-test" {
		createTestFile()
	} else if os.Args[1] == "snapshot" {
		runInteractiveTest()
	} else {
		fmt.Println("Usage:")
		fmt.Println("  decrypt                    # Simple 'hello world' test")
		fmt.Println("  decrypt snapshot           # Decrypt snapshot files")
		fmt.Println("  decrypt create-test        # Create test file")
	}
}

func runSimpleTest() {
	fmt.Println("🧪 Simple 'hello world!' decryption test")

	keyInfo, err := loadKeyInfo()
	if err != nil {
		fmt.Printf("%s❌ Failed to load key info: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	fmt.Printf("This test needs %d key shares to decrypt 'hello world!'\n", keyInfo.RequiredShares)
	fmt.Println()

	testFile := "/app/test_hello.encrypted"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		fmt.Printf("%s❌ Test file not found. Creating it first...%s\n", ColorRed, ColorReset)
		createTestFile()
		fmt.Println()
	}

	shares := getKeyShares(keyInfo.RequiredShares)
	masterKey, err := reconstructMasterKey(shares)
	if err != nil {
		return
	}

	fmt.Printf("%s✅ Master key reconstructed!%s\n", ColorGreen, ColorReset)

	fmt.Printf("🔓 Decrypting test message...\n")
	decryptedData, err := decryptFile(testFile, masterKey)
	if err != nil {
		fmt.Printf("%s❌ Decryption failed: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	fmt.Printf("%s✅ SUCCESS! Decrypted message: \"%s\"%s\n", ColorGreen, string(decryptedData), ColorReset)
	fmt.Println()
	fmt.Printf("%s🎉 Your Shamir Secret Sharing system works perfectly!%s\n", ColorGreen, ColorReset)
}

func runInteractiveTest() {
	fmt.Println("🔓 Snapshot decryption mode")

	keyInfo, err := loadKeyInfo()
	if err != nil {
		fmt.Printf("%s❌ Failed to load key info: %v%s\n", ColorRed, err, ColorReset)
		fmt.Println("Using default: 3 shares required")
		keyInfo.RequiredShares = 3
	}

	fmt.Printf("This will decrypt a snapshot using %d key shares.\n", keyInfo.RequiredShares)
	fmt.Println()

	fmt.Print("Enter snapshot file path: ")
	var filePath string
	fmt.Scanln(&filePath)
	filePath = strings.TrimSpace(filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("%s❌ File %s not found.%s\n", ColorRed, filePath, ColorReset)
		return
	}

	shares := getKeyShares(keyInfo.RequiredShares)
	masterKey, err := reconstructMasterKey(shares)
	if err != nil {
		return
	}

	fmt.Printf("%s✅ Master key reconstructed: %s%s\n", ColorGreen, hex.EncodeToString(masterKey), ColorReset)

	fmt.Printf("🔓 Decrypting snapshot: %s\n", filePath)

	decryptedData, err := decryptFile(filePath, masterKey)
	if err != nil {
		fmt.Printf("%s❌ Decryption failed: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	fmt.Printf("%s✅ SUCCESS! Decrypted snapshot size: %d bytes%s\n", ColorGreen, len(decryptedData), ColorReset)
	fmt.Printf("%s💾 Snapshot decrypted successfully!%s\n", ColorGreen, ColorReset)
	fmt.Println()
	fmt.Printf("%s🎉 Decryption completed successfully!%s\n", ColorGreen, ColorReset)
}

func createTestFile() {
	fmt.Println("📝 Creating test encrypted file...")

	keyHex, err := os.ReadFile("/app/keys/master.key")
	if err != nil {
		fmt.Printf("%s❌ Cannot read master key: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	keyStr := strings.TrimSpace(string(keyHex))
	masterKey, err := hex.DecodeString(keyStr)
	if err != nil {
		fmt.Printf("%s❌ Cannot decode master key: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	plaintext := []byte("hello world!")

	encryptedData, err := encryptData(plaintext, masterKey)
	if err != nil {
		fmt.Printf("%s❌ Encryption failed: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	testFile := "/app/test_hello.encrypted"
	if err := os.WriteFile(testFile, encryptedData, 0600); err != nil {
		fmt.Printf("%s❌ Failed to save test file: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	fmt.Printf("%s✅ Test file created: %s%s\n", ColorGreen, testFile, ColorReset)
	fmt.Printf("📝 Contains encrypted: \"hello world!\"\n")
	fmt.Printf("🔑 Encrypted with master key: %s\n", hex.EncodeToString(masterKey))
	fmt.Println()
	fmt.Println("Now run without arguments to test decryption!")
}

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

func getKeyShares(count int) []string {
	shares := make([]string, count)
	for i := 0; i < count; i++ {
		fmt.Printf("Enter KEY SHARE #%d: ", i+1)
		var share string
		fmt.Scanln(&share)
		shares[i] = strings.TrimSpace(share)
	}
	return shares
}

func reconstructMasterKey(shares []string) ([]byte, error) {
	fmt.Println()
	fmt.Printf("🔐 Reconstructing master key from %d shares...\n", len(shares))

	shareBytes := make([][]byte, len(shares))
	for i, share := range shares {
		bytes, err := hex.DecodeString(share)
		if err != nil {
			fmt.Printf("%s❌ Invalid hex in share %d: %v%s\n", ColorRed, i+1, err, ColorReset)
			return nil, err
		}
		shareBytes[i] = bytes
	}

	masterKey, err := shamir.Combine(shareBytes)
	if err != nil {
		fmt.Printf("%s❌ Failed to reconstruct key: %v%s\n", ColorRed, err, ColorReset)
		return nil, err
	}

	return masterKey, nil
}

func loadKeyInfo() (KeyInfo, error) {
	var keyInfo KeyInfo

	infoFile := "/app/keys/key_info.json"
	data, err := os.ReadFile(infoFile)
	if err != nil {
		return keyInfo, fmt.Errorf("failed to read key info file: %v", err)
	}

	if err := json.Unmarshal(data, &keyInfo); err != nil {
		return keyInfo, fmt.Errorf("failed to parse key info: %v", err)
	}

	return keyInfo, nil
}


package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/vault/shamir"
)

// ANSI color codes for terminal output
const (
	ColorReset  = "\033[0m"
	ColorBlue   = "\033[34m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorRed    = "\033[31m"
)

// Constants for key generation
const (
	keyLengthBytes        = 32  // AES-256 key length
	defaultTotalShares    = 3   // Default number of Shamir shares
	defaultThreshold      = 3   // Default minimum shares needed
	defaultKeyFilename    = "master.key"
	defaultInfoFilename   = "key_info.json"
	defaultTestFilename   = "test_hello.encrypted"
)

// KeyInfo stores metadata about generated keys
type KeyInfo struct {
	MasterKeyHex   string    `json:"master_key_hex"`
	GeneratedAt    time.Time `json:"generated_at"`
	TotalShares    int       `json:"total_shares"`
	RequiredShares int       `json:"required_shares"`
}

var (
	keyFile  string
	keyDir   string
	testFile string
)

func main() {
	fmt.Println("🔐 Encryption Key Generator")
	fmt.Println("===========================")

	loadConfig()
	totalShares, threshold, err := validateShamirConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📋 Configuration: %d total shares, %d required to decrypt\n", totalShares, threshold)

	if err := os.MkdirAll(keyDir, 0700); err != nil {
		fmt.Printf("❌ Failed to create key directory: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(keyFile); err == nil {
		fmt.Printf("⚠️  Master key already exists at %s\n", keyFile)
		fmt.Print("Do you want to regenerate it? This will invalidate existing encrypted snapshots! (y/N): ")

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("✅ Keeping existing key")
			return
		}
		fmt.Println("🔄 Regenerating master key...")
	}

	key, err := generateMasterKey()
	if err != nil {
		fmt.Printf("❌ Failed to generate master key: %v\n", err)
		os.Exit(1)
	}

	cleanupOldTestFile()

	shares, err := createKeyShares(hex.EncodeToString(key), totalShares, threshold)
	if err != nil {
		fmt.Printf("❌ Failed to create key shares: %v\n", err)
		os.Exit(1)
	}

	displayKeyShares(shares, threshold)

	keyInfo := KeyInfo{
		MasterKeyHex:   hex.EncodeToString(key),
		GeneratedAt:    time.Now(),
		TotalShares:    totalShares,
		RequiredShares: threshold,
	}

	if err := saveKeyInfo(keyInfo); err != nil {
		fmt.Printf("❌ Failed to save key info: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s✅ Encryption setup completed successfully!%s\n", ColorGreen, ColorReset)
	fmt.Printf("🔑 Master key saved to: %s\n", keyFile)
	fmt.Println("📝 Your snapshot program can now encrypt data using the master key")
}

func generateMasterKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %v", err)
	}

	keyHex := hex.EncodeToString(key)
	if err := os.WriteFile(keyFile, []byte(keyHex), 0600); err != nil {
		return nil, fmt.Errorf("failed to save key to file: %v", err)
	}

	fmt.Printf("🔑 Generated new 256-bit master key\n")
	return key, nil
}

func createKeyShares(keyHex string, totalShares, requiredShares int) ([]string, error) {
	fmt.Printf("🔐 Creating %d key shares (threshold: %d)\n", totalShares, requiredShares)

	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key hex: %v", err)
	}

	shares, err := shamir.Split(keyBytes, totalShares, requiredShares)
	if err != nil {
		return nil, fmt.Errorf("failed to create Shamir shares: %v", err)
	}

	shareStrings := make([]string, len(shares))
	for i, share := range shares {
		shareStrings[i] = hex.EncodeToString(share)
	}

	fmt.Printf("%s✅ Created %d key shares successfully%s\n", ColorGreen, len(shareStrings), ColorReset)
	return shareStrings, nil
}

func loadConfig() {
	envVars := readEnvFile()

	keyDir = getConfigValue(envVars, "KEY_DIR", "/app/keys")
	keyFilename := getConfigValue(envVars, "KEY_FILENAME", "master.key")
	testFile = getConfigValue(envVars, "TEST_FILE", "/app/test_hello.encrypted")

	keyFile = filepath.Join(keyDir, keyFilename)
}

func validateShamirConfig() (int, int, error) {
	envVars := readEnvFile()

	totalShares := getConfigInt(envVars, "SHAMIR_TOTAL_SHARES", 3)
	threshold := getConfigInt(envVars, "SHAMIR_THRESHOLD", 3)

	if threshold > totalShares {
		return 0, 0, fmt.Errorf("threshold (%d) cannot be greater than total shares (%d)", threshold, totalShares)
	}
	if threshold < 2 {
		return 0, 0, fmt.Errorf("threshold must be at least 2, got %d", threshold)
	}
	if totalShares < 2 {
		return 0, 0, fmt.Errorf("total shares must be at least 2, got %d", totalShares)
	}

	return totalShares, threshold, nil
}

func readEnvFile() map[string]string {
	envVars := make(map[string]string)

	envFile := "/app/.env"
	file, err := os.Open(envFile)
	if err != nil {
		return envVars
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
		envVars[key] = value
	}

	return envVars
}

func getConfigValue(envVars map[string]string, key, defaultValue string) string {
	if value, exists := envVars[key]; exists && value != "" {
		return value
	}
	return defaultValue
}

func getConfigInt(envVars map[string]string, key string, defaultValue int) int {
	if value, exists := envVars[key]; exists && value != "" {
		if intVal, err := strconv.Atoi(value); err == nil && intVal > 0 {
			return intVal
		}
	}
	return defaultValue
}

func displayKeyShares(shares []string, threshold int) {
	fmt.Println("🔐 ===== ENCRYPTION KEY SHARES =====")
	fmt.Printf("Generated %d key shares (%d required to decrypt)\n", len(shares), threshold)
	fmt.Println("⚠️  IMPORTANT: Store these shares securely and separately!")
	fmt.Println()

	for i, share := range shares {
		shareNumber := i + 1
		fmt.Printf("🔑 KEY SHARE #%d:\n", shareNumber)

		fmt.Println("   /" + strings.Repeat("-", len(share)+2) + "\\")
		fmt.Printf("   | %s%s%s |\n", ColorBlue, share, ColorReset)
		fmt.Println("   \\" + strings.Repeat("-", len(share)+2) + "/")
		fmt.Println()
	}

	fmt.Println("📋 DECRYPTION INSTRUCTIONS:")
	fmt.Printf("   • Any %d of these %d shares can reconstruct the master key\n", threshold, len(shares))
	fmt.Println("   • Each share should be stored by a different person/system")
	fmt.Println("   • Never store all shares in the same location")
	fmt.Println("   • These shares can decrypt ALL future snapshots")
	fmt.Println("🔐 ===================================")
}

func cleanupOldTestFile() {
	if _, err := os.Stat(testFile); err == nil {
		if err := os.Remove(testFile); err == nil {
			fmt.Printf("🗑️ Removed old test file (was encrypted with previous key)\n")
		}
	}
}

func saveKeyInfo(keyInfo KeyInfo) error {
	infoFile := keyDir + "/key_info.json"

	jsonData, err := json.MarshalIndent(keyInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal key info: %v", err)
	}

	if err := os.WriteFile(infoFile, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to save key info: %v", err)
	}

	fmt.Printf("💾 Key information saved to: %s\n", infoFile)
	return nil
}

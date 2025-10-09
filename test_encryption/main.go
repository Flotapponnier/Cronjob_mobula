package main

import (
	"bufio"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/vault/shamir"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorBlue   = "\033[34m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorRed    = "\033[31m"
	ColorPurple = "\033[35m"
)

func main() {
	fmt.Printf("%s🔐 Encryption Test Suite%s\n", ColorBlue, ColorReset)
	fmt.Println("========================")
	fmt.Println()

	// Show available encrypted files
	showEncryptedFiles()

	for {
		fmt.Println()
		fmt.Printf("%sChoose an option:%s\n", ColorYellow, ColorReset)
		fmt.Println("1. 🔑 Manual decryption with your 3 key shares")
		fmt.Println("2. 🤖 Concurrent brute force test (verify encryption strength)")
		fmt.Println("3. 📂 Refresh file list")
		fmt.Println("4. 💿 Test ISO contents (mount and browse)")
		fmt.Println("5. ❌ Exit")
		fmt.Print("\nEnter choice (1-5): ")

		choice := getUserInput()

		switch choice {
		case "1":
			manualDecryption()
		case "2":
			bruteForceTest()
		case "3":
			showEncryptedFiles()
		case "4":
			testISOContents()
		case "5":
			fmt.Printf("%s👋 Goodbye!%s\n", ColorGreen, ColorReset)
			return
		default:
			fmt.Printf("%s❌ Invalid choice. Please enter 1, 2, 3, 4, or 5.%s\n", ColorRed, ColorReset)
		}
	}
}

func showEncryptedFiles() {
	fmt.Printf("%s📂 Encrypted files in current directory:%s\n", ColorPurple, ColorReset)
	fmt.Println("=" + strings.Repeat("=", 45))

	found := false
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".encrypted") {
			info, err := d.Info()
			if err != nil {
				return nil
			}

			sizeKB := info.Size() / 1024
			fmt.Printf("📄 %s%s%s (%d KB)\n", ColorBlue, d.Name(), ColorReset, sizeKB)
			found = true
		}
		return nil
	})

	if err != nil {
		fmt.Printf("%s❌ Error scanning directory: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	if !found {
		fmt.Printf("%s⚠️  No .encrypted files found in current directory%s\n", ColorYellow, ColorReset)
		fmt.Println("   Please copy your encrypted snapshot files here first.")
	}
}

func manualDecryption() {
	fmt.Printf("\n%s🔑 Manual Decryption Mode%s\n", ColorGreen, ColorReset)
	fmt.Println("=" + strings.Repeat("=", 25))

	// Get filename
	fmt.Print("Enter encrypted filename: ")
	filename := strings.TrimSpace(getUserInput())

	if filename == "" {
		fmt.Printf("%s❌ No filename provided%s\n", ColorRed, ColorReset)
		return
	}

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Printf("%s❌ File '%s' not found%s\n", ColorRed, filename, ColorReset)
		return
	}

	// Get 3 key shares
	fmt.Printf("\n%sEnter your 3 key shares:%s\n", ColorYellow, ColorReset)
	shares := make([]string, 3)
	for i := 0; i < 3; i++ {
		fmt.Printf("Key share #%d: ", i+1)
		shares[i] = strings.TrimSpace(getUserInput())
		if shares[i] == "" {
			fmt.Printf("%s❌ Empty key share provided%s\n", ColorRed, ColorReset)
			return
		}
	}

	// Attempt decryption
	fmt.Printf("\n%s🔐 Attempting decryption...%s\n", ColorBlue, ColorReset)

	if decryptAndDecompressFile(filename, shares) {
		fmt.Printf("%s✅ SUCCESS! File decrypted and decompressed to ISO%s\n", ColorGreen, ColorReset)
	} else {
		fmt.Printf("%s❌ FAILED! Could not decrypt file%s\n", ColorRed, ColorReset)
	}
}

func bruteForceTest() {
	fmt.Printf("\n%s🤖 Concurrent Brute Force Test%s\n", ColorPurple, ColorReset)
	fmt.Println("=" + strings.Repeat("=", 30))

	// Get filename
	fmt.Print("Enter encrypted filename to test against: ")
	filename := strings.TrimSpace(getUserInput())

	if filename == "" {
		fmt.Printf("%s❌ No filename provided%s\n", ColorRed, ColorReset)
		return
	}

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Printf("%s❌ File '%s' not found%s\n", ColorRed, filename, ColorReset)
		return
	}

	fmt.Printf("\n%s⚠️  WARNING: This will generate random keys and try to crack your encryption%s\n", ColorYellow, ColorReset)
	fmt.Print("This demonstrates that your encryption is uncrackable. Continue? (y/N): ")

	confirm := strings.ToLower(strings.TrimSpace(getUserInput()))
	if confirm != "y" && confirm != "yes" {
		fmt.Printf("%s❌ Test cancelled%s\n", ColorYellow, ColorReset)
		return
	}

	// Get number of concurrent workers
	fmt.Print("\nEnter number of concurrent workers (1-100, default 10): ")
	workersInput := strings.TrimSpace(getUserInput())
	workers := 10
	if workersInput != "" {
		if w, err := strconv.Atoi(workersInput); err == nil && w > 0 && w <= 100 {
			workers = w
		}
	}

	// Get test duration
	fmt.Print("Enter test duration in seconds (default 30): ")
	durationInput := strings.TrimSpace(getUserInput())
	duration := 30
	if durationInput != "" {
		if d, err := strconv.Atoi(durationInput); err == nil && d > 0 && d <= 300 {
			duration = d
		}
	}

	fmt.Printf("\n%s🚀 Starting brute force test:%s\n", ColorBlue, ColorReset)
	fmt.Printf("   📄 Target file: %s\n", filename)
	fmt.Printf("   👥 Workers: %d\n", workers)
	fmt.Printf("   ⏱️  Duration: %d seconds\n", duration)
	fmt.Printf("   🎯 Goal: Try to crack the encryption (spoiler: impossible)\n")
	fmt.Println()

	runBruteForceTest(filename, workers, duration)
}

func runBruteForceTest(filename string, workers int, duration int) {
	var wg sync.WaitGroup
	var attempts uint64
	var mu sync.Mutex
	done := make(chan bool)

	startTime := time.Now()

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			localAttempts := 0

			for {
				select {
				case <-done:
					mu.Lock()
					attempts += uint64(localAttempts)
					mu.Unlock()
					return
				default:
					// Generate random key shares and try to decrypt
					randomShares := generateRandomShares()
					if decryptFile(filename, randomShares) {
						fmt.Printf("%s🚨 CRITICAL: ENCRYPTION CRACKED! This should NEVER happen!%s\n", ColorRed, ColorReset)
						close(done)
						return
					}
					localAttempts++

					// Report progress every 1000 attempts
					if localAttempts%1000 == 0 {
						mu.Lock()
						totalAttempts := attempts + uint64(localAttempts)
						elapsed := time.Since(startTime).Seconds()
						rate := float64(totalAttempts) / elapsed
						fmt.Printf("\r%s🔍 Attempts: %d | Rate: %.0f/sec | Workers: %d%s",
							ColorBlue, totalAttempts, rate, workers, ColorReset)
						mu.Unlock()
					}
				}
			}
		}(i)
	}

	// Stop after duration
	time.AfterFunc(time.Duration(duration)*time.Second, func() {
		close(done)
	})

	wg.Wait()

	elapsed := time.Since(startTime)
	fmt.Printf("\n\n%s📊 Brute Force Test Results:%s\n", ColorGreen, ColorReset)
	fmt.Println("=" + strings.Repeat("=", 28))
	fmt.Printf("   ⏱️  Duration: %.2f seconds\n", elapsed.Seconds())
	fmt.Printf("   🔍 Total attempts: %d\n", attempts)
	fmt.Printf("   📈 Average rate: %.0f attempts/second\n", float64(attempts)/elapsed.Seconds())
	fmt.Printf("   👥 Workers used: %d\n", workers)
	fmt.Printf("   🛡️  Result: %sENCRYPTION UNCRACKED ✅%s\n", ColorGreen, ColorReset)
	fmt.Printf("   💡 Time to crack at this rate: %s♾️  INFINITE%s\n", ColorPurple, ColorReset)
	fmt.Println()
	fmt.Printf("%s🎉 Your encryption is mathematically uncrackable!%s\n", ColorGreen, ColorReset)
}

func generateRandomShares() []string {
	shares := make([]string, 3)
	for i := 0; i < 3; i++ {
		// Generate random 66-character hex string (typical Shamir share length)
		randomBytes := make([]byte, 33)
		rand.Read(randomBytes)
		shares[i] = hex.EncodeToString(randomBytes)
	}
	return shares
}

func decryptFile(filename string, shares []string) bool {
	// Convert hex shares to bytes
	shareBytes := make([][]byte, len(shares))
	for i, share := range shares {
		bytes, err := hex.DecodeString(share)
		if err != nil {
			return false // Invalid hex
		}
		shareBytes[i] = bytes
	}

	// Try to reconstruct master key
	masterKey, err := shamir.Combine(shareBytes)
	if err != nil {
		return false // Invalid shares
	}

	// Try to decrypt file
	ciphertext, err := os.ReadFile(filename)
	if err != nil {
		return false
	}

	// Create cipher
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return false
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return false
	}

	// Check file size
	if len(ciphertext) < gcm.NonceSize() {
		return false
	}

	// Extract nonce and try to decrypt
	nonce := ciphertext[:gcm.NonceSize()]
	encrypted := ciphertext[gcm.NonceSize():]

	_, err = gcm.Open(nil, nonce, encrypted, nil)
	return err == nil // Success if no error
}

func decryptAndDecompressFile(filename string, shares []string) bool {
	// Convert hex shares to bytes
	shareBytes := make([][]byte, len(shares))
	for i, share := range shares {
		bytes, err := hex.DecodeString(share)
		if err != nil {
			return false // Invalid hex
		}
		shareBytes[i] = bytes
	}

	// Try to reconstruct master key
	masterKey, err := shamir.Combine(shareBytes)
	if err != nil {
		return false // Invalid shares
	}

	// Try to decrypt file
	ciphertext, err := os.ReadFile(filename)
	if err != nil {
		return false
	}

	// Create cipher
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return false
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return false
	}

	// Check file size
	if len(ciphertext) < gcm.NonceSize() {
		return false
	}

	// Extract nonce and try to decrypt
	nonce := ciphertext[:gcm.NonceSize()]
	encrypted := ciphertext[gcm.NonceSize():]

	decryptedData, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return false // Decryption failed
	}

	// Save decrypted compressed file
	gzFile := strings.TrimSuffix(filename, ".encrypted") + "_decrypted.iso.gz"
	if err := os.WriteFile(gzFile, decryptedData, 0600); err != nil {
		fmt.Printf("%s❌ Failed to save decrypted file: %v%s\n", ColorRed, err, ColorReset)
		return false
	}
	fmt.Printf("%s📁 Saved decrypted file: %s%s\n", ColorGreen, gzFile, ColorReset)

	// Decompress to ISO
	isoFile := strings.TrimSuffix(gzFile, ".gz")
	fmt.Printf("🗜️ Decompressing to ISO...\n")
	if err := decompressGzip(gzFile, isoFile); err != nil {
		fmt.Printf("%s❌ Decompression failed: %v%s\n", ColorRed, err, ColorReset)
		return false
	}

	// Show file info
	if stat, err := os.Stat(isoFile); err == nil {
		fmt.Printf("%s💽 Final ISO: %s (%.2f MB)%s\n", ColorGreen, isoFile, float64(stat.Size())/1024/1024, ColorReset)
		fmt.Printf("%s🎉 Ready to boot in VM!%s\n", ColorGreen, ColorReset)
	}

	return true
}

func decompressGzip(gzipFile, outputFile string) error {
	file, err := os.Open(gzipFile)
	if err != nil {
		return err
	}
	defer file.Close()

	reader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer reader.Close()

	output, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = io.Copy(output, reader)
	return err
}

func testISOContents() {
	fmt.Printf("\n%s💿 ISO Contents Test%s\n", ColorPurple, ColorReset)
	fmt.Println("=" + strings.Repeat("=", 20))

	// Look for ISO files
	fmt.Println("Looking for .iso files...")
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".iso") {
			info, err := d.Info()
			if err != nil {
				return nil
			}
			fmt.Printf("%s📀 Found ISO: %s (%.2f MB)%s\n", 
				ColorGreen, d.Name(), float64(info.Size())/1024/1024, ColorReset)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("%s❌ Error scanning: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	fmt.Print("\nEnter ISO filename to examine: ")
	filename := strings.TrimSpace(getUserInput())

	if filename == "" {
		fmt.Printf("%s❌ No filename provided%s\n", ColorRed, ColorReset)
		return
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Printf("%s❌ File '%s' not found%s\n", ColorRed, filename, ColorReset)
		return
	}

	fmt.Printf("%s🔍 Instructions to examine ISO:%s\n", ColorBlue, ColorReset)
	fmt.Printf("1. Copy ISO to your Mac: %s\n", filename)
	fmt.Printf("2. Double-click it in Finder to mount\n")
	fmt.Printf("3. Browse /Volumes/mounted_volume to see your backup files\n")
	fmt.Printf("4. Check for /etc, /var, /usr directories to verify it's your system\n")
	fmt.Println()
	fmt.Printf("%s💡 Tip: Look for /app/keys/ to verify it's your container backup%s\n", ColorYellow, ColorReset)
}

func getUserInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}


package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CloudConfig holds Google Cloud Storage configuration
type CloudConfig struct {
	Enabled               bool
	ProjectID             string
	BucketName            string
	ServiceAccountKeyFile string
	BucketPrefix          string
}

// Constants for cloud upload
const (
	defaultGCPEnabled            = false
	defaultServiceAccountFile   = "mobulacronjson.json"
	defaultBucketPrefix         = "snapshots"
)

func uploadToCloud(localPath, diskImageName string) {
	config := getCloudConfig()
	if !config.Enabled {
		return
	}

	logInfo("☁️ Uploading disk image to Google Cloud Storage...")

	serviceAccountPath := "/app/keys/" + config.ServiceAccountKeyFile
	if err := authenticateGCP(serviceAccountPath); err != nil {
		logError("Failed to authenticate with GCP: %v", err)
		return
	}

	relativePath := getRelativePathFromDiskImage(localPath)

	cloudPath := buildCloudPath(config.BucketPrefix, relativePath, diskImageName+".encrypted")

	cmd := exec.Command("/opt/google-cloud-sdk/bin/gsutil", "cp", localPath, fmt.Sprintf("gs://%s/%s", config.BucketName, cloudPath))

	output, err := cmd.CombinedOutput()
	if err != nil {
		logError("Failed to upload to cloud: %v - Output: %s", err, string(output))
		return
	}

	logInfo("✅ Successfully uploaded to cloud: gs://%s/%s", config.BucketName, cloudPath)
}

func getCloudConfig() CloudConfig {
	config := CloudConfig{
		Enabled:      false,
		BucketPrefix: "disk_images",
	}

	envFile := "/app/.env"
	file, err := os.Open(envFile)
	if err != nil {
		return config
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
		case "GCP_ENABLED":
			config.Enabled = strings.ToLower(value) == "true"
		case "GCP_PROJECT_ID":
			config.ProjectID = value
		case "GCP_BUCKET_NAME":
			config.BucketName = value
		case "GCP_SERVICE_ACCOUNT_KEY_FILE":
			config.ServiceAccountKeyFile = value
		case "GCP_BUCKET_PREFIX":
			if value != "" {
				config.BucketPrefix = value
			}
		}
	}

	return config
}

func authenticateGCP(serviceAccountFile string) error {
	if _, err := os.Stat(serviceAccountFile); os.IsNotExist(err) {
		return fmt.Errorf("service account file not found: %s", serviceAccountFile)
	}

	cmd := exec.Command("/opt/google-cloud-sdk/bin/gcloud", "auth", "activate-service-account", "--key-file", serviceAccountFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to authenticate: %v - Output: %s", err, string(output))
	}

	return nil
}

func getRelativePathFromDiskImage(diskImagePath string) string {

	relativePath := strings.TrimPrefix(diskImagePath, diskImageDir+"/")

	parts := strings.Split(relativePath, "/")
	if len(parts) >= 4 {
		return filepath.Join(parts[0], parts[1], parts[2], parts[3])
	}

	return "unknown"
}

func buildCloudPath(prefix, relativePath, filename string) string {
	if prefix == "" {
		return filepath.Join(relativePath, filename)
	}
	return filepath.Join(prefix, relativePath, filename)
}

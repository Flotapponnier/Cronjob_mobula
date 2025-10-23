package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// CloudConfig holds S3 Object Storage configuration
type CloudConfig struct {
	Enabled         bool
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	BucketPrefix    string
}

// Constants for cloud upload
const (
	defaultS3Enabled     = false
	defaultBucketPrefix  = "backups"
	defaultS3Endpoint    = "https://s3.gra.io.cloud.ovh.net"
	defaultS3Region      = "gra"
)

func uploadToCloud(localPath, diskImageName string) {
	config := getCloudConfig()
	if !config.Enabled {
		return
	}

	logInfo("☁️ Uploading disk image to OVH S3 Object Storage...")

	if err := uploadToS3(config, localPath, diskImageName); err != nil {
		logError("Failed to upload to S3: %v", err)
		return
	}

	logInfo("✅ Successfully uploaded to S3: s3://%s/%s", config.BucketName, buildS3Key(config.BucketPrefix, localPath, diskImageName))
}

func getCloudConfig() CloudConfig {
	config := CloudConfig{
		Enabled:      defaultS3Enabled,
		BucketPrefix: defaultBucketPrefix,
		Endpoint:     defaultS3Endpoint,
		Region:       defaultS3Region,
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
		case "S3_ENABLED":
			config.Enabled = strings.ToLower(value) == "true"
		case "S3_ENDPOINT":
			if value != "" {
				config.Endpoint = value
			}
		case "S3_REGION":
			if value != "" {
				config.Region = value
			}
		case "S3_ACCESS_KEY_ID":
			config.AccessKeyID = value
		case "S3_SECRET_ACCESS_KEY":
			config.SecretAccessKey = value
		case "S3_BUCKET_NAME":
			config.BucketName = value
		case "S3_BUCKET_PREFIX":
			if value != "" {
				config.BucketPrefix = value
			}
		}
	}

	return config
}

func uploadToS3(cfg CloudConfig, localPath, diskImageName string) error {
	// Validate configuration
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return fmt.Errorf("S3 credentials are not configured")
	}
	if cfg.BucketName == "" {
		return fmt.Errorf("S3 bucket name is not configured")
	}

	// Open the file to upload
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	// Create AWS config with custom endpoint resolver for OVH
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               cfg.Endpoint,
			SigningRegion:     cfg.Region,
			HostnameImmutable: true,
		}, nil
	})

	awsConfig, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.Region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %v", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsConfig)

	// Build S3 key (path in bucket) maintaining the year/day/month/hour structure
	s3Key := buildS3Key(cfg.BucketPrefix, localPath, diskImageName+".encrypted")

	// Upload file to S3
	logInfo("Uploading %s (%d bytes) to s3://%s/%s", diskImageName, fileInfo.Size(), cfg.BucketName, s3Key)

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(cfg.BucketName),
		Key:    aws.String(s3Key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %v", err)
	}

	return nil
}

func buildS3Key(prefix, localPath, filename string) string {
	// Get the relative path from disk_images directory
	relativePath := getRelativePathFromDiskImage(localPath)

	if prefix == "" {
		return filepath.Join(relativePath, filename)
	}
	return filepath.Join(prefix, relativePath, filename)
}

func getRelativePathFromDiskImage(diskImagePath string) string {
	// Extract the year/day/month/hour structure from the disk image path
	relativePath := strings.TrimPrefix(diskImagePath, diskImageDir+"/")

	parts := strings.Split(relativePath, "/")
	if len(parts) >= 4 {
		// Return year/day/month/hour path
		return filepath.Join(parts[0], parts[1], parts[2], parts[3])
	}

	return "unknown"
}

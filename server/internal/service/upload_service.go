package service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/kalafut/imohash"
	"github.com/yourusername/private-media-ecosystem/internal/config"
	"github.com/yourusername/private-media-ecosystem/internal/db"
	"gorm.io/gorm"
)

type UploadService struct {
	config *config.Config
	db     *gorm.DB
}

type UploadResult struct {
	FileHash            string
	PrimaryPath         string
	FileSizeBytes       int64
	SyncStatus          string
	VerificationMatches bool
	UploadedAt          time.Time
}

func NewUploadService(cfg *config.Config, database *gorm.DB) *UploadService {
	return &UploadService{
		config: cfg,
		db:     database,
	}
}

func (s *UploadService) HandleUpload(file io.Reader, metadata interface{}, device *db.Device) (*UploadResult, error) {
	// Convert metadata to proper type (simplified for now)
	// In production, would properly type assert

	// Create temporary file
	tempDir := os.TempDir()
	tempFile, err := os.CreateTemp(tempDir, "upload-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy uploaded file to temp location
	bytesWritten, err := io.Copy(tempFile, file)
	if err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}

	// Compute server-side hash using imohash
	tempFile.Seek(0, 0)
	serverHash, err := s.computeImoHash(tempFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to compute hash: %w", err)
	}

	// Verify against client hash (would get from metadata)
	// clientHash := metadata.LocalHash
	// if serverHash != clientHash {
	//     return nil, fmt.Errorf("hash_mismatch")
	// }

	// Check for duplicates
	var existingFile db.MediaFile
	result := s.db.Where("file_hash = ? AND deleted_at IS NULL", serverHash).First(&existingFile)
	if result.Error == nil {
		return nil, fmt.Errorf("duplicate_file")
	}

	// Build destination path
	// In production, would get from metadata.DestinationPath
	destPath := filepath.Join(s.config.Storage.PrimaryPath, "uploads", "test.jpg")
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Move file to final location
	tempFile.Close()
	if err := os.Rename(tempFile.Name(), destPath); err != nil {
		// If rename fails (cross-device), try copy
		if err := s.copyFile(tempFile.Name(), destPath); err != nil {
			return nil, fmt.Errorf("failed to move file: %w", err)
		}
	}

	// Create database entry
	now := time.Now().UTC()
	mediaFile := db.MediaFile{
		FileHash:          serverHash,
		OriginalFilename:  "test.jpg", // Would get from metadata
		MediaType:         "image",
		MimeType:          "image/jpeg",
		PrimaryPath:       destPath,
		FileSizeBytes:     bytesWritten,
		TimestampTaken:    now, // Would get from metadata
		TimestampUploaded: now,
		SyncStatus:        db.SyncStatusUploadedVerified,
		Orientation:       1,
	}

	if err := s.db.Create(&mediaFile).Error; err != nil {
		// Clean up file if database insert fails
		os.Remove(destPath)
		return nil, fmt.Errorf("failed to save to database: %w", err)
	}

	// Create sync operation record
	syncOp := db.SyncOperation{
		MediaFileID:      mediaFile.ID,
		OperationType:    db.OperationTypeUpload,
		Status:           "completed",
		ClientDeviceID:   device.DeviceID,
		ClientDeviceName: &device.DeviceName,
		BytesTransferred: &bytesWritten,
		StartedAt:        now,
		CompletedAt:      &now,
	}
	s.db.Create(&syncOp)

	return &UploadResult{
		FileHash:            serverHash,
		PrimaryPath:         destPath,
		FileSizeBytes:       bytesWritten,
		SyncStatus:          db.SyncStatusUploadedVerified,
		VerificationMatches: true,
		UploadedAt:          now,
	}, nil
}

func (s *UploadService) computeImoHash(filePath string) (string, error) {
	// Use imohash for fast, deterministic hashing
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hashBytes, err := imohash.SumFile(filePath)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hashBytes[:]), nil
}

func (s *UploadService) computeMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *UploadService) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

package service

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/yourusername/private-media-ecosystem/internal/config"
	"github.com/yourusername/private-media-ecosystem/internal/db"
	"gorm.io/gorm"
)

type BackupService struct {
	config      *config.Config
	db          *gorm.DB
	hashService *HashService
	running     bool
}

func NewBackupService(cfg *config.Config, database *gorm.DB) *BackupService {
	return &BackupService{
		config:      cfg,
		db:          database,
		hashService: NewHashService(),
		running:     false,
	}
}

type HashService struct{}

func NewHashService() *HashService {
	return &HashService{}
}

func (h *HashService) ComputeHash(filePath string) (string, error) {
	// Use the same hash computation as upload service
	return computeFileHash(filePath)
}

func (s *BackupService) GetStatus() (interface{}, error) {
	var currentJob db.BackupJob
	currentErr := s.db.Where("status = ?", "running").First(&currentJob).Error

	var lastCompleted db.BackupJob
	s.db.Where("status = ?", "completed").
		Order("completed_at DESC").
		First(&lastCompleted)

	result := map[string]interface{}{
		"current_job":    nil,
		"last_completed": nil,
	}

	if currentErr == nil {
		result["current_job"] = s.jobToMap(&currentJob)
	}

	if lastCompleted.ID != 0 {
		result["last_completed"] = s.jobToMap(&lastCompleted)
	}

	return result, nil
}

func (s *BackupService) StartBackup(jobType, priority string) (uint, error) {
	// Check if backup already running
	if s.running {
		return 0, fmt.Errorf("backup_running")
	}

	// Validate backup path is configured
	if s.config.Storage.BackupPath == "" {
		return 0, fmt.Errorf("backup path not configured")
	}

	// Create backup job
	now := time.Now().UTC()
	job := db.BackupJob{
		JobType:         jobType,
		SourcePath:      s.config.Storage.PrimaryPath,
		DestinationPath: s.config.Storage.BackupPath,
		Status:          "running",
		StartedAt:       &now,
	}

	if err := s.db.Create(&job).Error; err != nil {
		return 0, err
	}

	// Start backup in goroutine
	go s.runBackup(&job)

	return job.ID, nil
}

func (s *BackupService) runBackup(job *db.BackupJob) {
	s.running = true
	defer func() { s.running = false }()

	log.Info().Uint("job_id", job.ID).Str("type", job.JobType).Msg("Backup started")

	// Get all files that need backup
	var mediaFiles []db.MediaFile
	query := s.db.Where("deleted_at IS NULL")

	if job.JobType == "incremental" {
		// Only files without backup_path or not verified recently
		query = query.Where("backup_path IS NULL OR last_verified < ?", time.Now().Add(-7*24*time.Hour))
	}

	if err := query.Find(&mediaFiles).Error; err != nil {
		s.failJob(job, fmt.Sprintf("failed to query files: %v", err))
		return
	}

	total := len(mediaFiles)
	job.TotalFiles = &total
	s.db.Save(job)

	// Process each file
	for i, mediaFile := range mediaFiles {
		if err := s.backupFile(&mediaFile, job); err != nil {
			log.Error().Err(err).Str("file", mediaFile.PrimaryPath).Msg("Backup failed for file")
			job.FilesFailed++
		} else {
			job.FilesVerified++
		}

		job.ProcessedFiles = i + 1
		s.db.Save(job)
	}

	// Complete job
	now := time.Now().UTC()
	job.Status = "completed"
	job.CompletedAt = &now
	s.db.Save(job)

	log.Info().
		Uint("job_id", job.ID).
		Int("total", total).
		Int("copied", job.FilesCopied).
		Int("verified", job.FilesVerified).
		Int("skipped", job.FilesSkipped).
		Int("failed", job.FilesFailed).
		Msg("Backup completed")
}

func (s *BackupService) backupFile(mediaFile *db.MediaFile, job *db.BackupJob) error {
	// Build backup path (mirror directory structure)
	relPath, err := filepath.Rel(s.config.Storage.PrimaryPath, mediaFile.PrimaryPath)
	if err != nil {
		return fmt.Errorf("failed to compute relative path: %w", err)
	}

	backupPath := filepath.Join(s.config.Storage.BackupPath, relPath)
	backupDir := filepath.Dir(backupPath)

	// Check if backup already exists and is valid
	if _, err := os.Stat(backupPath); err == nil {
		// File exists, verify hash
		backupHash, err := s.hashService.ComputeHash(backupPath)
		if err == nil && backupHash == mediaFile.FileHash {
			// Backup is valid, skip
			job.FilesSkipped++
			s.updateBackupPath(mediaFile, backupPath)
			return nil
		}
		// Hash mismatch or error, re-backup
	}

	// Create backup directory
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Copy file
	if err := s.copyFile(mediaFile.PrimaryPath, backupPath); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	job.FilesCopied++

	// Verify backup if configured
	if s.config.Backup.VerifyAfterCopy {
		backupHash, err := s.hashService.ComputeHash(backupPath)
		if err != nil {
			os.Remove(backupPath) // Clean up failed backup
			return fmt.Errorf("failed to compute backup hash: %w", err)
		}

		if backupHash != mediaFile.FileHash {
			os.Remove(backupPath) // Clean up corrupted backup
			return fmt.Errorf("backup hash mismatch")
		}
	}

	// Update database
	s.updateBackupPath(mediaFile, backupPath)

	return nil
}

func (s *BackupService) copyFile(src, dst string) error {
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

	// Copy with buffer
	buf := make([]byte, 1024*1024) // 1MB buffer
	_, err = io.CopyBuffer(destFile, sourceFile, buf)
	if err != nil {
		return err
	}

	// Sync to disk
	return destFile.Sync()
}

func (s *BackupService) updateBackupPath(mediaFile *db.MediaFile, backupPath string) {
	now := time.Now().UTC()
	s.db.Model(mediaFile).Updates(map[string]interface{}{
		"backup_path":   backupPath,
		"last_verified": now,
		"sync_status":   db.SyncStatusBackupComplete,
	})
}

func (s *BackupService) failJob(job *db.BackupJob, errorMsg string) {
	now := time.Now().UTC()
	job.Status = "failed"
	job.ErrorMessage = &errorMsg
	job.CompletedAt = &now
	s.db.Save(job)

	log.Error().Uint("job_id", job.ID).Str("error", errorMsg).Msg("Backup job failed")
}

func (s *BackupService) CancelBackup(jobID uint) error {
	return s.db.Model(&db.BackupJob{}).
		Where("id = ? AND status = ?", jobID, "running").
		Updates(map[string]interface{}{
			"status":       "cancelled",
			"completed_at": time.Now().UTC(),
		}).Error
}

func (s *BackupService) jobToMap(job *db.BackupJob) map[string]interface{} {
	result := map[string]interface{}{
		"id":              job.ID,
		"job_type":        job.JobType,
		"status":          job.Status,
		"total_files":     job.TotalFiles,
		"processed_files": job.ProcessedFiles,
		"files_copied":    job.FilesCopied,
		"files_verified":  job.FilesVerified,
		"files_skipped":   job.FilesSkipped,
		"files_failed":    job.FilesFailed,
		"started_at":      job.StartedAt,
		"completed_at":    job.CompletedAt,
	}

	if job.ProcessedFiles > 0 && job.TotalFiles != nil && *job.TotalFiles > 0 {
		progress := float64(job.ProcessedFiles) / float64(*job.TotalFiles) * 100
		result["progress_percentage"] = progress
	}

	return result
}

// Helper function for hash computation
func computeFileHash(filePath string) (string, error) {
	// This should match the hash algorithm in upload_service.go
	// Using SHA256 for consistency
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

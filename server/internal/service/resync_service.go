package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/yourusername/private-media-ecosystem/internal/config"
	"github.com/yourusername/private-media-ecosystem/internal/db"
	"gorm.io/gorm"
)

type ResyncService struct {
	config      *config.Config
	db          *gorm.DB
	hashService *HashService
}

func NewResyncService(cfg *config.Config, database *gorm.DB) *ResyncService {
	return &ResyncService{
		config:      cfg,
		db:          database,
		hashService: NewHashService(),
	}
}

func (s *ResyncService) StartResync(scanPath string, dryRun, recomputeHashes bool) (uint, error) {
	// Create resync event
	now := time.Now().UTC()
	event := db.ResyncEvent{
		ScanPath:      scanPath,
		TriggerSource: "manual",
		StartedAt:     now,
	}

	if err := s.db.Create(&event).Error; err != nil {
		return 0, err
	}

	// Start resync in goroutine
	go s.runResync(&event, dryRun, recomputeHashes)

	return event.ID, nil
}

func (s *ResyncService) runResync(event *db.ResyncEvent, dryRun, recomputeHashes bool) {
	log.Info().Uint("event_id", event.ID).Str("scan_path_param", event.ScanPath).Str("primary_path", s.config.Storage.PrimaryPath).Msg("Resync started")

	startTime := time.Now()

	// Use primary path directly
	fullPath := s.config.Storage.PrimaryPath
	
	log.Info().Str("full_path", fullPath).Msg("Scanning directory")

	// Discover all files on disk
	filesOnDisk := make(map[string]string) // hash -> path
	var filesDiscovered int

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warn().Err(err).Str("path", path).Msg("Error walking path")
			return nil // Continue walking
		}

		// Skip directories and hidden files
		if info.IsDir() || info.Name()[0] == '.' {
			return nil
		}

		// Only process media files
		if !s.isMediaFile(path) {
			log.Debug().Str("path", path).Msg("Skipping non-media file")
			return nil
		}

		log.Info().Str("path", path).Msg("Processing media file")
		filesDiscovered++

		// Compute hash
		hash, err := s.hashService.ComputeHash(path)
		if err != nil {
			log.Warn().Err(err).Str("path", path).Msg("Failed to compute hash")
			return nil
		}

		filesOnDisk[hash] = path
		return nil
	})

	if err != nil {
		s.failEvent(event, fmt.Sprintf("walk failed: %v", err))
		return
	}

	event.FilesDiscovered = &filesDiscovered
	s.db.Save(event)

	// Get all files from database
	var dbFiles []db.MediaFile
	s.db.Where("deleted_at IS NULL").Find(&dbFiles)

	// Create lookup maps
	dbByHash := make(map[string]*db.MediaFile)
	for i := range dbFiles {
		dbByHash[dbFiles[i].FileHash] = &dbFiles[i]
	}

	// Reconciliation
	var filesMatched, filesOrphaned, filesNew, pathsUpdated, duplicatesFound int

	// Check files in database
	for _, dbFile := range dbFiles {
		if diskPath, exists := filesOnDisk[dbFile.FileHash]; exists {
			// File found on disk
			filesMatched++

			// Check if path changed
			if diskPath != dbFile.PrimaryPath {
				pathsUpdated++
				log.Info().
					Str("hash", dbFile.FileHash).
					Str("old_path", dbFile.PrimaryPath).
					Str("new_path", diskPath).
					Msg("File moved")

				if !dryRun {
					// Update path in database
					s.db.Model(&dbFile).Update("primary_path", diskPath)
				}
			}

			// Remove from disk map (processed)
			delete(filesOnDisk, dbFile.FileHash)
		} else {
			// File in DB but not on disk (orphaned)
			filesOrphaned++
			log.Warn().
				Str("hash", dbFile.FileHash).
				Str("path", dbFile.PrimaryPath).
				Msg("Orphaned database entry")

			if !dryRun && !recomputeHashes {
				// Soft delete orphaned entries
				now := time.Now().UTC()
				s.db.Model(&dbFile).Updates(map[string]interface{}{
					"deleted_at":      now,
					"deletion_reason": "orphaned_during_resync",
				})
			}
		}
	}

	// Files remaining in filesOnDisk are new (not in database)
	filesNew = len(filesOnDisk)
	for hash, path := range filesOnDisk {
		log.Info().Str("hash", hash).Str("path", path).Msg("New file discovered")

		if !dryRun {
			// Add to database
			s.addNewFile(hash, path)
		}
	}

	// Detect duplicates (same hash, different paths)
	hashCounts := make(map[string]int)
	for _, file := range dbFiles {
		hashCounts[file.FileHash]++
	}
	for _, count := range hashCounts {
		if count > 1 {
			duplicatesFound++
		}
	}

	// Update event
	duration := time.Since(startTime)
	now := time.Now().UTC()
	durationMs := duration.Milliseconds()

	event.FilesMatched = &filesMatched
	event.FilesOrphaned = &filesOrphaned
	event.FilesNew = &filesNew
	event.PathsUpdated = &pathsUpdated
	event.DuplicatesFound = &duplicatesFound
	event.CompletedAt = &now
	event.ScanDurationMs = &durationMs

	s.db.Save(event)

	log.Info().
		Uint("event_id", event.ID).
		Int("discovered", filesDiscovered).
		Int("matched", filesMatched).
		Int("orphaned", filesOrphaned).
		Int("new", filesNew).
		Int("path_updates", pathsUpdated).
		Int("duplicates", duplicatesFound).
		Dur("duration", duration).
		Msg("Resync completed")
}

func (s *ResyncService) addNewFile(hash, path string) error {
	// Get file info
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Determine media type
	mediaType := "image"
	if s.isVideoFile(path) {
		mediaType = "video"
	}

	// Determine MIME type
	mimeType := "application/octet-stream"
	ext := strings.ToLower(filepath.Ext(path))
	if mediaType == "image" {
		switch ext {
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		case ".png":
			mimeType = "image/png"
		case ".gif":
			mimeType = "image/gif"
		case ".webp":
			mimeType = "image/webp"
		default:
			mimeType = "image/jpeg"
		}
	} else if mediaType == "video" {
		switch ext {
		case ".mp4":
			mimeType = "video/mp4"
		case ".mov":
			mimeType = "video/quicktime"
		case ".avi":
			mimeType = "video/x-msvideo"
		case ".mkv":
			mimeType = "video/x-matroska"
		case ".webm":
			mimeType = "video/webm"
		default:
			mimeType = "video/mp4"
		}
	}

	// Create database entry
	mediaFile := db.MediaFile{
		FileHash:          hash,
		OriginalFilename:  filepath.Base(path),
		MediaType:         mediaType,
		MimeType:          mimeType,
		PrimaryPath:       path,
		FileSizeBytes:     stat.Size(),
		TimestampTaken:    stat.ModTime(), // Use file mtime as fallback
		TimestampUploaded: time.Now().UTC(),
		SyncStatus:        db.SyncStatusUploadedVerified,
		Orientation:       1,
	}

	return s.db.Create(&mediaFile).Error
}

func (s *ResyncService) isMediaFile(path string) bool {
	return s.isImageFile(path) || s.isVideoFile(path)
}

func (s *ResyncService) isImageFile(path string) bool {
	ext := filepath.Ext(path)
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".tiff", ".tif", ".bmp":
		return true
	}
	return false
}

func (s *ResyncService) isVideoFile(path string) bool {
	ext := filepath.Ext(path)
	switch strings.ToLower(ext) {
	case ".mp4", ".mov", ".avi", ".mkv", ".webm", ".m4v", ".flv":
		return true
	}
	return false
}

func (s *ResyncService) failEvent(event *db.ResyncEvent, errorMsg string) {
	now := time.Now().UTC()
	event.CompletedAt = &now
	s.db.Save(event)

	log.Error().Uint("event_id", event.ID).Str("error", errorMsg).Msg("Resync failed")
}

func (s *ResyncService) GetStatus(eventID uint) (interface{}, error) {
	var event db.ResyncEvent
	err := s.db.Where("id = ?", eventID).First(&event).Error
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"id":               event.ID,
		"scan_path":        event.ScanPath,
		"trigger_source":   event.TriggerSource,
		"files_discovered": event.FilesDiscovered,
		"files_matched":    event.FilesMatched,
		"files_orphaned":   event.FilesOrphaned,
		"files_new":        event.FilesNew,
		"paths_updated":    event.PathsUpdated,
		"duplicates_found": event.DuplicatesFound,
		"started_at":       event.StartedAt,
		"completed_at":     event.CompletedAt,
	}

	if event.CompletedAt != nil {
		result["status"] = "completed"
	} else {
		result["status"] = "running"
	}

	if event.ScanDurationMs != nil {
		result["scan_duration_ms"] = *event.ScanDurationMs
	}

	return result, nil
}

func (s *ResyncService) QueueReconciliation(path string) {
	// Called by filesystem watcher when files are moved/renamed
	log.Info().Str("path", path).Msg("Queuing reconciliation")
	// For now, just log. In production, this would trigger a targeted resync
}

package service

import (
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/rs/zerolog/log"
	"github.com/yourusername/private-media-ecosystem/internal/config"
	"github.com/yourusername/private-media-ecosystem/internal/db"
	"gorm.io/gorm"
)

type ThumbnailService struct {
	config *config.Config
	db     *gorm.DB
}

func NewThumbnailService(cfg *config.Config, database *gorm.DB) *ThumbnailService {
	// Ensure cache directory exists
	os.MkdirAll(cfg.Thumbnail.CachePath, 0755)
	return &ThumbnailService{
		config: cfg,
		db:     database,
	}
}

func (s *ThumbnailService) GetThumbnail(hash, size string, quality int) (string, error) {
	// Build thumbnail path
	thumbPath := s.getThumbnailPath(hash, size)

	// Check if thumbnail exists in cache
	if _, err := os.Stat(thumbPath); err == nil {
		return thumbPath, nil
	}

	// Get media file from database
	var mediaFile db.MediaFile
	if err := s.db.Where("file_hash = ?", hash).First(&mediaFile).Error; err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	// Generate thumbnail based on media type
	var err error
	if mediaFile.MediaType == "image" {
		err = s.generateImageThumbnail(mediaFile.PrimaryPath, thumbPath, size, quality)
	} else if mediaFile.MediaType == "video" {
		err = s.generateVideoThumbnail(mediaFile.PrimaryPath, thumbPath, size, quality)
	} else {
		return "", fmt.Errorf("unsupported media type: %s", mediaFile.MediaType)
	}

	if err != nil {
		return "", fmt.Errorf("thumbnail generation failed: %w", err)
	}

	return thumbPath, nil
}

func (s *ThumbnailService) generateImageThumbnail(sourcePath, destPath string, size string, quality int) error {
	// Open source image
	src, err := imaging.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}

	// Get target dimensions based on size
	targetWidth := s.getSizePixels(size)

	// Resize image maintaining aspect ratio
	thumb := imaging.Resize(src, targetWidth, 0, imaging.Lanczos)

	// Handle orientation if needed
	oriented := s.autoOrient(thumb, sourcePath)
	thumb = imaging.Clone(oriented)

	// Save thumbnail
	err = imaging.Save(thumb, destPath, imaging.JPEGQuality(quality))
	if err != nil {
		return fmt.Errorf("failed to save thumbnail: %w", err)
	}

	log.Debug().Str("source", sourcePath).Str("dest", destPath).Msg("Image thumbnail generated")
	return nil
}

func (s *ThumbnailService) generateVideoThumbnail(sourcePath, destPath string, size string, quality int) error {
	// Check if ffmpeg is available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found: %w", err)
	}

	// Extract frame at 1 second
	cmd := exec.Command("ffmpeg",
		"-i", sourcePath,
		"-ss", "00:00:01.000",
		"-vframes", "1",
		"-vf", fmt.Sprintf("scale=%d:-1", s.getSizePixels(size)),
		"-q:v", fmt.Sprintf("%d", 31-quality/4), // Convert quality to ffmpeg scale
		destPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w, output: %s", err, string(output))
	}

	log.Debug().Str("source", sourcePath).Str("dest", destPath).Msg("Video thumbnail generated")
	return nil
}

func (s *ThumbnailService) autoOrient(img image.Image, sourcePath string) image.Image {
	// Try to get orientation from EXIF
	file, err := os.Open(sourcePath)
	if err != nil {
		return img
	}
	defer file.Close()

	// Simple orientation detection (full EXIF parsing done elsewhere)
	// For now, just return original
	// TODO: Integrate with ExifService for proper orientation
	return img
}

func (s *ThumbnailService) getSizePixels(size string) int {
	switch size {
	case "small":
		return 150
	case "medium":
		return 300
	case "large":
		return 600
	default:
		return 300
	}
}

func (s *ThumbnailService) getThumbnailPath(hash, size string) string {
	// Create subdirectory based on first 2 chars of hash (for better filesystem performance)
	subdir := hash[:2]
	dir := filepath.Join(s.config.Thumbnail.CachePath, subdir)
	os.MkdirAll(dir, 0755)

	filename := fmt.Sprintf("%s_%s.jpg", hash, size)
	return filepath.Join(dir, filename)
}

func (s *ThumbnailService) IsVideoFile(filePath string) bool {
	ext := strings.ToLower(filePath)
	return strings.HasSuffix(ext, ".mp4") ||
		strings.HasSuffix(ext, ".mov") ||
		strings.HasSuffix(ext, ".avi") ||
		strings.HasSuffix(ext, ".mkv") ||
		strings.HasSuffix(ext, ".webm") ||
		strings.HasSuffix(ext, ".m4v")
}

func (s *ThumbnailService) IsImageFile(filePath string) bool {
	ext := strings.ToLower(filePath)
	return strings.HasSuffix(ext, ".jpg") ||
		strings.HasSuffix(ext, ".jpeg") ||
		strings.HasSuffix(ext, ".png") ||
		strings.HasSuffix(ext, ".gif") ||
		strings.HasSuffix(ext, ".webp") ||
		strings.HasSuffix(ext, ".tiff") ||
		strings.HasSuffix(ext, ".tif") ||
		strings.HasSuffix(ext, ".bmp")
}

// ClearCache removes all cached thumbnails
func (s *ThumbnailService) ClearCache() error {
	return os.RemoveAll(s.config.Thumbnail.CachePath)
}

// GetCacheSize returns total size of thumbnail cache in bytes
func (s *ThumbnailService) GetCacheSize() (int64, error) {
	var size int64
	err := filepath.Walk(s.config.Thumbnail.CachePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

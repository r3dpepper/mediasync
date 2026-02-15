package service

import (
	"time"

	"github.com/yourusername/private-media-ecosystem/internal/db"
	"gorm.io/gorm"
)

// SyncService handles sync status queries
type SyncService struct {
	db *gorm.DB
}

func NewSyncService(database *gorm.DB) *SyncService {
	return &SyncService{db: database}
}

func (s *SyncService) GetStatus(hash string) (*db.MediaFile, error) {
	var mediaFile db.MediaFile
	err := s.db.Where("file_hash = ? AND deleted_at IS NULL", hash).First(&mediaFile).Error
	return &mediaFile, err
}

func (s *SyncService) GetByHash(hash string) (*db.MediaFile, error) {
	var mediaFile db.MediaFile
	err := s.db.Where("file_hash = ?", hash).First(&mediaFile).Error
	return &mediaFile, err
}

func (s *SyncService) UpdateVerification(mediaFileID uint) error {
	return s.db.Model(&db.MediaFile{}).
		Where("id = ?", mediaFileID).
		Updates(map[string]interface{}{
			"verification_count": gorm.Expr("verification_count + 1"),
			"last_verified":      time.Now().UTC(),
		}).Error
}

func (s *SyncService) MarkDeletedLocal(mediaFileID uint, deviceID string) error {
	return s.db.Model(&db.MediaFile{}).
		Where("id = ?", mediaFileID).
		Update("sync_status", db.SyncStatusDeletedLocal).Error
}

// TimelineService handles timeline and search queries
type TimelineService struct {
	db *gorm.DB
}

func NewTimelineService(database *gorm.DB) *TimelineService {
	return &TimelineService{db: database}
}

type TimelineResponse struct {
	TotalCount int            `json:"total_count"`
	StartDate  *time.Time     `json:"start_date"`
	EndDate    *time.Time     `json:"end_date"`
	Items      []TimelineItem `json:"items"`
	Pagination PaginationInfo `json:"pagination"`
}

type TimelineItem struct {
	FileHash         string    `json:"file_hash"`
	OriginalFilename string    `json:"original_filename"`
	MediaType        string    `json:"media_type"`
	TimestampTaken   time.Time `json:"timestamp_taken"`
	PrimaryPath      string    `json:"primary_path"`
	Width            *int      `json:"width"`
	Height           *int      `json:"height"`
	Orientation      int       `json:"orientation"`
	ThumbnailURL     string    `json:"thumbnail_url"`
	Location         *Location `json:"location,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      *string `json:"name,omitempty"`
}

type PaginationInfo struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

func (s *TimelineService) GetTimeline(
	start, end *time.Time,
	mediaType string,
	limit, offset int,
	sortOrder string,
) (*TimelineResponse, error) {
	query := s.db.Model(&db.MediaFile{}).
		Where("deleted_at IS NULL")

	if start != nil {
		query = query.Where("timestamp_taken >= ?", start)
	}
	if end != nil {
		query = query.Where("timestamp_taken <= ?", end)
	}
	if mediaType != "" {
		query = query.Where("media_type = ?", mediaType)
	}

	// Get total count
	var totalCount int64
	query.Count(&totalCount)

	// Apply sorting
	if sortOrder == "asc" {
		query = query.Order("timestamp_taken ASC")
	} else {
		query = query.Order("timestamp_taken DESC")
	}

	// Apply pagination
	query = query.Limit(limit).Offset(offset)

	// Execute query
	var mediaFiles []db.MediaFile
	if err := query.Find(&mediaFiles).Error; err != nil {
		return nil, err
	}

	// Convert to timeline items
	items := make([]TimelineItem, len(mediaFiles))
	for i, mf := range mediaFiles {
		items[i] = TimelineItem{
			FileHash:         mf.FileHash,
			OriginalFilename: mf.OriginalFilename,
			MediaType:        mf.MediaType,
			TimestampTaken:   mf.TimestampTaken,
			PrimaryPath:      mf.PrimaryPath,
			Width:            mf.Width,
			Height:           mf.Height,
			Orientation:      mf.Orientation,
			ThumbnailURL:     "/api/thumbnail/" + mf.FileHash,
		}
	}

	return &TimelineResponse{
		TotalCount: int(totalCount),
		StartDate:  start,
		EndDate:    end,
		Items:      items,
		Pagination: PaginationInfo{
			Limit:   limit,
			Offset:  offset,
			HasMore: offset+limit < int(totalCount),
		},
	}, nil
}

func (s *TimelineService) Search(query, mediaType string, limit, offset int) (interface{}, error) {
	// TODO: Implement FTS5 search
	return map[string]interface{}{
		"query":         query,
		"total_results": 0,
		"items":         []interface{}{},
		"note":          "Search implementation pending",
	}, nil
}

func (s *TimelineService) GetMetadata(hash string) (interface{}, error) {
	var mediaFile db.MediaFile
	if err := s.db.Where("file_hash = ?", hash).First(&mediaFile).Error; err != nil {
		return nil, err
	}

	var exifData db.ExifMetadata
	s.db.Where("media_file_id = ?", mediaFile.ID).First(&exifData)

	return map[string]interface{}{
		"file_hash":         mediaFile.FileHash,
		"original_filename": mediaFile.OriginalFilename,
		"media_type":        mediaFile.MediaType,
		"mime_type":         mediaFile.MimeType,
		"file_size_bytes":   mediaFile.FileSizeBytes,
		"dimensions": map[string]interface{}{
			"width":       mediaFile.Width,
			"height":      mediaFile.Height,
			"orientation": mediaFile.Orientation,
		},
		"timestamps": map[string]interface{}{
			"taken":    mediaFile.TimestampTaken,
			"uploaded": mediaFile.TimestampUploaded,
			"modified": mediaFile.TimestampModified,
		},
	}, nil
}

type Stats struct {
	TotalFiles       int64                  `json:"total_files"`
	TotalSizeBytes   int64                  `json:"total_size_bytes"`
	FilesByType      map[string]int64       `json:"files_by_type"`
	SyncStatusCounts map[string]int64       `json:"sync_status_counts"`
	BackupStatus     map[string]interface{} `json:"backup_status"`
}

func (s *TimelineService) GetStats() (*Stats, error) {
	stats := &Stats{
		FilesByType:      make(map[string]int64),
		SyncStatusCounts: make(map[string]int64),
	}

	// Total files
	s.db.Model(&db.MediaFile{}).
		Where("deleted_at IS NULL").
		Count(&stats.TotalFiles)

	// Total size
	var totalSize struct{ Total int64 }
	s.db.Model(&db.MediaFile{}).
		Where("deleted_at IS NULL").
		Select("SUM(file_size_bytes) as total").
		Scan(&totalSize)
	stats.TotalSizeBytes = totalSize.Total

	// Files by type
	var typeResults []struct {
		MediaType string
		Count     int64
	}
	s.db.Model(&db.MediaFile{}).
		Where("deleted_at IS NULL").
		Select("media_type, COUNT(*) as count").
		Group("media_type").
		Scan(&typeResults)
	for _, tr := range typeResults {
		stats.FilesByType[tr.MediaType] = tr.Count
	}

	// Sync status counts
	var statusResults []struct {
		SyncStatus string
		Count      int64
	}
	s.db.Model(&db.MediaFile{}).
		Where("deleted_at IS NULL").
		Select("sync_status, COUNT(*) as count").
		Group("sync_status").
		Scan(&statusResults)
	for _, sr := range statusResults {
		stats.SyncStatusCounts[sr.SyncStatus] = sr.Count
	}

	// Backup status
	var backedUpCount int64
	s.db.Model(&db.MediaFile{}).
		Where("deleted_at IS NULL AND backup_path IS NOT NULL").
		Count(&backedUpCount)

	stats.BackupStatus = map[string]interface{}{
		"files_backed_up": backedUpCount,
		"files_pending":   stats.TotalFiles - backedUpCount,
	}

	return stats, nil
}

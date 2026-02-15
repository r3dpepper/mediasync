package db

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

// InitDB initializes the SQLite database
func InitDB(dbPath string) (*gorm.DB, error) {
	var err error
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Auto-migrate models
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("auto-migration failed: %w", err)
	}

	log.Info().Str("path", dbPath).Msg("Database initialized")
	return db, nil
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&MediaFile{},
		&ExifMetadata{},
		&SyncOperation{},
		&BackupJob{},
		&ResyncEvent{},
		&Device{},
	)
}

// Close closes the database connection
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return db
}

// MediaFile represents a media file in the system
type MediaFile struct {
	ID                uint    `gorm:"primaryKey"`
	FileHash          string  `gorm:"uniqueIndex;not null;size:128"`
	OriginalFilename  string  `gorm:"not null;size:512"`
	MediaType         string  `gorm:"not null;size:32;index"`
	MimeType          string  `gorm:"not null;size:128"`
	PrimaryPath       string  `gorm:"not null;size:1024;index"`
	BackupPath        *string `gorm:"size:1024"`
	FileSizeBytes     int64   `gorm:"not null"`
	DurationSeconds   *float64
	Width             *int
	Height            *int
	Orientation       int       `gorm:"default:1"`
	TimestampTaken    time.Time `gorm:"not null;index"`
	TimestampUploaded time.Time `gorm:"not null"`
	TimestampModified *time.Time
	SyncStatus        string `gorm:"not null;size:64;index;default:'pending_upload'"`
	VerificationCount int    `gorm:"default:0"`
	LastVerified      *time.Time
	DeletedAt         *time.Time `gorm:"index"`
	DeletionReason    *string    `gorm:"size:256"`
	CreatedAt         time.Time  `gorm:"autoCreateTime"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime"`
}

// ExifMetadata stores EXIF data for media files
type ExifMetadata struct {
	ID           uint      `gorm:"primaryKey"`
	MediaFileID  uint      `gorm:"uniqueIndex;not null"`
	MediaFile    MediaFile `gorm:"foreignKey:MediaFileID;constraint:OnDelete:CASCADE"`
	CameraMake   *string   `gorm:"size:128"`
	CameraModel  *string   `gorm:"size:128"`
	LensModel    *string   `gorm:"size:128"`
	Aperture     *float64
	ShutterSpeed *string `gorm:"size:64"`
	ISO          *int
	FocalLength  *float64
	Flash        *string  `gorm:"size:64"`
	GPSLatitude  *float64 `gorm:"index"`
	GPSLongitude *float64 `gorm:"index"`
	GPSAltitude  *float64
	GPSTimestamp *time.Time
	LocationName *string `gorm:"size:256"`
	FrameRate    *float64
	BitRate      *int64
	Codec        *string   `gorm:"size:64"`
	RawExifJSON  *string   `gorm:"type:text"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

// SyncOperation tracks upload/sync operations
type SyncOperation struct {
	ID                 uint      `gorm:"primaryKey"`
	MediaFileID        uint      `gorm:"not null;index"`
	MediaFile          MediaFile `gorm:"foreignKey:MediaFileID;constraint:OnDelete:CASCADE"`
	OperationType      string    `gorm:"not null;size:64"`
	Status             string    `gorm:"not null;size:64;index"`
	ClientDeviceID     string    `gorm:"not null;size:128;index"`
	ClientDeviceName   *string   `gorm:"size:256"`
	ClientAppVersion   *string   `gorm:"size:32"`
	BytesTransferred   *int64
	TransferDurationMs *int64
	RetryCount         int       `gorm:"default:0"`
	ErrorMessage       *string   `gorm:"type:text"`
	ErrorStacktrace    *string   `gorm:"type:text"`
	StartedAt          time.Time `gorm:"not null"`
	CompletedAt        *time.Time
	CreatedAt          time.Time `gorm:"autoCreateTime"`
}

// BackupJob tracks backup operations
type BackupJob struct {
	ID              uint   `gorm:"primaryKey"`
	JobType         string `gorm:"not null;size:64"`
	SourcePath      string `gorm:"not null;size:1024"`
	DestinationPath string `gorm:"not null;size:1024"`
	Status          string `gorm:"not null;size:64;index"`
	TotalFiles      *int
	ProcessedFiles  int `gorm:"default:0"`
	TotalBytes      *int64
	ProcessedBytes  int64   `gorm:"default:0"`
	FilesCopied     int     `gorm:"default:0"`
	FilesVerified   int     `gorm:"default:0"`
	FilesSkipped    int     `gorm:"default:0"`
	FilesFailed     int     `gorm:"default:0"`
	ErrorMessage    *string `gorm:"type:text"`
	StartedAt       *time.Time
	CompletedAt     *time.Time
	CreatedAt       time.Time `gorm:"autoCreateTime"`
}

// ResyncEvent tracks filesystem reconciliation events
type ResyncEvent struct {
	ID               uint   `gorm:"primaryKey"`
	ScanPath         string `gorm:"not null;size:1024"`
	TriggerSource    string `gorm:"not null;size:64"`
	FilesDiscovered  *int
	FilesMatched     *int
	FilesOrphaned    *int
	FilesNew         *int
	PathsUpdated     *int
	DuplicatesFound  *int
	HashesRecomputed *int
	ScanDurationMs   *int64
	StartedAt        time.Time `gorm:"not null"`
	CompletedAt      *time.Time
	CreatedAt        time.Time `gorm:"autoCreateTime"`
}

// Device represents a registered Android client
type Device struct {
	ID          uint      `gorm:"primaryKey"`
	DeviceID    string    `gorm:"uniqueIndex;not null;size:128"`
	DeviceName  string    `gorm:"not null;size:256"`
	DeviceType  string    `gorm:"not null;size:32"`
	FirstSeen   time.Time `gorm:"not null"`
	LastSeen    time.Time `gorm:"not null;index"`
	AppVersion  *string   `gorm:"size:32"`
	CanUpload   bool      `gorm:"default:true"`
	CanDownload bool      `gorm:"default:true"`
	CanDelete   bool      `gorm:"default:false"`
	IsActive    bool      `gorm:"default:true;index"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// Constants for sync status
const (
	SyncStatusPendingUpload    = "pending_upload"
	SyncStatusUploading        = "uploading"
	SyncStatusUploadedVerified = "uploaded_verified"
	SyncStatusDeletedLocal     = "deleted_local"
	SyncStatusBackupComplete   = "backup_complete"
	SyncStatusError            = "error"
)

// Constants for operation types
const (
	OperationTypeUpload      = "upload"
	OperationTypeVerify      = "verify"
	OperationTypeDeleteLocal = "delete_local"
	OperationTypeBackup      = "backup"
)

// Constants for device types
const (
	DeviceTypePhone  = "phone"
	DeviceTypeTV     = "tv"
	DeviceTypeTablet = "tablet"
)

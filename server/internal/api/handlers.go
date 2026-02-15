package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/private-media-ecosystem/internal/db"
)

type UploadMetadata struct {
	OriginalFilename string    `json:"original_filename" binding:"required"`
	TimestampTaken   time.Time `json:"timestamp_taken" binding:"required"`
	DestinationPath  string    `json:"destination_path" binding:"required"`
	DeviceID         string    `json:"device_id" binding:"required"`
	LocalHash        string    `json:"local_hash" binding:"required"`
	ExifData         *ExifData `json:"exif_data"`
}

type ExifData struct {
	Width        *int     `json:"width"`
	Height       *int     `json:"height"`
	Orientation  int      `json:"orientation"`
	CameraMake   *string  `json:"camera_make"`
	CameraModel  *string  `json:"camera_model"`
	GPSLatitude  *float64 `json:"gps_latitude"`
	GPSLongitude *float64 `json:"gps_longitude"`
	ISO          *int     `json:"iso"`
	Aperture     *float64 `json:"aperture"`
	FocalLength  *float64 `json:"focal_length"`
}

func (s *Server) handleUpload(c *gin.Context) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(100 << 20); err != nil { // 100 MB max
		errorResponse(c, http.StatusBadRequest, "invalid_form", "Failed to parse multipart form")
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "missing_file", "No file in request")
		return
	}
	defer file.Close()

	// Parse metadata
	var metadata UploadMetadata
	if err := c.ShouldBindJSON(&metadata); err != nil {
		// Try to get from form data instead
		metadataStr := c.PostForm("metadata")
		if metadataStr == "" {
			errorResponse(c, http.StatusBadRequest, "missing_metadata", "No metadata provided")
			return
		}
		// Parse JSON from form field
		// ... (omitted for brevity, would use json.Unmarshal)
	}

	// Validate device is registered
	device, err := s.services.Device.GetByDeviceID(metadata.DeviceID)
	if err != nil {
		errorResponse(c, http.StatusUnauthorized, "device_not_registered", "Device not registered")
		return
	}

	if !device.CanUpload {
		errorResponse(c, http.StatusForbidden, "upload_not_allowed", "Device does not have upload permission")
		return
	}

	// Check file size
	maxSize := int64(s.config.Upload.MaxSizeMB) * 1024 * 1024
	if header.Size > maxSize {
		errorResponse(c, http.StatusRequestEntityTooLarge, "file_too_large",
			fmt.Sprintf("File size exceeds maximum of %d MB", s.config.Upload.MaxSizeMB))
		return
	}

	// Process upload
	result, err := s.services.Upload.HandleUpload(file, metadata, device)
	if err != nil {
		// Handle specific errors
		switch err.Error() {
		case "duplicate_file":
			errorResponse(c, http.StatusConflict, "duplicate_file", "File with this hash already exists")
		case "hash_mismatch":
			errorResponse(c, http.StatusConflict, "hash_mismatch", "Client and server hashes do not match")
		case "storage_full":
			errorResponse(c, http.StatusInsufficientStorage, "storage_full", "Storage is full")
		default:
			errorResponse(c, http.StatusInternalServerError, "upload_failed", err.Error())
		}
		return
	}

	// Update device last seen
	s.services.Device.UpdateLastSeen(device.ID)

	c.JSON(http.StatusCreated, gin.H{
		"success":              true,
		"file_hash":            result.FileHash,
		"primary_path":         result.PrimaryPath,
		"file_size_bytes":      result.FileSizeBytes,
		"sync_status":          result.SyncStatus,
		"verification_matches": result.VerificationMatches,
		"uploaded_at":          result.UploadedAt,
	})
}

func (s *Server) handleUploadChunk(c *gin.Context) {
	// TODO: Implement chunked upload for large files
	errorResponse(c, http.StatusNotImplemented, "not_implemented", "Chunked upload not yet implemented")
}

func (s *Server) handleGetStatus(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		errorResponse(c, http.StatusBadRequest, "missing_hash", "Hash parameter is required")
		return
	}

	status, err := s.services.Sync.GetStatus(hash)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "not_found", "File not found")
		return
	}

	c.JSON(http.StatusOK, status)
}

func (s *Server) handleVerify(c *gin.Context) {
	hash := c.Param("hash")

	var req struct {
		LocalHash string `json:"local_hash" binding:"required"`
		DeviceID  string `json:"device_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Get file from database
	mediaFile, err := s.services.Sync.GetByHash(hash)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "not_found", "File not found")
		return
	}

	// Verify hash matches
	verified := mediaFile.FileHash == req.LocalHash

	if verified {
		// Update verification count
		s.services.Sync.UpdateVerification(mediaFile.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"verified":               verified,
		"server_hash":            mediaFile.FileHash,
		"hashes_match":           verified,
		"verification_timestamp": time.Now(),
		"safe_to_delete_local":   verified && mediaFile.SyncStatus == db.SyncStatusUploadedVerified,
	})
}

func (s *Server) handleDeleteLocal(c *gin.Context) {
	hash := c.Param("hash")

	var req struct {
		DeviceID          string    `json:"device_id" binding:"required"`
		DeletionTimestamp time.Time `json:"deletion_timestamp" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Get file from database
	mediaFile, err := s.services.Sync.GetByHash(hash)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "not_found", "File not found")
		return
	}

	// Update sync status to deleted_local
	if err := s.services.Sync.MarkDeletedLocal(mediaFile.ID, req.DeviceID); err != nil {
		errorResponse(c, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"sync_status": db.SyncStatusDeletedLocal,
		"message":     "Local copy marked as deleted. File remains on server.",
	})
}

func (s *Server) handleBrowse(c *gin.Context) {
	path := c.Param("path")
	if path == "" {
		path = "/"
	}

	// TODO: Implement filesystem browsing
	fullPath := filepath.Join(s.config.Storage.PrimaryPath, path)

	c.JSON(http.StatusOK, gin.H{
		"path":        path,
		"full_path":   fullPath,
		"total_items": 0,
		"directories": []interface{}{},
		"files":       []interface{}{},
		"note":        "Browsing implementation pending",
	})
}

func (s *Server) handleGetRoots(c *gin.Context) {
	// TODO: Get actual disk stats
	roots := []gin.H{
		{
			"path":            "/",
			"name":            "My Passport",
			"total_bytes":     1000000000000,
			"used_bytes":      456789012345,
			"available_bytes": 543210987655,
			"is_primary":      true,
		},
	}

	if s.config.Storage.BackupPath != "" {
		roots = append(roots, gin.H{
			"path":            "/backup",
			"name":            "Backup SSD",
			"total_bytes":     500000000000,
			"used_bytes":      123456789012,
			"available_bytes": 376543210988,
			"is_primary":      false,
		})
	}

	c.JSON(http.StatusOK, gin.H{"roots": roots})
}

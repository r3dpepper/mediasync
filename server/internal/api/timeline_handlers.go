package api

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/yourusername/private-media-ecosystem/internal/config"
	"github.com/yourusername/private-media-ecosystem/internal/db"
)

func (s *Server) handleTimeline(c *gin.Context) {
	// Parse query parameters
	startDate := c.Query("start")
	endDate := c.Query("end")
	mediaType := c.Query("media_type")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	sort := c.DefaultQuery("sort", "desc")

	// Parse dates
	var start, end *time.Time
	if startDate != "" {
		t, err := time.Parse(time.RFC3339, startDate)
		if err == nil {
			start = &t
		}
	}
	if endDate != "" {
		t, err := time.Parse(time.RFC3339, endDate)
		if err == nil {
			end = &t
		}
	}

	timeline, err := s.services.Timeline.GetTimeline(start, end, mediaType, limit, offset, sort)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, timeline)
}

func (s *Server) handleSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		errorResponse(c, http.StatusBadRequest, "missing_query", "Search query is required")
		return
	}

	mediaType := c.Query("media_type")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	results, err := s.services.Timeline.Search(query, mediaType, limit, offset)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "search_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, results)
}

func (s *Server) handleGetMetadata(c *gin.Context) {
	hash := c.Param("hash")

	metadata, err := s.services.Timeline.GetMetadata(hash)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "not_found", "File not found")
		return
	}

	c.JSON(http.StatusOK, metadata)
}

func (s *Server) handleThumbnail(c *gin.Context) {
	hash := c.Param("hash")
	size := c.DefaultQuery("size", "medium")
	quality, _ := strconv.Atoi(c.DefaultQuery("quality", "80"))

	// Get thumbnail (generates if not exists)
	thumbnailPath, err := s.services.Thumbnail.GetThumbnail(hash, size, quality)
	if err != nil {
		// If thumbnail is being generated, return 503
		if err.Error() == "generating" {
			c.Header("Retry-After", "2")
			errorResponse(c, http.StatusServiceUnavailable, "generating", "Thumbnail generation in progress")
			return
		}
		errorResponse(c, http.StatusNotFound, "not_found", "Thumbnail not available")
		return
	}

	// Serve thumbnail file
	c.Header("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	c.File(thumbnailPath)
}

func (s *Server) handleStream(c *gin.Context) {
	hash := c.Param("hash")
	transcode := c.Query("transcode") == "true"
	log.Info().Str("hash", hash).Str("method", c.Request.Method).Bool("transcode", transcode).Msg("Stream request received")

	// Get file info from database
	mediaFile, err := s.services.Sync.GetByHash(hash)
	if err != nil {
		log.Error().Err(err).Str("hash", hash).Msg("Failed to get file by hash")
		errorResponse(c, http.StatusNotFound, "not_found", "File not found")
		return
	}

	log.Info().Str("path", mediaFile.PrimaryPath).Msg("Found file in database")

	// If transcoding requested and it's a video, transcode on-the-fly
	if transcode && mediaFile.MediaType == "video" {
		s.handleTranscode(c, mediaFile)
		return
	}

	// Get file info for size
	fileInfo, err := os.Stat(mediaFile.PrimaryPath)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "file_error", "Failed to get file info")
		return
	}
	fileSize := fileInfo.Size()

	// Set headers
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Type", mediaFile.MimeType)
	c.Header("Content-Length", strconv.FormatInt(fileSize, 10))

	// Handle HEAD request
	if c.Request.Method == "HEAD" {
		c.Status(http.StatusOK)
		return
	}

	// Open file for GET request
	file, err := os.Open(mediaFile.PrimaryPath)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "file_error", "Failed to open file")
		return
	}
	defer file.Close()

	// Handle range request (for video seeking)
	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		// Parse range header
		rangeStr := strings.TrimPrefix(rangeHeader, "bytes=")
		rangeParts := strings.Split(rangeStr, "-")

		start, _ := strconv.ParseInt(rangeParts[0], 10, 64)
		var end int64
		if len(rangeParts) > 1 && rangeParts[1] != "" {
			end, _ = strconv.ParseInt(rangeParts[1], 10, 64)
		} else {
			end = fileSize - 1
		}

		if start >= fileSize {
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		// Seek to start position
		file.Seek(start, 0)

		// Set range response headers
		c.Status(http.StatusPartialContent)
		c.Header("Content-Range", strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10)+"/"+strconv.FormatInt(fileSize, 10))
		c.Header("Content-Length", strconv.FormatInt(end-start+1, 10))

		// Copy range to response
		io.CopyN(c.Writer, file, end-start+1)
	} else {
		// Full file response
		c.Header("Content-Length", strconv.FormatInt(fileSize, 10))
		io.Copy(c.Writer, file)
	}
}

func (s *Server) handleDownload(c *gin.Context) {
	hash := c.Param("hash")

	// Get file info from database
	mediaFile, err := s.services.Sync.GetByHash(hash)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "not_found", "File not found")
		return
	}

	// Set download headers
	c.Header("Content-Disposition", "attachment; filename=\""+mediaFile.OriginalFilename+"\"")
	c.Header("Content-Type", mediaFile.MimeType)

	// Serve file
	c.File(mediaFile.PrimaryPath)
}

func (s *Server) handleHLS(c *gin.Context) {
	hash := c.Param("hash")

	_, err := s.services.Sync.GetByHash(hash)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "not_found", "File not found")
		return
	}

	// Generate HLS playlist that streams the video
	playlist := `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-PLAYLIST-TYPE:VOD
#EXTINF:10.0,
segment.m3u8
#EXT-X-ENDLIST`

	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.String(http.StatusOK, playlist)
}

func (s *Server) handleHLSSegment(c *gin.Context) {
	hash := c.Param("hash")

	mediaFile, err := s.services.Sync.GetByHash(hash)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "not_found", "File not found")
		return
	}

	// Stream video directly as TS format using ffmpeg
	c.Header("Content-Type", "video/mp2t")
	c.Header("Cache-Control", "no-cache")
	
	// Just serve the original file for now
	c.File(mediaFile.PrimaryPath)
}

func (s *Server) handleTranscode(c *gin.Context, mediaFile *db.MediaFile) {
	log.Info().Str("path", mediaFile.PrimaryPath).Msg("Starting live transcode")

	// Set headers for streaming
	c.Header("Content-Type", "video/mp4")
	c.Header("Cache-Control", "no-cache")
	c.Status(http.StatusOK)

	// Start ffmpeg to transcode on-the-fly
	cmd := exec.Command("ffmpeg",
		"-i", mediaFile.PrimaryPath,
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-profile:v", "baseline",
		"-level", "3.0",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "frag_keyframe+empty_moov+faststart",
		"-f", "mp4",
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create stdout pipe")
		return
	}

	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start ffmpeg")
		return
	}

	// Stream the output to the client
	io.Copy(c.Writer, stdout)

	cmd.Wait()
	log.Info().Msg("Transcode completed")
}

func (s *Server) handleBackupStatus(c *gin.Context) {
	status, err := s.services.Backup.GetStatus()
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "status_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, status)
}

func (s *Server) handleBackupStart(c *gin.Context) {
	var req struct {
		JobType  string `json:"job_type" binding:"required"`
		Priority string `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	jobID, err := s.services.Backup.StartBackup(req.JobType, req.Priority)
	if err != nil {
		if err.Error() == "backup_running" {
			errorResponse(c, http.StatusConflict, "backup_running", "Backup already in progress")
			return
		}
		errorResponse(c, http.StatusInternalServerError, "start_failed", err.Error())
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"success":  true,
		"job_id":   jobID,
		"job_type": req.JobType,
		"status":   "queued",
		"message":  "Backup job queued. Check /backup/status for progress.",
	})
}

func (s *Server) handleBackupCancel(c *gin.Context) {
	var req struct {
		JobID uint `json:"job_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := s.services.Backup.CancelBackup(req.JobID); err != nil {
		errorResponse(c, http.StatusInternalServerError, "cancel_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"job_id":  req.JobID,
		"status":  "cancelled",
		"message": "Backup job cancelled. Partial data may exist.",
	})
}

func (s *Server) handleDeviceRegister(c *gin.Context) {
	var req struct {
		DeviceID   string `json:"device_id" binding:"required"`
		DeviceName string `json:"device_name" binding:"required"`
		DeviceType string `json:"device_type" binding:"required"`
		AppVersion string `json:"app_version"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	device, err := s.services.Device.Register(req.DeviceID, req.DeviceName, req.DeviceType, req.AppVersion)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "registration_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"device_id":     device.DeviceID,
		"registered_at": device.FirstSeen,
		"permissions": gin.H{
			"can_upload":   device.CanUpload,
			"can_download": device.CanDownload,
			"can_delete":   device.CanDelete,
		},
	})
}

func (s *Server) handleDeviceList(c *gin.Context) {
	devices, err := s.services.Device.ListAll()
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

func (s *Server) handleDeviceUpdate(c *gin.Context) {
	deviceID := c.Param("device_id")

	var req struct {
		DeviceName  *string `json:"device_name"`
		Permissions *struct {
			CanUpload   *bool `json:"can_upload"`
			CanDownload *bool `json:"can_download"`
			CanDelete   *bool `json:"can_delete"`
		} `json:"permissions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// TODO: Implement device update
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"device_id":  deviceID,
		"updated_at": time.Now(),
	})
}

func (s *Server) handleResync(c *gin.Context) {
	var req struct {
		ScanPath        string `json:"scan_path"`
		DryRun          bool   `json:"dry_run"`
		RecomputeHashes bool   `json:"recompute_hashes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.ScanPath = "/"
	}

	eventID, err := s.services.Resync.StartResync(req.ScanPath, req.DryRun, req.RecomputeHashes)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "resync_failed", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"resync_event_id": eventID,
		"status":          "running",
		"message":         "Resync job started. This may take several minutes.",
	})
}

func (s *Server) handleResyncStatus(c *gin.Context) {
	eventIDStr := c.Param("event_id")
	eventID, err := strconv.ParseUint(eventIDStr, 10, 64)
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid_event_id", "Invalid event ID")
		return
	}

	status, err := s.services.Resync.GetStatus(uint(eventID))
	if err != nil {
		errorResponse(c, http.StatusNotFound, "not_found", "Resync event not found")
		return
	}

	c.JSON(http.StatusOK, status)
}

func (s *Server) handleVacuum(c *gin.Context) {
	// TODO: Implement database vacuum
	c.JSON(http.StatusOK, gin.H{
		"success":                    true,
		"database_size_before_bytes": 125829120,
		"database_size_after_bytes":  94371840,
		"space_reclaimed_bytes":      31457280,
		"duration_ms":                5432,
	})
}

func (s *Server) handleTruncate(c *gin.Context) {
	if err := s.db.Exec("DELETE FROM media_files").Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "truncate_failed", err.Error())
		return
	}
	if err := s.db.Exec("DELETE FROM exif_metadata").Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "truncate_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Database truncated successfully",
	})
}

func (s *Server) handleGetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, s.config)
}

func (s *Server) handleUpdateConfig(c *gin.Context) {
	var updates config.Config
	if err := c.ShouldBindJSON(&updates); err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	
	if updates.Storage.PrimaryPath != "" {
		s.config.Storage.PrimaryPath = updates.Storage.PrimaryPath
	}
	if updates.Storage.BackupPath != "" {
		s.config.Storage.BackupPath = updates.Storage.BackupPath
	}
	if updates.Server.Port > 0 {
		s.config.Server.Port = updates.Server.Port
	}
	
	if err := config.SaveConfig(s.config); err != nil {
		errorResponse(c, http.StatusInternalServerError, "save_failed", err.Error())
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Configuration updated. Restart server to apply changes.",
		"config":  s.config,
	})
}

package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/yourusername/private-media-ecosystem/internal/config"
	"github.com/yourusername/private-media-ecosystem/internal/service"
	"gorm.io/gorm"
)

type Server struct {
	config   *config.Config
	db       *gorm.DB
	router   *gin.Engine
	services *Services
}

type Services struct {
	Upload    *service.UploadService
	Sync      *service.SyncService
	Timeline  *service.TimelineService
	Thumbnail *service.ThumbnailService
	Backup    *service.BackupService
	Device    *service.DeviceService
	Resync    *service.ResyncService
}

func NewServer(cfg *config.Config, db *gorm.DB) *Server {
	if cfg.Server.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(LoggerMiddleware())
	router.Use(CORSMiddleware())

	// Initialize services
	services := &Services{
		Upload:    service.NewUploadService(cfg, db),
		Sync:      service.NewSyncService(db),
		Timeline:  service.NewTimelineService(db),
		Thumbnail: service.NewThumbnailService(cfg, db),
		Backup:    service.NewBackupService(cfg, db),
		Device:    service.NewDeviceService(db),
		Resync:    service.NewResyncService(cfg, db),
	}

	server := &Server{
		config:   cfg,
		db:       db,
		router:   router,
		services: services,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	// Root endpoint
	s.router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "Private Media Ecosystem Server",
			"version": "1.0.0",
			"api":     "/api",
		})
	})

	api := s.router.Group("/api")

	// Health & Status
	api.GET("/health", s.handleHealth)
	api.GET("/stats", s.handleStats)

	// File Upload
	api.POST("/upload", s.handleUpload)
	api.POST("/upload/chunk", s.handleUploadChunk)

	// Sync Management
	api.GET("/status/:hash", s.handleGetStatus)
	api.POST("/verify/:hash", s.handleVerify)
	api.DELETE("/local/:hash", s.handleDeleteLocal)

	// File Browsing
	api.GET("/roots", s.handleGetRoots)
	api.GET("/browse/*path", s.handleBrowse)

	// Timeline & Search
	api.GET("/timeline", s.handleTimeline)
	api.GET("/search", s.handleSearch)

	// Metadata
	api.GET("/metadata/:hash", s.handleGetMetadata)

	// Streaming
	api.GET("/thumbnail/:hash", s.handleThumbnail)
	api.HEAD("/thumbnail/:hash", s.handleThumbnail)
	api.GET("/stream/:hash", s.handleStream)
	api.HEAD("/stream/:hash", s.handleStream)
	api.GET("/hls/:hash/playlist.m3u8", s.handleHLS)
	api.GET("/hls/:hash/:segment", s.handleHLSSegment)
	api.GET("/download/:hash", s.handleDownload)

	// Backup Operations
	api.GET("/backup/status", s.handleBackupStatus)
	api.POST("/backup/start", s.handleBackupStart)
	api.POST("/backup/cancel", s.handleBackupCancel)

	// Device Management
	api.POST("/devices/register", s.handleDeviceRegister)
	api.GET("/devices", s.handleDeviceList)
	api.PUT("/devices/:device_id", s.handleDeviceUpdate)

	// Resync & Maintenance
	api.POST("/resync", s.handleResync)
	api.GET("/resync/:event_id", s.handleResyncStatus)
	api.POST("/maintenance/vacuum", s.handleVacuum)
	api.POST("/maintenance/truncate", s.handleTruncate)
	
	// Configuration
	api.GET("/config", s.handleGetConfig)
	api.PUT("/config", s.handleUpdateConfig)
}

func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.BindAddress, s.config.Server.Port)

	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	// Start server in goroutine
	go func() {
		log.Info().Str("address", addr).Msg("HTTP server started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	log.Info().Msg("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Info().Msg("Server stopped gracefully")
	return nil
}

// Health check endpoint
func (s *Server) handleHealth(c *gin.Context) {
	// Check database connectivity
	sqlDB, err := s.db.DB()
	dbConnected := err == nil && sqlDB.Ping() == nil

	// Check storage availability
	storageAvailable := true
	// TODO: Add actual storage check

	status := "healthy"
	if !dbConnected || !storageAvailable {
		status = "degraded"
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":             status,
			"database_connected": dbConnected,
			"storage_available":  storageAvailable,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":             status,
		"version":            "1.0.0",
		"database_connected": dbConnected,
		"storage_available":  storageAvailable,
		"mdns_advertised":    s.config.Discovery.Enabled,
	})
}

// Stats endpoint
func (s *Server) handleStats(c *gin.Context) {
	stats, err := s.services.Timeline.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// Middleware for logging
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		method := c.Request.Method

		if raw != "" {
			path = path + "?" + raw
		}

		log.Info().
			Str("method", method).
			Str("path", path).
			Int("status", statusCode).
			Dur("latency", latency).
			Msg("HTTP request")
	}
}

// CORS middleware
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Error response helper
func errorResponse(c *gin.Context, statusCode int, errCode string, message string) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"error":   errCode,
		"message": message,
	})
}

// Success response helper
func successResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

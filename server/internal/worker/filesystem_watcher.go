package worker

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
	"github.com/yourusername/private-media-ecosystem/internal/config"
	"github.com/yourusername/private-media-ecosystem/internal/service"
	"gorm.io/gorm"
)

type FilesystemWatcher struct {
	config   *config.Config
	db       *gorm.DB
	resync   *service.ResyncService
	watcher  *fsnotify.Watcher
	stopChan chan struct{}
}

func NewFilesystemWatcher(cfg *config.Config, db *gorm.DB) *FilesystemWatcher {
	return &FilesystemWatcher{
		config:   cfg,
		db:       db,
		resync:   service.NewResyncService(cfg, db),
		stopChan: make(chan struct{}),
	}
}

func (w *FilesystemWatcher) Name() string {
	return "FilesystemWatcher"
}

func (w *FilesystemWatcher) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.watcher = watcher
	defer w.watcher.Close()

	// Add primary storage path to watch
	if err := w.addRecursive(w.config.Storage.PrimaryPath); err != nil {
		log.Error().Err(err).Msg("Failed to watch primary storage")
		return err
	}

	log.Info().Str("path", w.config.Storage.PrimaryPath).Msg("Filesystem watcher started")

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-w.stopChan:
			return nil

		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			log.Error().Err(err).Msg("Filesystem watcher error")
		}
	}
}

func (w *FilesystemWatcher) Stop() error {
	close(w.stopChan)
	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}

func (w *FilesystemWatcher) handleEvent(event fsnotify.Event) {
	// Filter out non-media files
	if !w.isMediaFile(event.Name) {
		return
	}

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		log.Debug().Str("file", event.Name).Msg("File created")
		// New file detected - could trigger import

	case event.Op&fsnotify.Write == fsnotify.Write:
		log.Debug().Str("file", event.Name).Msg("File modified")
		// File modified - might need to recompute hash

	case event.Op&fsnotify.Remove == fsnotify.Remove:
		log.Debug().Str("file", event.Name).Msg("File removed")
		// File deleted - mark as orphaned in database

	case event.Op&fsnotify.Rename == fsnotify.Rename:
		log.Info().Str("file", event.Name).Msg("File moved/renamed")
		// File moved - trigger reconciliation
		w.resync.QueueReconciliation(event.Name)

	case event.Op&fsnotify.Chmod == fsnotify.Chmod:
		// Ignore permission changes
	}
}

func (w *FilesystemWatcher) addRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip hidden directories
			if len(info.Name()) > 0 && info.Name()[0] == '.' {
				return filepath.SkipDir
			}

			if err := w.watcher.Add(path); err != nil {
				log.Warn().Err(err).Str("path", path).Msg("Failed to watch directory")
			}
		}

		return nil
	})
}

func (w *FilesystemWatcher) isMediaFile(path string) bool {
	ext := filepath.Ext(path)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".tiff", ".tif", ".bmp",
		".mp4", ".mov", ".avi", ".mkv", ".webm", ".m4v", ".flv":
		return true
	}
	return false
}

package worker

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/yourusername/private-media-ecosystem/internal/config"
	"gorm.io/gorm"
)

type Manager struct {
	config  *config.Config
	db      *gorm.DB
	workers []Worker
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

type Worker interface {
	Start(ctx context.Context) error
	Stop() error
	Name() string
}

func NewManager(cfg *config.Config, db *gorm.DB) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		config: cfg,
		db:     db,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize workers
	manager.workers = []Worker{
		NewThumbnailWorker(cfg, db),
		// NewBackupWorker(cfg, db),
		// NewFilesystemWatcher(cfg, db),
	}

	return manager
}

func (m *Manager) Start() {
	log.Info().Msg("Starting background workers")

	for _, worker := range m.workers {
		m.wg.Add(1)
		go func(w Worker) {
			defer m.wg.Done()

			log.Info().Str("worker", w.Name()).Msg("Starting worker")
			if err := w.Start(m.ctx); err != nil {
				log.Error().
					Err(err).
					Str("worker", w.Name()).
					Msg("Worker error")
			}
		}(worker)
	}
}

func (m *Manager) Stop() {
	log.Info().Msg("Stopping background workers")
	m.cancel()

	for _, worker := range m.workers {
		if err := worker.Stop(); err != nil {
			log.Error().
				Err(err).
				Str("worker", worker.Name()).
				Msg("Worker stop error")
		}
	}

	m.wg.Wait()
	log.Info().Msg("All workers stopped")
}

// ThumbnailWorker generates thumbnails asynchronously
type ThumbnailWorker struct {
	config *config.Config
	db     *gorm.DB
	queue  chan string
}

func NewThumbnailWorker(cfg *config.Config, db *gorm.DB) *ThumbnailWorker {
	return &ThumbnailWorker{
		config: cfg,
		db:     db,
		queue:  make(chan string, 100),
	}
}

func (w *ThumbnailWorker) Name() string {
	return "ThumbnailWorker"
}

func (w *ThumbnailWorker) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case hash := <-w.queue:
			log.Debug().Str("hash", hash).Msg("Generating thumbnail")
			// TODO: Implement actual thumbnail generation
		}
	}
}

func (w *ThumbnailWorker) Stop() error {
	close(w.queue)
	return nil
}

func (w *ThumbnailWorker) Enqueue(hash string) {
	select {
	case w.queue <- hash:
	default:
		log.Warn().Str("hash", hash).Msg("Thumbnail queue full, dropping request")
	}
}

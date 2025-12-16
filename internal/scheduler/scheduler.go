package scheduler

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jung/doc2git/internal/config"
)

// SyncFunc is the function signature for sync operations
type SyncFunc func(ctx context.Context, cfg *config.Config) error

// Scheduler manages periodic sync operations with graceful shutdown
type Scheduler struct {
	cfg       *config.Config
	syncFunc  SyncFunc
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.Mutex

	// Statistics
	lastSyncTime  time.Time
	lastSyncError error
	syncCount     int64
	startTime     time.Time
}

// NewScheduler creates a new scheduler instance
func NewScheduler(cfg *config.Config, syncFunc SyncFunc) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		cfg:       cfg,
		syncFunc:  syncFunc,
		ctx:       ctx,
		cancel:    cancel,
		startTime: time.Now(),
	}
}

// Start begins the scheduler loop
func (s *Scheduler) Start() {
	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		s.Shutdown()
	}()

	// Initial sync
	log.Println("Starting initial sync...")
	s.runSyncSafe()

	// Check if one-shot mode (SyncInterval <= 0)
	if s.cfg.SyncInterval <= 0 {
		log.Println("One-shot mode: SYNC_INTERVAL not set or <= 0, exiting after initial sync")
		return
	}

	// Periodic sync
	log.Printf("Scheduler started. Next sync in %v", s.cfg.SyncInterval)
	ticker := time.NewTicker(s.cfg.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			log.Println("Scheduler stopped")
			return
		case <-ticker.C:
			log.Println("Starting scheduled sync...")
			s.runSyncSafe()
			log.Printf("Next sync in %v", s.cfg.SyncInterval)
		}
	}
}

// runSyncSafe executes sync with mutex protection to prevent concurrent runs
func (s *Scheduler) runSyncSafe() {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		log.Println("Sync already in progress, skipping...")
		return
	}
	s.isRunning = true
	s.mu.Unlock()

	s.wg.Add(1)
	defer func() {
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
		s.wg.Done()
	}()

	startTime := time.Now()
	err := s.syncFunc(s.ctx, s.cfg)

	s.mu.Lock()
	s.lastSyncTime = time.Now()
	s.lastSyncError = err
	s.syncCount++
	s.mu.Unlock()

	if err != nil {
		log.Printf("Sync failed: %v (duration: %v)", err, time.Since(startTime))
	} else {
		log.Printf("Sync completed successfully (duration: %v)", time.Since(startTime))
	}
}

// Shutdown initiates graceful shutdown
func (s *Scheduler) Shutdown() {
	log.Println("Initiating graceful shutdown...")
	s.cancel()

	// Wait for running sync to complete (with timeout)
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Graceful shutdown completed")
	case <-time.After(30 * time.Second):
		log.Println("Shutdown timeout, forcing exit")
	}
}

// Stats returns current scheduler statistics
func (s *Scheduler) Stats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()

	var lastError string
	if s.lastSyncError != nil {
		lastError = s.lastSyncError.Error()
	}

	return Stats{
		LastSyncTime:  s.lastSyncTime,
		LastSyncError: lastError,
		SyncCount:     s.syncCount,
		IsRunning:     s.isRunning,
		Uptime:        time.Since(s.startTime),
	}
}

// Stats holds scheduler statistics
type Stats struct {
	LastSyncTime  time.Time
	LastSyncError string
	SyncCount     int64
	IsRunning     bool
	Uptime        time.Duration
}

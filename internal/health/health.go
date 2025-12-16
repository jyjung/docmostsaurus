package health

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status represents the health check response
type Status struct {
	Status        string    `json:"status"`
	LastSync      time.Time `json:"last_sync,omitempty"`
	LastError     string    `json:"last_error,omitempty"`
	SyncCount     int64     `json:"sync_count"`
	IsRunning     bool      `json:"is_running"`
	Uptime        string    `json:"uptime"`
	NextSync      string    `json:"next_sync,omitempty"`
	SyncInterval  string    `json:"sync_interval,omitempty"`
}

// Checker maintains health check state
type Checker struct {
	mu            sync.RWMutex
	lastSyncTime  time.Time
	lastSyncError error
	syncCount     int64
	isRunning     bool
	startTime     time.Time
	syncInterval  time.Duration
}

// NewChecker creates a new health checker
func NewChecker(syncInterval time.Duration) *Checker {
	return &Checker{
		startTime:    time.Now(),
		syncInterval: syncInterval,
	}
}

// UpdateSyncStatus updates the sync status
func (c *Checker) UpdateSyncStatus(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastSyncTime = time.Now()
	c.lastSyncError = err
	c.syncCount++
}

// SetRunning sets the running state
func (c *Checker) SetRunning(running bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isRunning = running
}

// GetStatus returns the current health status
func (c *Checker) GetStatus() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := Status{
		Status:       "healthy",
		LastSync:     c.lastSyncTime,
		SyncCount:    c.syncCount,
		IsRunning:    c.isRunning,
		Uptime:       time.Since(c.startTime).Round(time.Second).String(),
		SyncInterval: c.syncInterval.String(),
	}

	if c.lastSyncError != nil {
		status.Status = "degraded"
		status.LastError = c.lastSyncError.Error()
	}

	// Calculate next sync time
	if !c.lastSyncTime.IsZero() && c.syncInterval > 0 {
		nextSync := c.lastSyncTime.Add(c.syncInterval)
		if nextSync.After(time.Now()) {
			status.NextSync = time.Until(nextSync).Round(time.Second).String()
		}
	}

	// Mark as unhealthy if last sync was too long ago (2x interval)
	if !c.lastSyncTime.IsZero() && c.syncInterval > 0 {
		if time.Since(c.lastSyncTime) > 2*c.syncInterval {
			status.Status = "unhealthy"
		}
	}

	return status
}

// Handler returns an HTTP handler for health checks
func (c *Checker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := c.GetStatus()

		w.Header().Set("Content-Type", "application/json")

		if status.Status == "unhealthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(status)
	}
}

// Server manages the health check HTTP server
type Server struct {
	checker *Checker
	addr    string
	server  *http.Server
}

// NewServer creates a new health check server
func NewServer(checker *Checker, addr string) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", checker.Handler())
	mux.HandleFunc("/healthz", checker.Handler()) // Kubernetes compatibility
	mux.HandleFunc("/ready", checker.Handler())   // Readiness probe

	return &Server{
		checker: checker,
		addr:    addr,
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

// Start starts the health check server in the background
func (s *Server) Start() {
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Just log, don't fail the main process
			println("Health check server error:", err.Error())
		}
	}()
}

// Stop stops the health check server
func (s *Server) Stop() error {
	return s.server.Close()
}

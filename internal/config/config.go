package config

import (
	"os"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	// Docmost configuration
	DocmostBaseURL  string
	DocmostEmail    string
	DocmostPassword string

	// Sync settings
	SyncInterval time.Duration
	OutputDir    string

	// HTTP server settings
	HTTPPort string

	// Git settings
	GitRepoPath  string
	GitBranch    string
	AutoPush     bool
	GitRemoteURL string
	GitUsername  string
	GitPassword  string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		DocmostBaseURL:  getEnv("DOCMOST_BASE_URL", "http://localhost:3000"),
		DocmostEmail:    getEnv("DOCMOST_EMAIL", ""),
		DocmostPassword: getEnv("DOCMOST_PASSWORD", ""),
		OutputDir:       getEnv("OUTPUT_DIR", "./output"),
		HTTPPort:    getEnv("HTTP_PORT", ":8080"),
		GitRepoPath:     getEnv("GIT_REPO_PATH", "./docusaurus-docs"),
		GitBranch:       getEnv("GIT_BRANCH", "main"),
		AutoPush:        getEnv("AUTO_PUSH", "false") == "true",
		GitRemoteURL:    getEnv("GIT_REMOTE_URL", ""),
		GitUsername:     getEnv("GIT_USERNAME", ""),
		GitPassword:     getEnv("GIT_PASSWORD", ""),
	}

	// Parse sync interval
	// If SYNC_INTERVAL is empty or not set, run once and exit (SyncInterval = 0)
	intervalStr := os.Getenv("SYNC_INTERVAL")
	if intervalStr == "" {
		cfg.SyncInterval = 0
	} else {
		interval, err := time.ParseDuration(intervalStr)
		if err != nil {
			interval = time.Hour
		}
		cfg.SyncInterval = interval
	}

	return cfg, nil
}

// Validate checks if required configuration is present
func (c *Config) Validate() error {
	if c.DocmostBaseURL == "" {
		return ErrMissingBaseURL
	}
	if c.DocmostEmail == "" {
		return ErrMissingEmail
	}
	if c.DocmostPassword == "" {
		return ErrMissingPassword
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Custom errors
type ConfigError string

func (e ConfigError) Error() string {
	return string(e)
}

const (
	ErrMissingBaseURL  ConfigError = "DOCMOST_BASE_URL is required"
	ErrMissingEmail    ConfigError = "DOCMOST_EMAIL is required"
	ErrMissingPassword ConfigError = "DOCMOST_PASSWORD is required"
)

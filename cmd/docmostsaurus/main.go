package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jung/doc2git/internal/config"
	"github.com/jung/doc2git/internal/docmost"
	"github.com/jung/doc2git/internal/health"
	"github.com/jung/doc2git/internal/lock"
	"github.com/jung/doc2git/internal/postprocess"
	"github.com/jung/doc2git/internal/scheduler"
)

func main() {
	// Parse command line flags
	outputDir := flag.String("output", "", "Output directory for exported markdown files (overrides OUTPUT_DIR env)")
	oneShot := flag.Bool("once", false, "Run once and exit (ignore SYNC_INTERVAL)")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Override output directory if specified via flag
	if *outputDir != "" {
		cfg.OutputDir = *outputDir
	}

	// Override sync interval if one-shot mode
	if *oneShot {
		cfg.SyncInterval = 0
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		fmt.Fprintln(os.Stderr, "\nRequired environment variables:")
		fmt.Fprintln(os.Stderr, "  DOCMOST_BASE_URL  - Docmost server URL (e.g., http://192.168.31.101:3456)")
		fmt.Fprintln(os.Stderr, "  DOCMOST_EMAIL     - Docmost login email")
		fmt.Fprintln(os.Stderr, "  DOCMOST_PASSWORD  - Docmost login password")
		fmt.Fprintln(os.Stderr, "\nOptional environment variables:")
		fmt.Fprintln(os.Stderr, "  OUTPUT_DIR        - Output directory (default: ./output)")
		fmt.Fprintln(os.Stderr, "  SYNC_INTERVAL     - Sync interval (default: 1h, e.g., 30m, 2h)")
		fmt.Fprintln(os.Stderr, "  HTTP_PORT         - HTTP server port (default: :8080)")
		os.Exit(1)
	}

	// Acquire file lock to prevent concurrent instances
	fileLock := lock.NewFileLock()
	if err := fileLock.TryLock(); err != nil {
		log.Fatalf("Failed to acquire lock: %v", err)
	}
	defer fileLock.Unlock()

	log.Println("=== Docmost Markdown Exporter ===")
	log.Printf("Server: %s", cfg.DocmostBaseURL)
	log.Printf("Output: %s", cfg.OutputDir)
	if cfg.SyncInterval > 0 {
		log.Printf("Sync Interval: %v", cfg.SyncInterval)
	} else {
		log.Println("Mode: One-shot (run once and exit)")
	}

	// Start HTTP server (health check + future API endpoints)
	healthChecker := health.NewChecker(cfg.SyncInterval)
	healthServer := health.NewServer(healthChecker, cfg.HTTPPort)
	healthServer.Start()
	log.Printf("HTTP server started on %s", cfg.HTTPPort)
	defer healthServer.Stop()

	// Create scheduler with sync function
	sched := scheduler.NewScheduler(cfg, func(ctx context.Context, cfg *config.Config) error {
		healthChecker.SetRunning(true)
		defer healthChecker.SetRunning(false)

		err := runSync(ctx, cfg)
		healthChecker.UpdateSyncStatus(err)
		return err
	})

	// Start scheduler (blocks until shutdown)
	sched.Start()

	log.Println("=== Shutdown Complete ===")
}

// runSync performs a single sync operation
func runSync(ctx context.Context, cfg *config.Config) error {
	// Check for cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Create Docmost client
	client, err := docmost.NewClient(cfg.DocmostBaseURL, cfg.DocmostEmail, cfg.DocmostPassword)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	// Login
	log.Println("Logging in to Docmost...")
	if err := client.Login(); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	log.Println("Login successful!")

	// Export all spaces
	log.Println("Exporting all spaces...")
	exportedSpaces, err := client.ExportAllSpaces()
	if err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	if len(exportedSpaces) == 0 {
		log.Println("No spaces found to export.")
		return nil
	}

	// Save exported files to output directory
	totalFiles := 0
	for _, exported := range exportedSpaces {
		// Check for cancellation between spaces
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		spaceName := sanitizeDirName(exported.Space.Name)
		spaceDir := filepath.Join(cfg.OutputDir, spaceName)
		spaceDirTemp := filepath.Join(cfg.OutputDir, spaceName+"_temp")
		spaceDirOld := filepath.Join(cfg.OutputDir, spaceName+"_old")

		// Clean up any existing temp directory from previous failed runs
		cleanupTempDir(spaceDirTemp)

		// Create temp directory for atomic swap
		if err := os.MkdirAll(spaceDirTemp, 0755); err != nil {
			log.Printf("Error creating temp directory %s: %v", spaceDirTemp, err)
			continue
		}

		// Track if any error occurred during processing
		var processingError error

		// Write files to temp directory
		for filename, content := range exported.Files {
			filePath := filepath.Join(spaceDirTemp, filename)

			// Create parent directories if needed
			parentDir := filepath.Dir(filePath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				log.Printf("Error creating directory %s: %v", parentDir, err)
				continue
			}

			// Write file
			if err := os.WriteFile(filePath, content, 0644); err != nil {
				log.Printf("Error writing file %s: %v", filePath, err)
				continue
			}

			totalFiles++
		}

		// Save metadata JSON file to temp directory
		if exported.Metadata != nil {
			metaPath := filepath.Join(spaceDirTemp, "_metadata.json")
			metaData, err := json.MarshalIndent(exported.Metadata, "", "  ")
			if err != nil {
				log.Printf("Error marshaling metadata for %s: %v", exported.Space.Name, err)
				processingError = err
			} else {
				if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
					log.Printf("Error writing metadata file %s: %v", metaPath, err)
					processingError = err
				} else {
					log.Printf("Space '%s': metadata saved to %s", exported.Space.Name, metaPath)
				}
			}
		}

		// Check for errors before post-processing
		if processingError != nil {
			log.Printf("Skipping space '%s' due to errors, cleaning up temp directory", exported.Space.Name)
			cleanupTempDir(spaceDirTemp)
			continue
		}

		log.Printf("Space '%s': %d files saved to %s", exported.Space.Name, len(exported.Files), spaceDirTemp)

		// Post-process: Fix files/folders split due to "/" in title and update _metadata.json
		log.Printf("Post-processing: Fixing slash-split files and titles in %s...", spaceDirTemp)
		if err := postprocess.FixSlashInTitles(spaceDirTemp); err != nil {
			log.Printf("Warning: failed to fix slash-split files: %v", err)
		}

		// Post-process: Remove orphaned files not in _metadata.json
		log.Printf("Post-processing: Removing orphaned files in %s...", spaceDirTemp)
		if err := postprocess.RemoveOrphanedFiles(spaceDirTemp); err != nil {
			log.Printf("Warning: failed to remove orphaned files: %v", err)
		}

		// Post-process: Wrap placeholders with backticks (before frontmatter)
		log.Printf("Post-processing: Wrapping placeholders with backticks in %s...", spaceDirTemp)
		if err := postprocess.WrapPlaceholdersWithBackticks(spaceDirTemp); err != nil {
			log.Printf("Warning: failed to wrap placeholders: %v", err)
		}

		// Post-process: Wrap angle brackets with backticks (before frontmatter)
		log.Printf("Post-processing: Wrapping angle brackets with backticks in %s...", spaceDirTemp)
		if err := postprocess.WrapAngleBracketsWithBackticks(spaceDirTemp); err != nil {
			log.Printf("Warning: failed to wrap angle brackets: %v", err)
		}

		// Post-process: Wrap raw HTML (like tables) with code blocks
		log.Printf("Post-processing: Wrapping raw HTML with code blocks in %s...", spaceDirTemp)
		if err := postprocess.WrapRawHTMLWithCodeBlock(spaceDirTemp); err != nil {
			log.Printf("Warning: failed to wrap raw HTML: %v", err)
		}

		// Post-process: Merge files that were incorrectly split due to "/" in title (BEFORE romanization)
		// This handles Korean filenames like "Security365 환경 인증/인가 관련 공통 에러 페이지.md"
		log.Printf("Post-processing: Merging slash-split files (before romanization) in %s...", spaceDirTemp)
		if err := postprocess.MergeSlashSplitFiles(spaceDirTemp); err != nil {
			log.Printf("Warning: failed to merge slash-split files: %v", err)
		}

		// Post-process: Romanize Korean filenames and add frontmatter
		log.Printf("Post-processing: Romanizing Korean filenames in %s...", spaceDirTemp)
		results, err := postprocess.RomanizeSpace(spaceDirTemp)
		if err != nil {
			log.Printf("Warning: failed to romanize space %s: %v", exported.Space.Name, err)
		} else {
			for _, r := range results {
				if r.OriginalPath != r.RomanizedPath {
					log.Printf("  Renamed: %s -> %s", r.OriginalPath, r.RomanizedPath)
				}
				if r.FrontmatterAdded {
					log.Printf("  Added frontmatter: %s (title: %s)", r.RomanizedPath, r.OriginalTitle)
				}
			}

			// Move files into matching folders (e.g., meomeideu.md -> meomeideu/meomeideu.md)
			log.Printf("Post-processing: Moving files into matching folders in %s...", spaceDirTemp)
			if err := postprocess.MoveFilesIntoMatchingFolders(spaceDirTemp); err != nil {
				log.Printf("Warning: failed to move files into folders: %v", err)
			}

			// Merge Korean folders into romanized folders (e.g., 머메이드/files -> meomeideu/files)
			log.Printf("Post-processing: Merging Korean folder contents into romanized folders in %s...", spaceDirTemp)
			if err := postprocess.MergeKoreanFoldersIntoRomanized(spaceDirTemp); err != nil {
				log.Printf("Warning: failed to merge Korean folders: %v", err)
			}

			// Rename any remaining Korean folders to romanized names
			log.Printf("Post-processing: Renaming remaining Korean folders in %s...", spaceDirTemp)
			if err := postprocess.RenameRemainingKoreanFolders(spaceDirTemp); err != nil {
				log.Printf("Warning: failed to rename remaining Korean folders: %v", err)
			}

			// Rename any remaining Korean .md files to romanized names
			log.Printf("Post-processing: Renaming remaining Korean files in %s...", spaceDirTemp)
			if err := postprocess.RenameRemainingKoreanFiles(spaceDirTemp); err != nil {
				log.Printf("Warning: failed to rename remaining Korean files: %v", err)
			}

			// Sanitize special characters in folder and .md file names (e.g., & -> -and-)
			log.Printf("Post-processing: Sanitizing special characters in %s...", spaceDirTemp)
			if err := postprocess.SanitizeSpecialCharacters(spaceDirTemp); err != nil {
				log.Printf("Warning: failed to sanitize special characters: %v", err)
			}

			// Remove space before .md extension (e.g., "OIDC .md" -> "OIDC.md")
			log.Printf("Post-processing: Removing space before extension in %s...", spaceDirTemp)
			if err := postprocess.RemoveSpaceBeforeExtension(spaceDirTemp); err != nil {
				log.Printf("Warning: failed to remove space before extension: %v", err)
			}

			// Move files into matching folders again (after sanitization, folder/file names may now match)
			// e.g., sihaengchako.md -> sihaengchako/sihaengchako.md
			log.Printf("Post-processing: Moving files into matching folders (after sanitization) in %s...", spaceDirTemp)
			if err := postprocess.MoveFilesIntoMatchingFolders(spaceDirTemp); err != nil {
				log.Printf("Warning: failed to move files into folders: %v", err)
			}

			// Merge files that were incorrectly split due to "/" in title (AFTER romanization)
			// This handles romanized filenames like "Security365-hwangyeong-injeung/inga-gwanryeon-gongtong-ereo-peiji.md"
			log.Printf("Post-processing: Merging slash-split files (after romanization) in %s...", spaceDirTemp)
			if err := postprocess.MergeSlashSplitFiles(spaceDirTemp); err != nil {
				log.Printf("Warning: failed to merge slash-split files: %v", err)
			}

			// Cleanup empty directories
			if err := postprocess.CleanupEmptyDirs(spaceDirTemp); err != nil {
				log.Printf("Warning: failed to cleanup empty dirs: %v", err)
			}
		}

		// Perform atomic swap: replace old directory with new one
		log.Printf("Performing atomic swap for space '%s'...", exported.Space.Name)
		if err := atomicSwap(spaceDir, spaceDirTemp, spaceDirOld); err != nil {
			log.Printf("Error during atomic swap for %s: %v", exported.Space.Name, err)
			cleanupTempDir(spaceDirTemp)
			continue
		}
		log.Printf("Space '%s': successfully swapped to %s", exported.Space.Name, spaceDir)
	}

	log.Println("=== Sync Complete ===")
	log.Printf("Total spaces: %d", len(exportedSpaces))
	log.Printf("Total files:  %d", totalFiles)
	log.Printf("Output dir:   %s", cfg.OutputDir)

	return nil
}

// sanitizeDirName creates a safe directory name
func sanitizeDirName(name string) string {
	replacer := strings.NewReplacer(
		// " ", "-",
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "",
	)
	return strings.TrimSpace(replacer.Replace(name))
}

// atomicSwap performs an atomic directory swap using the Blue-Green deployment pattern.
// It replaces finalDir with tempDir atomically by:
// 1. Renaming current finalDir to oldDir (if exists)
// 2. Renaming tempDir to finalDir
// 3. Removing oldDir
// This ensures zero-downtime updates and safe rollback on failure.
func atomicSwap(finalDir, tempDir, oldDir string) error {
	// 1. Remove any existing old directory from previous runs
	if _, err := os.Stat(oldDir); err == nil {
		if err := os.RemoveAll(oldDir); err != nil {
			return fmt.Errorf("failed to remove existing old directory: %w", err)
		}
	}

	// 2. Rename current directory to old (if it exists)
	if _, err := os.Stat(finalDir); err == nil {
		if err := os.Rename(finalDir, oldDir); err != nil {
			return fmt.Errorf("failed to rename current to old: %w", err)
		}
	}

	// 3. Rename temp directory to final
	if err := os.Rename(tempDir, finalDir); err != nil {
		// Rollback: restore old directory to final
		if _, statErr := os.Stat(oldDir); statErr == nil {
			if rollbackErr := os.Rename(oldDir, finalDir); rollbackErr != nil {
				return fmt.Errorf("failed to rename temp to final: %w (rollback also failed: %v)", err, rollbackErr)
			}
		}
		return fmt.Errorf("failed to rename temp to final: %w", err)
	}

	// 4. Remove old directory (swap completed, this is just cleanup)
	if _, err := os.Stat(oldDir); err == nil {
		if err := os.RemoveAll(oldDir); err != nil {
			// Just warn, the swap was successful
			log.Printf("Warning: failed to remove old directory %s: %v", oldDir, err)
		}
	}

	return nil
}

// cleanupTempDir removes the temporary directory if it exists.
// Used for cleanup on error.
func cleanupTempDir(tempDir string) {
	if _, err := os.Stat(tempDir); err == nil {
		if err := os.RemoveAll(tempDir); err != nil {
			log.Printf("Warning: failed to cleanup temp directory %s: %v", tempDir, err)
		}
	}
}

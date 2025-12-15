package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jung/doc2git/internal/config"
	"github.com/jung/doc2git/internal/docmost"
	"github.com/jung/doc2git/internal/postprocess"
)

func main() {
	// Parse command line flags
	outputDir := flag.String("output", "", "Output directory for exported markdown files (overrides OUTPUT_DIR env)")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Override output directory if specified via flag
	if *outputDir != "" {
		cfg.OutputDir = *outputDir
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
		os.Exit(1)
	}

	fmt.Println("=== Docmost Markdown Exporter ===")
	fmt.Printf("Server: %s\n", cfg.DocmostBaseURL)
	fmt.Printf("Output: %s\n", cfg.OutputDir)
	fmt.Println()

	// Create Docmost client
	client, err := docmost.NewClient(cfg.DocmostBaseURL, cfg.DocmostEmail, cfg.DocmostPassword)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	// Login
	fmt.Println("Logging in to Docmost...")
	if err := client.Login(); err != nil {
		fmt.Fprintf(os.Stderr, "Login failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Login successful!")
	fmt.Println()

	// Export all spaces
	fmt.Println("Exporting all spaces...")
	exportedSpaces, err := client.ExportAllSpaces()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Export failed: %v\n", err)
		os.Exit(1)
	}

	if len(exportedSpaces) == 0 {
		fmt.Println("No spaces found to export.")
		os.Exit(0)
	}

	// Save exported files to output directory
	totalFiles := 0
	for _, exported := range exportedSpaces {
		spaceDir := filepath.Join(cfg.OutputDir, sanitizeDirName(exported.Space.Name))

		// Create space directory
		if err := os.MkdirAll(spaceDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", spaceDir, err)
			continue
		}

		// Write files
		for filename, content := range exported.Files {
			filePath := filepath.Join(spaceDir, filename)

			// Create parent directories if needed
			parentDir := filepath.Dir(filePath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", parentDir, err)
				continue
			}

			// Write file
			if err := os.WriteFile(filePath, content, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing file %s: %v\n", filePath, err)
				continue
			}

			totalFiles++
		}

		// Save metadata JSON file
		if exported.Metadata != nil {
			metaPath := filepath.Join(spaceDir, "_metadata.json")
			metaData, err := json.MarshalIndent(exported.Metadata, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshaling metadata for %s: %v\n", exported.Space.Name, err)
			} else {
				if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing metadata file %s: %v\n", metaPath, err)
				} else {
					fmt.Printf("Space '%s': metadata saved to %s\n", exported.Space.Name, metaPath)
				}
			}
		}

		fmt.Printf("Space '%s': %d files saved to %s\n", exported.Space.Name, len(exported.Files), spaceDir)

		// Post-process: Remove untitled placeholder files (untitled.md with "# untitled" content)
		fmt.Printf("Post-processing: Removing untitled placeholder files in %s...\n", spaceDir)
		if err := postprocess.RemoveUntitledFiles(spaceDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove untitled files: %v\n", err)
		}

		// Post-process: Wrap placeholders with backticks (before frontmatter)
		fmt.Printf("Post-processing: Wrapping placeholders with backticks in %s...\n", spaceDir)
		if err := postprocess.WrapPlaceholdersWithBackticks(spaceDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to wrap placeholders: %v\n", err)
		}

		// Post-process: Wrap angle brackets with backticks (before frontmatter)
		fmt.Printf("Post-processing: Wrapping angle brackets with backticks in %s...\n", spaceDir)
		if err := postprocess.WrapAngleBracketsWithBackticks(spaceDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to wrap angle brackets: %v\n", err)
		}

		// Post-process: Wrap raw HTML (like tables) with code blocks
		fmt.Printf("Post-processing: Wrapping raw HTML with code blocks in %s...\n", spaceDir)
		if err := postprocess.WrapRawHTMLWithCodeBlock(spaceDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to wrap raw HTML: %v\n", err)
		}

		// Post-process: Merge files that were incorrectly split due to "/" in title (BEFORE romanization)
		// This handles Korean filenames like "Security365 환경 인증/인가 관련 공통 에러 페이지.md"
		fmt.Printf("Post-processing: Merging slash-split files (before romanization) in %s...\n", spaceDir)
		if err := postprocess.MergeSlashSplitFiles(spaceDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to merge slash-split files: %v\n", err)
		}

		// Post-process: Romanize Korean filenames and add frontmatter
		fmt.Printf("Post-processing: Romanizing Korean filenames in %s...\n", spaceDir)
		results, err := postprocess.RomanizeSpace(spaceDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to romanize space %s: %v\n", exported.Space.Name, err)
		} else {
			for _, r := range results {
				if r.OriginalPath != r.RomanizedPath {
					fmt.Printf("  Renamed: %s -> %s\n", r.OriginalPath, r.RomanizedPath)
				}
				if r.FrontmatterAdded {
					fmt.Printf("  Added frontmatter: %s (title: %s)\n", r.RomanizedPath, r.OriginalTitle)
				}
			}

			// Move files into matching folders (e.g., meomeideu.md -> meomeideu/meomeideu.md)
			fmt.Printf("Post-processing: Moving files into matching folders in %s...\n", spaceDir)
			if err := postprocess.MoveFilesIntoMatchingFolders(spaceDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to move files into folders: %v\n", err)
			}

			// Merge Korean folders into romanized folders (e.g., 머메이드/files -> meomeideu/files)
			fmt.Printf("Post-processing: Merging Korean folder contents into romanized folders in %s...\n", spaceDir)
			if err := postprocess.MergeKoreanFoldersIntoRomanized(spaceDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to merge Korean folders: %v\n", err)
			}

			// Rename any remaining Korean folders to romanized names
			fmt.Printf("Post-processing: Renaming remaining Korean folders in %s...\n", spaceDir)
			if err := postprocess.RenameRemainingKoreanFolders(spaceDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to rename remaining Korean folders: %v\n", err)
			}

			// Rename any remaining Korean .md files to romanized names
			fmt.Printf("Post-processing: Renaming remaining Korean files in %s...\n", spaceDir)
			if err := postprocess.RenameRemainingKoreanFiles(spaceDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to rename remaining Korean files: %v\n", err)
			}

			// Sanitize special characters in folder and .md file names (e.g., & -> -and-)
			fmt.Printf("Post-processing: Sanitizing special characters in %s...\n", spaceDir)
			if err := postprocess.SanitizeSpecialCharacters(spaceDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to sanitize special characters: %v\n", err)
			}

			// Remove space before .md extension (e.g., "OIDC .md" -> "OIDC.md")
			fmt.Printf("Post-processing: Removing space before extension in %s...\n", spaceDir)
			if err := postprocess.RemoveSpaceBeforeExtension(spaceDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove space before extension: %v\n", err)
			}

			// Move files into matching folders again (after sanitization, folder/file names may now match)
			// e.g., sihaengchako.md -> sihaengchako/sihaengchako.md
			fmt.Printf("Post-processing: Moving files into matching folders (after sanitization) in %s...\n", spaceDir)
			if err := postprocess.MoveFilesIntoMatchingFolders(spaceDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to move files into folders: %v\n", err)
			}

			// Merge files that were incorrectly split due to "/" in title (AFTER romanization)
			// This handles romanized filenames like "Security365-hwangyeong-injeung/inga-gwanryeon-gongtong-ereo-peiji.md"
			fmt.Printf("Post-processing: Merging slash-split files (after romanization) in %s...\n", spaceDir)
			if err := postprocess.MergeSlashSplitFiles(spaceDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to merge slash-split files: %v\n", err)
			}

			// Cleanup empty directories
			if err := postprocess.CleanupEmptyDirs(spaceDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to cleanup empty dirs: %v\n", err)
			}

			// Final pass: Remove untitled placeholder files again (in case any were created during postprocessing)
			fmt.Printf("Post-processing: Final removal of untitled placeholder files in %s...\n", spaceDir)
			if err := postprocess.RemoveUntitledFiles(spaceDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove untitled files (final pass): %v\n", err)
			}
		}
	}

	fmt.Println()
	fmt.Println("=== Export Complete ===")
	fmt.Printf("Total spaces: %d\n", len(exportedSpaces))
	fmt.Printf("Total files:  %d\n", totalFiles)
	fmt.Printf("Output dir:   %s\n", cfg.OutputDir)
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

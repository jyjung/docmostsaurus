package postprocess

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jung/doc2git/internal/hangul"
)

// PageMeta represents metadata for a single page
type PageMeta struct {
	ID           string      `json:"id"`
	SlugID       string      `json:"slugId"`
	Title        string      `json:"title"`
	Icon         *string     `json:"icon,omitempty"`
	Position     string      `json:"position"`
	ParentPageID *string     `json:"parentPageId,omitempty"`
	HasChildren  bool        `json:"hasChildren"`
	Children     []*PageMeta `json:"children,omitempty"`
	FilePath     string      `json:"filePath,omitempty"`
}

// SpaceMeta represents metadata for a space including page tree structure
type SpaceMeta struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Slug        string      `json:"slug"`
	Description string      `json:"description,omitempty"`
	CreatedAt   string      `json:"createdAt"`
	UpdatedAt   string      `json:"updatedAt"`
	Pages       []*PageMeta `json:"pages"`
	TotalPages  int         `json:"totalPages"`
}

// RenameResult contains the result of renaming operation
type RenameResult struct {
	OriginalPath   string
	RomanizedPath  string
	OriginalTitle  string
	FrontmatterAdded bool
}

// RomanizeSpace reads _metadata.json and renames Korean files/folders to romanized names
func RomanizeSpace(spaceDir string) ([]RenameResult, error) {
	metaPath := filepath.Join(spaceDir, "_metadata.json")

	// Read metadata file
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var spaceMeta SpaceMeta
	if err := json.Unmarshal(metaData, &spaceMeta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	var results []RenameResult

	// Process all pages recursively with sidebar position
	for i, page := range spaceMeta.Pages {
		pageResults, err := processPage(spaceDir, page, "", i+1)
		if err != nil {
			fmt.Printf("Warning: failed to process page %s: %v\n", page.Title, err)
			continue
		}
		results = append(results, pageResults...)
	}

	return results, nil
}

// processPage processes a single page and its children
func processPage(spaceDir string, page *PageMeta, parentRomanizedDir string, sidebarPosition int) ([]RenameResult, error) {
	var results []RenameResult

	if page.FilePath == "" {
		// Process children even if this page has no file
		if page.HasChildren && len(page.Children) > 0 {
			for i, child := range page.Children {
				childResults, err := processPage(spaceDir, child, parentRomanizedDir, i+1)
				if err != nil {
					fmt.Printf("Warning: failed to process child page %s: %v\n", child.Title, err)
					continue
				}
				results = append(results, childResults...)
			}
		}
		return results, nil
	}

	originalPath := filepath.Join(spaceDir, page.FilePath)

	// Check if file exists
	if _, err := os.Stat(originalPath); os.IsNotExist(err) {
		fmt.Printf("Warning: file not found: %s\n", originalPath)
		return results, nil
	}

	// Calculate romanized path
	romanizedFilePath := romanizePath(page.FilePath)
	romanizedFullPath := filepath.Join(spaceDir, romanizedFilePath)

	// Determine the romanized directory for children
	currentRomanizedDir := ""
	if page.HasChildren {
		// For pages with children, the directory name is the file name without .md
		baseName := strings.TrimSuffix(filepath.Base(romanizedFilePath), ".md")
		currentRomanizedDir = filepath.Join(filepath.Dir(romanizedFilePath), baseName)
	}

	result := RenameResult{
		OriginalPath:  page.FilePath,
		RomanizedPath: romanizedFilePath,
		OriginalTitle: page.Title,
	}

	// Create parent directories if needed
	parentDir := filepath.Dir(romanizedFullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", parentDir, err)
	}

	// Read original file content
	content, err := os.ReadFile(originalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", originalPath, err)
	}

	// Add frontmatter if it's a markdown file
	if strings.HasSuffix(page.FilePath, ".md") {
		newContent := addFrontmatter(string(content), page.Title, sidebarPosition)
		content = []byte(newContent)
		result.FrontmatterAdded = true
	}

	// Write to new location
	if err := os.WriteFile(romanizedFullPath, content, 0644); err != nil {
		return nil, fmt.Errorf("failed to write file %s: %w", romanizedFullPath, err)
	}

	// Remove original file if it's different from the new path
	if originalPath != romanizedFullPath {
		if err := os.Remove(originalPath); err != nil {
			fmt.Printf("Warning: failed to remove original file %s: %v\n", originalPath, err)
		}
	}

	results = append(results, result)

	// Process children
	if page.HasChildren && len(page.Children) > 0 {
		for i, child := range page.Children {
			childResults, err := processPage(spaceDir, child, currentRomanizedDir, i+1)
			if err != nil {
				fmt.Printf("Warning: failed to process child page %s: %v\n", child.Title, err)
				continue
			}
			results = append(results, childResults...)
		}
	}

	return results, nil
}

// romanizePath converts a file path with Korean characters to romanized form
func romanizePath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasSuffix(part, ".md") {
			// Handle markdown files: romanize name but keep .md extension
			baseName := strings.TrimSuffix(part, ".md")
			romanized := hangul.Romanize(baseName)
			// Replace spaces with hyphens for cleaner URLs
			romanized = strings.ReplaceAll(romanized, " ", "-")
			parts[i] = romanized + ".md"
		} else {
			// Handle directories
			romanized := hangul.Romanize(part)
			romanized = strings.ReplaceAll(romanized, " ", "-")
			parts[i] = romanized
		}
	}
	return strings.Join(parts, "/")
}

// addFrontmatter adds YAML frontmatter to markdown content
func addFrontmatter(content string, title string, sidebarPosition int) string {
	// Remove .md extension from title if present
	title = strings.TrimSuffix(title, ".md")

	// Check if frontmatter already exists
	if strings.HasPrefix(strings.TrimSpace(content), "---") {
		// Frontmatter already exists, don't add again
		return content
	}

	frontmatter := fmt.Sprintf("---\ntitle: %s\nsidebar_position: %d\n---\n\n", title, sidebarPosition)
	return frontmatter + content
}

// MoveFilesIntoMatchingFolders moves files into folders when both share the same name
// e.g., meomeideu.md and meomeideu/ folder exist at same level -> move meomeideu.md into meomeideu/
func MoveFilesIntoMatchingFolders(spaceDir string) error {
	// Collect all directories first
	var dirs []string
	err := filepath.Walk(spaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != spaceDir {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// For each directory, check if there's a matching .md file at the same level
	for _, dir := range dirs {
		dirName := filepath.Base(dir)
		parentDir := filepath.Dir(dir)
		matchingFile := filepath.Join(parentDir, dirName+".md")

		// Check if matching file exists
		if _, err := os.Stat(matchingFile); err == nil {
			// File exists, move it into the folder
			newPath := filepath.Join(dir, dirName+".md")

			// Check if destination already exists
			if _, err := os.Stat(newPath); err == nil {
				fmt.Printf("Warning: destination already exists, skipping: %s\n", newPath)
				continue
			}

			// Move the file
			if err := os.Rename(matchingFile, newPath); err != nil {
				fmt.Printf("Warning: failed to move file %s to %s: %v\n", matchingFile, newPath, err)
				continue
			}
			fmt.Printf("  Moved: %s -> %s\n", matchingFile, newPath)
		}
	}

	return nil
}

// MergeKoreanFoldersIntoRomanized moves contents from Korean-named folders into their romanized counterparts
// e.g., 머메이드/files/ -> meomeideu/files/ when both 머메이드/ and meomeideu/ exist
func MergeKoreanFoldersIntoRomanized(spaceDir string) error {
	// Collect all directories at each level
	dirsByParent := make(map[string][]string)

	err := filepath.Walk(spaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != spaceDir {
			parentDir := filepath.Dir(path)
			dirsByParent[parentDir] = append(dirsByParent[parentDir], path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// For each parent directory, find Korean folders and their romanized counterparts
	for _, dirs := range dirsByParent {
		for _, koreanDir := range dirs {
			koreanName := filepath.Base(koreanDir)

			// Skip if the folder name doesn't contain Korean characters
			if !containsKorean(koreanName) {
				continue
			}

			// Get the romanized name for this Korean folder
			romanizedName := hangul.Romanize(koreanName)
			romanizedName = strings.ReplaceAll(romanizedName, " ", "-")

			// Check if romanized folder exists at the same level
			romanizedDir := filepath.Join(filepath.Dir(koreanDir), romanizedName)

			if romanizedDir == koreanDir {
				// Same path, skip
				continue
			}

			// Check if romanized directory exists
			if info, err := os.Stat(romanizedDir); err == nil && info.IsDir() {
				// Both Korean and romanized folders exist, merge contents
				fmt.Printf("  Merging Korean folder contents: %s -> %s\n", koreanDir, romanizedDir)

				if err := mergeDirectoryContents(koreanDir, romanizedDir); err != nil {
					fmt.Printf("Warning: failed to merge %s into %s: %v\n", koreanDir, romanizedDir, err)
					continue
				}

				// Remove the now-empty Korean folder
				if err := os.RemoveAll(koreanDir); err != nil {
					fmt.Printf("Warning: failed to remove Korean folder %s: %v\n", koreanDir, err)
				}
			}
		}
	}

	return nil
}

// containsKorean checks if a string contains Korean characters
func containsKorean(s string) bool {
	for _, r := range s {
		// Korean Unicode ranges: Hangul Syllables (AC00-D7AF), Hangul Jamo (1100-11FF), etc.
		if (r >= 0xAC00 && r <= 0xD7AF) || (r >= 0x1100 && r <= 0x11FF) || (r >= 0x3130 && r <= 0x318F) {
			return true
		}
	}
	return false
}

// mergeDirectoryContents moves all contents from src directory to dst directory
func mergeDirectoryContents(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// If destination directory exists, merge recursively
			if _, err := os.Stat(dstPath); err == nil {
				if err := mergeDirectoryContents(srcPath, dstPath); err != nil {
					return err
				}
				// Remove source directory after merging
				if err := os.RemoveAll(srcPath); err != nil {
					return fmt.Errorf("failed to remove source directory %s: %w", srcPath, err)
				}
			} else {
				// Destination doesn't exist, just move the directory
				if err := os.Rename(srcPath, dstPath); err != nil {
					return fmt.Errorf("failed to move directory %s to %s: %w", srcPath, dstPath, err)
				}
				fmt.Printf("    Moved directory: %s -> %s\n", srcPath, dstPath)
			}
		} else {
			// For files, check if destination exists
			if _, err := os.Stat(dstPath); err == nil {
				fmt.Printf("    Warning: file already exists, skipping: %s\n", dstPath)
				continue
			}
			// Move the file
			if err := os.Rename(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to move file %s to %s: %w", srcPath, dstPath, err)
			}
			fmt.Printf("    Moved file: %s -> %s\n", srcPath, dstPath)
		}
	}

	return nil
}

// CleanupEmptyDirs removes empty directories after renaming
func CleanupEmptyDirs(spaceDir string) error {
	return filepath.Walk(spaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}

		// Skip the root space directory
		if path == spaceDir {
			return nil
		}

		// Check if directory is empty
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}

		if len(entries) == 0 {
			if err := os.Remove(path); err != nil {
				fmt.Printf("Warning: failed to remove empty directory %s: %v\n", path, err)
			}
		}

		return nil
	})
}

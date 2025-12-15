package postprocess

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	OriginalPath     string
	RomanizedPath    string
	OriginalTitle    string
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

	// If the file moved to a different directory and references files/, copy the files/ folder
	originalDir := filepath.Dir(originalPath)
	newDir := filepath.Dir(romanizedFullPath)
	if originalDir != newDir {
		// Check if MD file references files/ and if source files/ exists
		if strings.Contains(string(content), "](files/") {
			sourceFilesDir := filepath.Join(originalDir, "files")
			if info, err := os.Stat(sourceFilesDir); err == nil && info.IsDir() {
				destFilesDir := filepath.Join(newDir, "files")
				// Copy/merge files folder (copyFilesToDestination handles existing files)
				fmt.Printf("  Copying files folder for moved MD: %s -> %s\n", sourceFilesDir, destFilesDir)
				if err := copyFilesToDestination(sourceFilesDir, destFilesDir); err != nil {
					fmt.Printf("Warning: failed to copy files folder: %v\n", err)
				}
			}
		}
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
			// romanized = strings.ReplaceAll(romanized, " ", "-")
			// Clean up multiple consecutive hyphens
			romanized = cleanupHyphens(romanized)
			parts[i] = romanized + ".md"
		} else {
			// Handle directories
			romanized := hangul.Romanize(part)
			// romanized = strings.ReplaceAll(romanized, " ", "-")
			// Clean up multiple consecutive hyphens
			romanized = cleanupHyphens(romanized)
			parts[i] = romanized
		}
	}
	return strings.Join(parts, "/")
}

// cleanupHyphens removes multiple consecutive hyphens and leading/trailing hyphens
func cleanupHyphens(s string) string {
	// Replace multiple consecutive hyphens with single hyphen
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	// Remove leading/trailing hyphens
	s = strings.Trim(s, "-")
	return s
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
// Also copies the files/ folder contents from the same level into the target folder's files/
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

			// Before moving the file, copy the files/ folder from the same level
			// The md file may reference images in the files/ folder at the same level
			sourceFilesDir := filepath.Join(parentDir, "files")
			if info, err := os.Stat(sourceFilesDir); err == nil && info.IsDir() {
				destFilesDir := filepath.Join(dir, "files")
				fmt.Printf("  Copying files folder for %s: %s -> %s\n", dirName+".md", sourceFilesDir, destFilesDir)
				if err := copyFilesToDestination(sourceFilesDir, destFilesDir); err != nil {
					fmt.Printf("Warning: failed to copy files folder: %v\n", err)
				}
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

// copyFilesToDestination copies files from source files/ folder to destination files/ folder
// If destination files/ folder exists, it merges the contents (does not overwrite existing files)
func copyFilesToDestination(srcDir, dstDir string) error {
	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectories
			if err := copyFilesToDestination(srcPath, dstPath); err != nil {
				fmt.Printf("    Warning: failed to copy subdirectory %s: %v\n", srcPath, err)
			}
		} else {
			// Check if destination file already exists
			if _, err := os.Stat(dstPath); err == nil {
				// File already exists, skip
				continue
			}

			// Copy the file
			srcContent, err := os.ReadFile(srcPath)
			if err != nil {
				fmt.Printf("    Warning: failed to read file %s: %v\n", srcPath, err)
				continue
			}

			if err := os.WriteFile(dstPath, srcContent, 0644); err != nil {
				fmt.Printf("    Warning: failed to write file %s: %v\n", dstPath, err)
				continue
			}
			fmt.Printf("    Copied: %s -> %s\n", srcPath, dstPath)
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

	// Sort parent directories by depth (deepest first) to process children before parents
	// This prevents the issue where parent folder merge copies already-processed child folders
	var parentDirs []string
	for parentDir := range dirsByParent {
		parentDirs = append(parentDirs, parentDir)
	}
	sort.Slice(parentDirs, func(i, j int) bool {
		// Sort by path depth descending (deeper paths first)
		depthI := strings.Count(parentDirs[i], string(filepath.Separator))
		depthJ := strings.Count(parentDirs[j], string(filepath.Separator))
		return depthI > depthJ
	})

	// For each parent directory (deepest first), find Korean folders and their romanized counterparts
	for _, parentDir := range parentDirs {
		dirs := dirsByParent[parentDir]
		for _, koreanDir := range dirs {
			koreanName := filepath.Base(koreanDir)

			// Skip if the folder name doesn't contain Korean characters
			if !containsKorean(koreanName) {
				continue
			}

			// Check if the Korean folder still exists (may have been deleted in previous iteration)
			if _, err := os.Stat(koreanDir); os.IsNotExist(err) {
				continue
			}

			// Get the romanized name for this Korean folder
			romanizedName := hangul.Romanize(koreanName)
			// romanizedName = strings.ReplaceAll(romanizedName, " ", "-")

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
			// Log when files folder is being moved or merged
			if entry.Name() == "files" {
				fmt.Printf("    [files folder] Processing: %s -> %s\n", srcPath, dstPath)
			}
			// If destination directory exists, merge recursively
			if _, err := os.Stat(dstPath); err == nil {
				if entry.Name() == "files" {
					fmt.Printf("    [files folder] Merging into existing: %s -> %s\n", srcPath, dstPath)
				}
				if err := mergeDirectoryContents(srcPath, dstPath); err != nil {
					return err
				}
				// Remove source directory after merging
				if err := os.RemoveAll(srcPath); err != nil {
					return fmt.Errorf("failed to remove source directory %s: %w", srcPath, err)
				}
				if entry.Name() == "files" {
					fmt.Printf("    [files folder] Merge complete, source removed: %s\n", srcPath)
				}
			} else {
				// Destination doesn't exist, just move the directory
				if err := os.Rename(srcPath, dstPath); err != nil {
					return fmt.Errorf("failed to move directory %s to %s: %w", srcPath, dstPath, err)
				}
				if entry.Name() == "files" {
					fmt.Printf("    [files folder] Moved: %s -> %s\n", srcPath, dstPath)
				} else {
					fmt.Printf("    Moved directory: %s -> %s\n", srcPath, dstPath)
				}
			}
		} else {
			// For files, check if destination exists
			if _, err := os.Stat(dstPath); err == nil {
				// File already exists at destination, remove source file (keep destination)
				if err := os.Remove(srcPath); err != nil {
					fmt.Printf("    Warning: failed to remove duplicate source file %s: %v\n", srcPath, err)
				}
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

// RenameRemainingKoreanFolders renames any remaining Korean-named folders to romanized names
// This handles folders that weren't merged because no romanized counterpart existed
func RenameRemainingKoreanFolders(spaceDir string) error {
	// We need to process from deepest to shallowest, so collect all Korean folders first
	var koreanFolders []string

	err := filepath.Walk(spaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != spaceDir {
			folderName := filepath.Base(path)
			if containsKorean(folderName) {
				koreanFolders = append(koreanFolders, path)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Sort by depth descending (deepest first)
	sort.Slice(koreanFolders, func(i, j int) bool {
		depthI := strings.Count(koreanFolders[i], string(filepath.Separator))
		depthJ := strings.Count(koreanFolders[j], string(filepath.Separator))
		return depthI > depthJ
	})

	// Rename each Korean folder to romanized name
	for _, koreanFolder := range koreanFolders {
		// Check if folder still exists (may have been moved as part of parent)
		if _, err := os.Stat(koreanFolder); os.IsNotExist(err) {
			continue
		}

		folderName := filepath.Base(koreanFolder)
		romanizedName := hangul.Romanize(folderName)
		// romanizedName = strings.ReplaceAll(romanizedName, " ", "-")

		romanizedPath := filepath.Join(filepath.Dir(koreanFolder), romanizedName)

		// If romanized path is the same, skip
		if romanizedPath == koreanFolder {
			continue
		}

		// If romanized folder already exists, merge into it
		if _, err := os.Stat(romanizedPath); err == nil {
			fmt.Printf("  Merging remaining Korean folder: %s -> %s\n", koreanFolder, romanizedPath)
			if err := mergeDirectoryContents(koreanFolder, romanizedPath); err != nil {
				fmt.Printf("Warning: failed to merge %s into %s: %v\n", koreanFolder, romanizedPath, err)
				continue
			}
			if err := os.RemoveAll(koreanFolder); err != nil {
				fmt.Printf("Warning: failed to remove Korean folder %s: %v\n", koreanFolder, err)
			}
		} else {
			// Romanized folder doesn't exist, just rename
			fmt.Printf("  Renaming Korean folder: %s -> %s\n", koreanFolder, romanizedPath)
			if err := os.Rename(koreanFolder, romanizedPath); err != nil {
				fmt.Printf("Warning: failed to rename %s to %s: %v\n", koreanFolder, romanizedPath, err)
			}
		}
	}

	return nil
}

// RenameRemainingKoreanFiles renames any remaining Korean-named .md files to romanized names
// This handles files that weren't processed by RomanizeSpace (not in _metadata.json)
func RenameRemainingKoreanFiles(spaceDir string) error {
	var koreanFiles []string

	err := filepath.Walk(spaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileName := filepath.Base(path)
			// Only process .md files with Korean characters
			if strings.HasSuffix(strings.ToLower(fileName), ".md") && containsKorean(fileName) {
				koreanFiles = append(koreanFiles, path)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Rename each Korean file to romanized name
	for _, koreanFile := range koreanFiles {
		// Check if file still exists
		if _, err := os.Stat(koreanFile); os.IsNotExist(err) {
			continue
		}

		fileName := filepath.Base(koreanFile)
		// Remove .md extension, romanize, then add it back
		baseName := strings.TrimSuffix(fileName, ".md")
		romanizedName := hangul.Romanize(baseName)
		// romanizedName = strings.ReplaceAll(romanizedName, " ", "-")
		romanizedName = romanizedName + ".md"

		// If romanized name is the same, skip
		if romanizedName == fileName {
			continue
		}

		romanizedPath := filepath.Join(filepath.Dir(koreanFile), romanizedName)

		// Check if destination exists
		if _, err := os.Stat(romanizedPath); err == nil {
			fmt.Printf("Warning: romanized file already exists, skipping: %s\n", romanizedPath)
			continue
		}

		fmt.Printf("  Renaming Korean file: %s -> %s\n", koreanFile, romanizedPath)
		if err := os.Rename(koreanFile, romanizedPath); err != nil {
			fmt.Printf("Warning: failed to rename %s to %s: %v\n", koreanFile, romanizedPath, err)
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

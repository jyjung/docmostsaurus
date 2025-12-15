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

// SanitizeSpecialCharacters renames folders and .md files that contain special characters
// that could break Docusaurus. Non-.md files keep their original names.
// Special characters like &, +, (, ), etc. are replaced with safe alternatives.
func SanitizeSpecialCharacters(spaceDir string) error {
	// Collect all paths that need sanitizing (folders and .md files)
	var pathsToSanitize []string

	err := filepath.Walk(spaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == spaceDir {
			return nil
		}

		name := filepath.Base(path)

		// Check if name contains special characters that need sanitizing
		if needsSanitizing(name) {
			// For files, only sanitize .md files
			if !info.IsDir() {
				if strings.HasSuffix(strings.ToLower(name), ".md") {
					pathsToSanitize = append(pathsToSanitize, path)
				}
				// Non-.md files are skipped (keep original names)
			} else {
				// Directories are always sanitized
				pathsToSanitize = append(pathsToSanitize, path)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Sort by depth descending (deepest first) to avoid path conflicts
	sort.Slice(pathsToSanitize, func(i, j int) bool {
		depthI := strings.Count(pathsToSanitize[i], string(filepath.Separator))
		depthJ := strings.Count(pathsToSanitize[j], string(filepath.Separator))
		return depthI > depthJ
	})

	// Rename each path
	for _, oldPath := range pathsToSanitize {
		// Check if path still exists (may have been moved as part of parent)
		if _, err := os.Stat(oldPath); os.IsNotExist(err) {
			continue
		}

		oldName := filepath.Base(oldPath)
		newName := sanitizeName(oldName)

		if newName == oldName {
			continue
		}

		newPath := filepath.Join(filepath.Dir(oldPath), newName)

		// Check if new path already exists
		if _, err := os.Stat(newPath); err == nil {
			// Destination exists, merge if directory
			info, _ := os.Stat(oldPath)
			if info.IsDir() {
				fmt.Printf("  Merging sanitized folder: %s -> %s\n", oldPath, newPath)
				if err := mergeDirectoryContents(oldPath, newPath); err != nil {
					fmt.Printf("Warning: failed to merge %s into %s: %v\n", oldPath, newPath, err)
					continue
				}
				if err := os.RemoveAll(oldPath); err != nil {
					fmt.Printf("Warning: failed to remove folder %s: %v\n", oldPath, err)
				}
			} else {
				fmt.Printf("Warning: sanitized file already exists, skipping: %s\n", newPath)
			}
		} else {
			// Destination doesn't exist, just rename
			fmt.Printf("  Sanitizing: %s -> %s\n", oldPath, newPath)
			if err := os.Rename(oldPath, newPath); err != nil {
				fmt.Printf("Warning: failed to rename %s to %s: %v\n", oldPath, newPath, err)
			}
		}
	}

	return nil
}

// needsSanitizing checks if a name contains special characters that need to be sanitized
func needsSanitizing(name string) bool {
	// Include space in the list of characters that need sanitizing
	specialChars := []string{" ", "&", "+", "(", ")", "[", "]", "{", "}", "'", "\"", ",", ";", "!", "@", "#", "$", "%", "^", "=", "`", "~"}
	for _, char := range specialChars {
		if strings.Contains(name, char) {
			return true
		}
	}
	return false
}

// sanitizeName replaces special characters in a name with safe alternatives
func sanitizeName(name string) string {
	// Check if it's a .md file
	isMdFile := strings.HasSuffix(strings.ToLower(name), ".md")
	baseName := name
	ext := ""
	if isMdFile {
		ext = filepath.Ext(name)
		baseName = strings.TrimSuffix(name, ext)
	}

	// Replace special characters (including space -> hyphen)
	replacer := strings.NewReplacer(
		// " ", "-",
		"&", "-and-",
		"+", "-plus-",
		"(", "",
		")", "",
		"[", "",
		"]", "",
		"{", "",
		"}", "",
		"'", "",
		"\"", "",
		",", "",
		";", "",
		"!", "",
		"@", "-at-",
		"#", "-num-",
		"$", "",
		"%", "-pct-",
		"^", "",
		"=", "-eq-",
		"`", "",
		"~", "",
	)
	sanitized := replacer.Replace(baseName)

	// Clean up multiple consecutive hyphens
	for strings.Contains(sanitized, "--") {
		sanitized = strings.ReplaceAll(sanitized, "--", "-")
	}

	// Remove leading/trailing hyphens
	sanitized = strings.Trim(sanitized, "-")

	// Add extension back for .md files
	if isMdFile {
		sanitized = sanitized + ext
	}

	return sanitized
}

// MergeSlashSplitFiles fixes files that were incorrectly split by docmost due to "/" in the title.
// When a page title contains "/", docmost creates a nested directory structure instead of a single file.
// This function reads _metadata.json to find such pages and merges the split files back together.
//
// This function handles BOTH cases:
// 1. Before romanization: Korean filenames (e.g., "Security365 환경 인증/인가 관련 공통 에러 페이지.md")
// 2. After romanization: Romanized filenames (e.g., "Security365-hwangyeong-injeung/inga-gwanryeon-gongtong-ereo-peiji.md")
//
// Example:
//
//	Title: "Security365 환경 인증/인가 관련 공통 에러 페이지"
//	Wrong structure created by docmost:
//	  └── Security365 환경 인증/
//	      └── 인가 관련 공통 에러 페이지.md
//	After romanization:
//	  └── Security365-hwangyeong-injeung/
//	      └── inga-gwanryeon-gongtong-ereo-peiji.md
//	Expected (after this fix):
//	  └── Security365-hwangyeong-injeung-inga-gwanryeon-gongtong-ereo-peiji.md
func MergeSlashSplitFiles(spaceDir string) error {
	metaPath := filepath.Join(spaceDir, "_metadata.json")

	// Read metadata file
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	var spaceMeta SpaceMeta
	if err := json.Unmarshal(metaData, &spaceMeta); err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Find all pages with "/" in their title
	slashPages := findPagesWithSlashInTitle(spaceMeta.Pages)

	for _, page := range slashPages {
		if err := mergeSlashSplitFile(spaceDir, page, false); err != nil {
			fmt.Printf("Warning: failed to merge slash-split file (Korean) for '%s': %v\n", page.Title, err)
		}
		if err := mergeSlashSplitFile(spaceDir, page, true); err != nil {
			fmt.Printf("Warning: failed to merge slash-split file (romanized) for '%s': %v\n", page.Title, err)
		}
	}

	return nil
}

// findPagesWithSlashInTitle recursively finds all pages that have "/" in their title
func findPagesWithSlashInTitle(pages []*PageMeta) []*PageMeta {
	var result []*PageMeta

	for _, page := range pages {
		if strings.Contains(page.Title, "/") {
			result = append(result, page)
		}
		if page.HasChildren && len(page.Children) > 0 {
			result = append(result, findPagesWithSlashInTitle(page.Children)...)
		}
	}

	return result
}

// mergeSlashSplitFile merges a file that was incorrectly split due to "/" in the title
// If romanized is true, it looks for romanized filenames; otherwise, it looks for original Korean filenames
func mergeSlashSplitFile(spaceDir string, page *PageMeta, romanized bool) error {
	// The title contains "/", which means docmost created a nested structure
	// We need to find the incorrectly created path and merge it into a single file

	// Split the title by "/" to understand the nested structure
	titleParts := strings.Split(page.Title, "/")
	if len(titleParts) < 2 {
		return nil // No slash, nothing to merge
	}

	// Build the expected wrong path parts
	// First part becomes directory, last part becomes file
	var wrongPathParts []string
	for i, part := range titleParts {
		part = strings.TrimSpace(part)

		var processedPart string
		if romanized {
			processedPart = hangul.Romanize(part)
			// processedPart = strings.ReplaceAll(processedPart, " ", "-")
			// Clean up multiple consecutive hyphens
			for strings.Contains(processedPart, "--") {
				processedPart = strings.ReplaceAll(processedPart, "--", "-")
			}
			processedPart = strings.Trim(processedPart, "-")
		} else {
			// Keep original (Korean) filename
			processedPart = part
		}

		if i == len(titleParts)-1 {
			// Last part is the file
			wrongPathParts = append(wrongPathParts, processedPart+".md")
		} else {
			// Other parts are directories
			wrongPathParts = append(wrongPathParts, processedPart)
		}
	}

	// Determine the parent directory from the page's context
	parentDir := spaceDir

	// Search for the wrong directory structure
	wrongDirName := wrongPathParts[0]

	// Build the correct merged filename
	var correctFileNameParts []string
	for _, part := range titleParts {
		part = strings.TrimSpace(part)

		var processedPart string
		if romanized {
			processedPart = hangul.Romanize(part)
			// processedPart = strings.ReplaceAll(processedPart, " ", "-")
			for strings.Contains(processedPart, "--") {
				processedPart = strings.ReplaceAll(processedPart, "--", "-")
			}
			processedPart = strings.Trim(processedPart, "-")
		} else {
			// For Korean filenames, replace spaces with hyphens to match file system
			processedPart = part
		}
		correctFileNameParts = append(correctFileNameParts, processedPart)
	}

	var correctFileName string
	if romanized {
		correctFileName = strings.Join(correctFileNameParts, "-") + ".md"
	} else {
		// For Korean, join with space (as it would appear in the original title)
		// but this becomes a single filename
		correctFileName = strings.Join(correctFileNameParts, "") + ".md"
	}

	// Find and process the wrong structure
	return filepath.Walk(parentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if !info.IsDir() {
			return nil
		}

		dirName := filepath.Base(path)
		if dirName != wrongDirName {
			return nil
		}

		// Found a directory matching the first part of the title
		// Build the wrong file path by joining all parts after the first directory
		wrongFilePath := path
		for _, part := range wrongPathParts[1:] {
			wrongFilePath = filepath.Join(wrongFilePath, part)
		}

		if _, err := os.Stat(wrongFilePath); os.IsNotExist(err) {
			return nil // Wrong file doesn't exist, skip
		}

		// Read the content from the wrong location
		content, err := os.ReadFile(wrongFilePath)
		if err != nil {
			fmt.Printf("Warning: failed to read file %s: %v\n", wrongFilePath, err)
			return nil
		}

		// Determine the correct path (same parent as the wrong directory)
		correctFilePath := filepath.Join(filepath.Dir(path), correctFileName)

		// Check if correct file already exists
		if _, err := os.Stat(correctFilePath); err == nil {
			fmt.Printf("  Correct file already exists, skipping: %s\n", correctFilePath)
			return nil
		}

		// Write to the correct location
		fmt.Printf("  Merging slash-split file: %s -> %s\n", wrongFilePath, correctFilePath)
		if err := os.WriteFile(correctFilePath, content, 0644); err != nil {
			fmt.Printf("Warning: failed to write merged file %s: %v\n", correctFilePath, err)
			return nil
		}

		// Remove the wrong file
		if err := os.Remove(wrongFilePath); err != nil {
			fmt.Printf("Warning: failed to remove wrong file %s: %v\n", wrongFilePath, err)
		}

		// Try to remove the empty parent directories
		cleanupEmptyParentDirs(path, parentDir)

		return filepath.SkipDir // Found and processed, skip further processing in this directory
	})
}

// cleanupEmptyParentDirs removes empty directories up to the stopDir
func cleanupEmptyParentDirs(dir, stopDir string) {
	for dir != stopDir && dir != filepath.Dir(dir) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		if len(entries) > 0 {
			return // Directory not empty
		}

		// Remove empty directory
		if err := os.Remove(dir); err != nil {
			fmt.Printf("Warning: failed to remove empty directory %s: %v\n", dir, err)
			return
		}
		fmt.Printf("  Removed empty directory: %s\n", dir)

		// Move up to parent
		dir = filepath.Dir(dir)
	}
}

// RemoveSpaceBeforeExtension renames .md files that have a space before the extension.
// For example: "OIDC .md" -> "OIDC.md"
// This fixes issues where Docusaurus fails to load chunks for files with space before extension.
func RemoveSpaceBeforeExtension(spaceDir string) error {
	// Collect all .md files that have space before extension
	var pathsToRename []string

	err := filepath.Walk(spaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		name := filepath.Base(path)

		// Check if the file has space before .md extension
		if hasSpaceBeforeExtension(name) {
			pathsToRename = append(pathsToRename, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Sort by depth descending (deepest first) to avoid path conflicts
	sort.Slice(pathsToRename, func(i, j int) bool {
		depthI := strings.Count(pathsToRename[i], string(filepath.Separator))
		depthJ := strings.Count(pathsToRename[j], string(filepath.Separator))
		return depthI > depthJ
	})

	// Rename each file
	for _, oldPath := range pathsToRename {
		// Check if path still exists
		if _, err := os.Stat(oldPath); os.IsNotExist(err) {
			continue
		}

		oldName := filepath.Base(oldPath)
		newName := removeSpaceBeforeExt(oldName)

		if newName == oldName {
			continue
		}

		newPath := filepath.Join(filepath.Dir(oldPath), newName)

		// Check if new path already exists
		if _, err := os.Stat(newPath); err == nil {
			fmt.Printf("Warning: target file already exists, skipping: %s\n", newPath)
			continue
		}

		fmt.Printf("  Removing space before extension: %s -> %s\n", oldPath, newPath)
		if err := os.Rename(oldPath, newPath); err != nil {
			fmt.Printf("Warning: failed to rename %s to %s: %v\n", oldPath, newPath, err)
		}
	}

	return nil
}

// hasSpaceBeforeExtension checks if a filename has space(s) before the extension
func hasSpaceBeforeExtension(name string) bool {
	// Match common extensions with space before them
	extensions := []string{".md", ".MD", ".Md", ".mD"}
	for _, ext := range extensions {
		if strings.HasSuffix(name, ext) {
			// Check if there's a space before the extension
			baseName := strings.TrimSuffix(name, ext)
			if strings.HasSuffix(baseName, " ") {
				return true
			}
		}
	}
	return false
}

// removeSpaceBeforeExt removes trailing spaces before the file extension
func removeSpaceBeforeExt(name string) string {
	// Find the extension (case-insensitive for .md)
	lowerName := strings.ToLower(name)
	if !strings.HasSuffix(lowerName, ".md") {
		return name
	}

	// Get the actual extension (preserving case)
	ext := name[len(name)-3:] // ".md" or ".MD" etc.
	baseName := name[:len(name)-3]

	// Remove trailing spaces from base name
	baseName = strings.TrimRight(baseName, " ")

	return baseName + ext
}

// RemoveUntitledFiles removes untitled placeholder files created by Docmost.
// It removes files matching these criteria:
// 1. Filename is "untitled.md" (case-insensitive) with content "# untitled" or "# untitled (N)"
// 2. Filename is "untitled N.md" (where N is a number, case-insensitive) with content starting with "# untitled"
func RemoveUntitledFiles(spaceDir string) error {
	var filesToRemove []string

	err := filepath.Walk(spaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		fileName := filepath.Base(path)
		lowerFileName := strings.ToLower(fileName)

		// Check if the file matches untitled patterns
		isUntitled, isNumbered := isUntitledFile(lowerFileName)
		if !isUntitled {
			return nil
		}

		// Read the file content
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Warning: failed to read file %s: %v\n", path, err)
			return nil
		}

		trimmedContent := strings.TrimSpace(string(content))

		// For "untitled.md": content must be "# untitled" or "# untitled (N)"
		// For "untitled N.md": content must start with "# untitled"
		shouldRemove := false
		if isNumbered {
			// Numbered untitled files: check if content starts with "# untitled"
			shouldRemove = strings.HasPrefix(trimmedContent, "# untitled")
		} else {
			// Plain untitled.md: check if content is "# untitled" or starts with "# untitled"
			shouldRemove = isUntitledContent(trimmedContent)
		}

		if shouldRemove {
			filesToRemove = append(filesToRemove, path)
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Remove the identified files
	for _, filePath := range filesToRemove {
		fmt.Printf("  Removing untitled placeholder: %s\n", filePath)
		if err := os.Remove(filePath); err != nil {
			fmt.Printf("Warning: failed to remove file %s: %v\n", filePath, err)
		}
	}

	if len(filesToRemove) > 0 {
		fmt.Printf("  Removed %d untitled placeholder file(s)\n", len(filesToRemove))
	}

	return nil
}

// isUntitledContent checks if content matches untitled placeholder patterns.
// Returns true if content is exactly "# untitled" or "# untitled (N)" with no other meaningful content.
func isUntitledContent(trimmedContent string) bool {
	// Exact match
	if trimmedContent == "# untitled" {
		return true
	}

	// Check for "# untitled (N)" pattern
	if !strings.HasPrefix(trimmedContent, "# untitled") {
		return false
	}

	// Get the part after "# untitled"
	rest := trimmedContent[10:] // len("# untitled") = 10

	// If nothing after "# untitled", it's a match (already handled above, but just in case)
	if len(rest) == 0 {
		return true
	}

	// Check what comes after "# untitled"
	firstChar := rest[0]

	// If followed by space and "(", check for "(N)" pattern with no additional content
	if firstChar == ' ' && len(rest) > 1 && rest[1] == '(' {
		// Find closing paren
		closeIdx := strings.Index(rest, ")")
		if closeIdx > 2 {
			// Check if content between ( and ) is all digits
			between := rest[2:closeIdx]
			allDigits := true
			for _, r := range between {
				if r < '0' || r > '9' {
					allDigits = false
					break
				}
			}
			if allDigits && len(between) > 0 {
				// Check if there's any meaningful content after "(N)"
				afterParen := strings.TrimSpace(rest[closeIdx+1:])
				if len(afterParen) == 0 {
					return true
				}
			}
		}
	}

	return false
}

// isUntitledFile checks if a lowercase filename matches untitled patterns.
// Returns (isUntitled, isNumbered) where:
// - isUntitled: true if filename is "untitled.md" or "untitled N.md" (N is a number)
// - isNumbered: true if filename is "untitled N.md" pattern
func isUntitledFile(lowerFileName string) (bool, bool) {
	// Check for exact "untitled.md"
	if lowerFileName == "untitled.md" {
		return true, false
	}

	// Check for "untitled N.md" pattern (where N is one or more digits)
	if !strings.HasPrefix(lowerFileName, "untitled ") {
		return false, false
	}
	if !strings.HasSuffix(lowerFileName, ".md") {
		return false, false
	}

	// Extract the part between "untitled " and ".md"
	middle := lowerFileName[9 : len(lowerFileName)-3] // len("untitled ") = 9, len(".md") = 3

	// Check if middle part is all digits
	if len(middle) == 0 {
		return false, false
	}
	for _, r := range middle {
		if r < '0' || r > '9' {
			return false, false
		}
	}

	return true, true
}

package docmost

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// ExportedSpace contains the exported space data
type ExportedSpace struct {
	Space    Space
	Files    map[string][]byte // filepath -> content (markdown files and attachments)
	Metadata *SpaceMeta        // metadata with page tree structure
}

// ExportSpaceAsZip exports an entire space as a ZIP file (markdown format)
func (c *Client) ExportSpaceAsZip(spaceID string) ([]byte, error) {
	reqBody := map[string]interface{}{
		"spaceId":            spaceID,
		"format":             "markdown",
		"includeAttachments": true,
	}
	body, _ := json.Marshal(reqBody)

	resp, err := c.doRequest("POST", "/api/spaces/export", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to export space: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("export space failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return io.ReadAll(resp.Body)
}

// ExportSpace exports a space and returns extracted files with metadata
func (c *Client) ExportSpace(space Space) (*ExportedSpace, error) {
	zipData, err := c.ExportSpaceAsZip(space.ID)
	if err != nil {
		return nil, err
	}

	files, err := extractZip(zipData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract zip: %w", err)
	}

	// Get metadata with page tree structure
	metadata, err := c.GetSpaceMetadata(space, files)
	if err != nil {
		// Log warning but don't fail the export
		fmt.Printf("Warning: failed to get metadata for space %s: %v\n", space.Name, err)
	}

	return &ExportedSpace{
		Space:    space,
		Files:    files,
		Metadata: metadata,
	}, nil
}

// ExportAllSpaces exports all accessible spaces
func (c *Client) ExportAllSpaces() ([]*ExportedSpace, error) {
	spaces, err := c.ListSpaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list spaces: %w", err)
	}

	var exportedSpaces []*ExportedSpace
	for _, space := range spaces {
		fmt.Printf("Exporting space: %s (%s)\n", space.Name, space.ID)

		exported, err := c.ExportSpace(space)
		if err != nil {
			fmt.Printf("Warning: failed to export space %s: %v\n", space.Name, err)
			continue
		}

		exportedSpaces = append(exportedSpaces, exported)
		fmt.Printf("  Exported %d files from space: %s\n", len(exported.Files), space.Name)
	}

	return exportedSpaces, nil
}

// extractZip extracts files from a ZIP archive
func extractZip(zipData []byte) (map[string][]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}

	files := make(map[string][]byte)
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", file.Name, err)
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", file.Name, err)
		}

		// Normalize path separators
		name := filepath.ToSlash(file.Name)
		files[name] = content
	}

	return files, nil
}

// SanitizeFilename creates a safe filename from a title
func SanitizeFilename(title string) string {
	// Replace problematic characters
	replacer := strings.NewReplacer(
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
	return strings.TrimSpace(replacer.Replace(title))
}

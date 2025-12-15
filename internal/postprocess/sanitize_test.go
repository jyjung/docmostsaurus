package postprocess

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestNeedsSanitizing tests the needsSanitizing function
func TestNeedsSanitizing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"plain text", "simple-name", false},
		{"with space", "hello world", true},
		{"with ampersand", "foo&bar", true},
		{"with plus", "foo+bar", true},
		{"with parentheses", "foo(bar)", true},
		{"with brackets", "foo[bar]", true},
		{"with braces", "foo{bar}", true},
		{"with single quote", "foo'bar", true},
		{"with double quote", "foo\"bar", true},
		{"with comma", "foo,bar", true},
		{"with semicolon", "foo;bar", true},
		{"with exclamation", "foo!bar", true},
		{"with at sign", "foo@bar", true},
		{"with hash", "foo#bar", true},
		{"with dollar", "foo$bar", true},
		{"with percent", "foo%bar", true},
		{"with caret", "foo^bar", true},
		{"with equals", "foo=bar", true},
		{"with backtick", "foo`bar", true},
		{"with tilde", "foo~bar", true},
		{"with hyphen only", "foo-bar", false},
		{"with underscore only", "foo_bar", false},
		{"with dot only", "foo.bar", false},
		{"empty string", "", false},
		{"korean without special", "한글이름", false},
		{"korean with space", "한글 이름", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := needsSanitizing(tc.input)
			if result != tc.expected {
				t.Errorf("needsSanitizing(%q) = %v, expected %v", tc.input, result, tc.expected)
			}
		})
	}
}

// TestSanitizeName tests the sanitizeName function
func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain text", "simple-name", "simple-name"},
		{"space to hyphen", "hello world", "hello-world"},
		{"multiple spaces", "hello   world", "hello-world"},
		{"ampersand replacement", "foo&bar", "foo-and-bar"},
		{"plus replacement", "foo+bar", "foo-plus-bar"},
		{"at sign replacement", "foo@bar", "foo-at-bar"},
		{"hash replacement", "foo#bar", "foo-num-bar"},
		{"percent replacement", "foo%bar", "foo-pct-bar"},
		{"equals replacement", "foo=bar", "foo-eq-bar"},
		{"parentheses removal", "foo(bar)", "foobar"},
		{"brackets removal", "foo[bar]", "foobar"},
		{"braces removal", "foo{bar}", "foobar"},
		{"quotes removal", "foo'bar\"baz", "foobarbaz"},
		{"comma semicolon removal", "foo,bar;baz", "foobarbaz"},
		{"exclamation removal", "hello!", "hello"},
		{"dollar removal", "price$100", "price100"},
		{"caret removal", "foo^bar", "foobar"},
		{"backtick removal", "foo`bar", "foobar"},
		{"tilde removal", "foo~bar", "foobar"},
		{"leading special chars", "  hello", "hello"},
		{"trailing special chars", "hello  ", "hello"},
		{"complex mixed", "Hello (World) & Friends!", "Hello-World-and-Friends"},
		{"md file with space", "hello world.md", "hello-world.md"},
		{"md file with special", "test (1).md", "test-1.md"},
		{"md file complex", "API Guide & Reference (v2).md", "API-Guide-and-Reference-v2.md"},
		{"preserve extension case", "README.MD", "README.MD"},
		{"multiple hyphens cleanup", "foo---bar", "foo-bar"},
		{"empty string", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeName(tc.input)
			if result != tc.expected {
				t.Errorf("sanitizeName(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestSanitizeSpecialCharacters_Basic tests basic sanitization of files and folders
func TestSanitizeSpecialCharacters_Basic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-sanitize-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create directory with special characters
	specialDir := filepath.Join(tempDir, "docs (v1)")
	if err := os.MkdirAll(specialDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	// Create file with special characters
	specialFile := filepath.Join(tempDir, "API Guide & Reference.md")
	if err := os.WriteFile(specialFile, []byte("# API Guide"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run sanitization
	if err := SanitizeSpecialCharacters(tempDir); err != nil {
		t.Fatalf("SanitizeSpecialCharacters failed: %v", err)
	}

	// Verify directory was renamed
	expectedDir := filepath.Join(tempDir, "docs-v1")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("expected directory %s to exist", expectedDir)
	}
	if _, err := os.Stat(specialDir); !os.IsNotExist(err) {
		t.Errorf("original directory %s should be removed", specialDir)
	}

	// Verify file was renamed
	expectedFile := filepath.Join(tempDir, "API-Guide-and-Reference.md")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", expectedFile)
	}
	if _, err := os.Stat(specialFile); !os.IsNotExist(err) {
		t.Errorf("original file %s should be removed", specialFile)
	}

	// Verify content preserved
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != "# API Guide" {
		t.Errorf("file content mismatch: got %q", string(content))
	}
}

// TestSanitizeSpecialCharacters_NestedDirectories tests sanitization with nested directories
func TestSanitizeSpecialCharacters_NestedDirectories(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-sanitize-nested-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create nested structure with special characters
	// parent (v1)/
	//   └── child [test]/
	//       └── file (1).md
	parentDir := filepath.Join(tempDir, "parent (v1)")
	childDir := filepath.Join(parentDir, "child [test]")
	if err := os.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	testFile := filepath.Join(childDir, "file (1).md")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run sanitization
	if err := SanitizeSpecialCharacters(tempDir); err != nil {
		t.Fatalf("SanitizeSpecialCharacters failed: %v", err)
	}

	// Verify final structure
	expectedFile := filepath.Join(tempDir, "parent-v1", "child-test", "file-1.md")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", expectedFile)
	}

	// Verify content
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != "content" {
		t.Errorf("content mismatch")
	}
}

// TestSanitizeSpecialCharacters_NonMdFilesPreserved tests that non-.md files are not renamed
func TestSanitizeSpecialCharacters_NonMdFilesPreserved(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-sanitize-nonmd-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create non-.md file with special characters
	specialFile := filepath.Join(tempDir, "image (1).png")
	if err := os.WriteFile(specialFile, []byte("PNG"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Create .md file with special characters for comparison
	mdFile := filepath.Join(tempDir, "doc (1).md")
	if err := os.WriteFile(mdFile, []byte("# Doc"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run sanitization
	if err := SanitizeSpecialCharacters(tempDir); err != nil {
		t.Fatalf("SanitizeSpecialCharacters failed: %v", err)
	}

	// Verify non-.md file was NOT renamed
	if _, err := os.Stat(specialFile); os.IsNotExist(err) {
		t.Errorf("non-.md file %s should be preserved", specialFile)
	}

	// Verify .md file WAS renamed
	expectedMd := filepath.Join(tempDir, "doc-1.md")
	if _, err := os.Stat(expectedMd); os.IsNotExist(err) {
		t.Errorf(".md file should be renamed to %s", expectedMd)
	}
}

// TestSanitizeSpecialCharacters_DirectoriesAlwaysSanitized tests that directories are always sanitized
func TestSanitizeSpecialCharacters_DirectoriesAlwaysSanitized(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-sanitize-dirs-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create directory with special characters containing non-.md files
	specialDir := filepath.Join(tempDir, "assets (v1)")
	if err := os.MkdirAll(specialDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	// Add a non-.md file inside
	imageFile := filepath.Join(specialDir, "logo.png")
	if err := os.WriteFile(imageFile, []byte("PNG"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run sanitization
	if err := SanitizeSpecialCharacters(tempDir); err != nil {
		t.Fatalf("SanitizeSpecialCharacters failed: %v", err)
	}

	// Verify directory was renamed
	expectedDir := filepath.Join(tempDir, "assets-v1")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("directory should be renamed to %s", expectedDir)
	}

	// Verify file inside is accessible at new path
	expectedImage := filepath.Join(expectedDir, "logo.png")
	if _, err := os.Stat(expectedImage); os.IsNotExist(err) {
		t.Errorf("file should exist at %s", expectedImage)
	}
}

// TestSanitizeSpecialCharacters_NoChangesNeeded tests when no sanitization is needed
func TestSanitizeSpecialCharacters_NoChangesNeeded(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-sanitize-nochange-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create clean directory and file
	cleanDir := filepath.Join(tempDir, "clean-dir")
	if err := os.MkdirAll(cleanDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	cleanFile := filepath.Join(cleanDir, "clean-file.md")
	if err := os.WriteFile(cleanFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run sanitization
	if err := SanitizeSpecialCharacters(tempDir); err != nil {
		t.Fatalf("SanitizeSpecialCharacters failed: %v", err)
	}

	// Verify nothing changed
	if _, err := os.Stat(cleanDir); os.IsNotExist(err) {
		t.Errorf("clean directory should still exist")
	}
	if _, err := os.Stat(cleanFile); os.IsNotExist(err) {
		t.Errorf("clean file should still exist")
	}
}

// TestSanitizeSpecialCharacters_EmptyDirectory tests with empty directory
func TestSanitizeSpecialCharacters_EmptyDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-sanitize-empty-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Run sanitization on empty directory
	if err := SanitizeSpecialCharacters(tempDir); err != nil {
		t.Fatalf("SanitizeSpecialCharacters failed on empty dir: %v", err)
	}
}

// TestSanitizeSpecialCharacters_ConflictingNames tests when sanitized name already exists
func TestSanitizeSpecialCharacters_ConflictingNames(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-sanitize-conflict-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create file that will conflict after sanitization
	existingFile := filepath.Join(tempDir, "test-file.md")
	if err := os.WriteFile(existingFile, []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Create file with special chars that would sanitize to same name
	specialFile := filepath.Join(tempDir, "test (file).md")
	if err := os.WriteFile(specialFile, []byte("special"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run sanitization - should not overwrite existing file
	if err := SanitizeSpecialCharacters(tempDir); err != nil {
		t.Fatalf("SanitizeSpecialCharacters failed: %v", err)
	}

	// Verify existing file content is preserved
	content, err := os.ReadFile(existingFile)
	if err != nil {
		t.Fatalf("failed to read existing file: %v", err)
	}
	if string(content) != "existing" {
		t.Errorf("existing file content should be preserved, got %q", string(content))
	}

	// Special file should still exist (couldn't be renamed due to conflict)
	if _, err := os.Stat(specialFile); os.IsNotExist(err) {
		t.Errorf("special file should still exist due to conflict")
	}
}

// TestSanitizeSpecialCharacters_RealWorldExample tests a realistic documentation structure
func TestSanitizeSpecialCharacters_RealWorldExample(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-sanitize-realworld-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create realistic structure
	// docs/
	//   ├── Getting Started (Quick).md
	//   ├── API Reference & Guide/
	//   │   ├── Authentication [OAuth].md
	//   │   └── Endpoints (v2).md
	//   └── FAQ's.md

	apiDir := filepath.Join(tempDir, "API Reference & Guide")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	files := map[string]string{
		filepath.Join(tempDir, "Getting Started (Quick).md"): "# Quick Start",
		filepath.Join(apiDir, "Authentication [OAuth].md"):   "# OAuth",
		filepath.Join(apiDir, "Endpoints (v2).md"):           "# Endpoints",
		filepath.Join(tempDir, "FAQ's.md"):                   "# FAQ",
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", path, err)
		}
	}

	// Run sanitization
	if err := SanitizeSpecialCharacters(tempDir); err != nil {
		t.Fatalf("SanitizeSpecialCharacters failed: %v", err)
	}

	// Verify expected structure
	expectedFiles := []string{
		filepath.Join(tempDir, "Getting-Started-Quick.md"),
		filepath.Join(tempDir, "API-Reference-and-Guide", "Authentication-OAuth.md"),
		filepath.Join(tempDir, "API-Reference-and-Guide", "Endpoints-v2.md"),
		filepath.Join(tempDir, "FAQs.md"),
	}

	for _, path := range expectedFiles {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file to exist: %s", path)
		}
	}
}

// TestMergeSlashSplitFiles tests the MergeSlashSplitFiles function
func TestMergeSlashSplitFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-merge-slash-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create metadata with a page that has "/" in the title
	metadata := SpaceMeta{
		ID:   "test-space",
		Name: "Test Space",
		Slug: "test",
		Pages: []*PageMeta{
			{
				ID:          "page1",
				SlugID:      "abc123",
				Title:       "Security365 환경 인증/인가 관련 공통 에러 페이지",
				Position:    "a0",
				HasChildren: false,
			},
		},
		TotalPages: 1,
	}

	metaData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal metadata: %v", err)
	}

	metaPath := filepath.Join(tempDir, "_metadata.json")
	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		t.Fatalf("failed to write metadata: %v", err)
	}

	// Create the wrong directory structure (as docmost would create it)
	// After romanization, it would be:
	// Security365-hwangyeong-injeung/
	//   └── inga-gwanryeon-gongtong-ereo-peiji.md
	wrongDir := filepath.Join(tempDir, "Security365-hwangyeong-injeung")
	if err := os.MkdirAll(wrongDir, 0755); err != nil {
		t.Fatalf("failed to create wrong dir: %v", err)
	}

	wrongFile := filepath.Join(wrongDir, "inga-gwanryeon-gongtong-ereo-peiji.md")
	if err := os.WriteFile(wrongFile, []byte("# Test Content"), 0644); err != nil {
		t.Fatalf("failed to create wrong file: %v", err)
	}

	// Run the merge function
	if err := MergeSlashSplitFiles(tempDir); err != nil {
		t.Fatalf("MergeSlashSplitFiles failed: %v", err)
	}

	// Verify the correct file was created
	correctFile := filepath.Join(tempDir, "Security365-hwangyeong-injeung-inga-gwanryeon-gongtong-ereo-peiji.md")
	if _, err := os.Stat(correctFile); os.IsNotExist(err) {
		t.Errorf("expected merged file to exist: %s", correctFile)
	}

	// Verify the content was preserved
	content, err := os.ReadFile(correctFile)
	if err != nil {
		t.Fatalf("failed to read merged file: %v", err)
	}
	if string(content) != "# Test Content" {
		t.Errorf("content mismatch: got %q", string(content))
	}

	// Verify the wrong file was removed
	if _, err := os.Stat(wrongFile); !os.IsNotExist(err) {
		t.Errorf("wrong file should be removed: %s", wrongFile)
	}

	// Verify the empty directory was removed
	if _, err := os.Stat(wrongDir); !os.IsNotExist(err) {
		t.Errorf("empty directory should be removed: %s", wrongDir)
	}
}

// TestMergeSlashSplitFiles_MultipleSlashes tests merging files with multiple "/" in title
func TestMergeSlashSplitFiles_MultipleSlashes(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-merge-multi-slash-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create metadata with a page that has multiple "/" in the title
	metadata := SpaceMeta{
		ID:   "test-space",
		Name: "Test Space",
		Slug: "test",
		Pages: []*PageMeta{
			{
				ID:          "page1",
				SlugID:      "abc123",
				Title:       "Part A/Part B/Part C",
				Position:    "a0",
				HasChildren: false,
			},
		},
		TotalPages: 1,
	}

	metaData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal metadata: %v", err)
	}

	metaPath := filepath.Join(tempDir, "_metadata.json")
	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		t.Fatalf("failed to write metadata: %v", err)
	}

	// Create the wrong nested directory structure
	// Part-A/Part-B/Part-C.md
	wrongDir := filepath.Join(tempDir, "Part-A", "Part-B")
	if err := os.MkdirAll(wrongDir, 0755); err != nil {
		t.Fatalf("failed to create wrong dir: %v", err)
	}

	wrongFile := filepath.Join(wrongDir, "Part-C.md")
	if err := os.WriteFile(wrongFile, []byte("# Multi Slash Content"), 0644); err != nil {
		t.Fatalf("failed to create wrong file: %v", err)
	}

	// Run the merge function
	if err := MergeSlashSplitFiles(tempDir); err != nil {
		t.Fatalf("MergeSlashSplitFiles failed: %v", err)
	}

	// Verify the correct file was created
	correctFile := filepath.Join(tempDir, "Part-A-Part-B-Part-C.md")
	if _, err := os.Stat(correctFile); os.IsNotExist(err) {
		t.Errorf("expected merged file to exist: %s", correctFile)
	}

	// Verify the content was preserved
	content, err := os.ReadFile(correctFile)
	if err != nil {
		t.Fatalf("failed to read merged file: %v", err)
	}
	if string(content) != "# Multi Slash Content" {
		t.Errorf("content mismatch: got %q", string(content))
	}
}

// TestMergeSlashSplitFiles_NoSlashInTitle tests when no pages have "/" in title
func TestMergeSlashSplitFiles_NoSlashInTitle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-merge-no-slash-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create metadata without "/" in titles
	metadata := SpaceMeta{
		ID:   "test-space",
		Name: "Test Space",
		Slug: "test",
		Pages: []*PageMeta{
			{
				ID:          "page1",
				SlugID:      "abc123",
				Title:       "Normal Title",
				Position:    "a0",
				HasChildren: false,
			},
		},
		TotalPages: 1,
	}

	metaData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal metadata: %v", err)
	}

	metaPath := filepath.Join(tempDir, "_metadata.json")
	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		t.Fatalf("failed to write metadata: %v", err)
	}

	// Create a normal file
	normalFile := filepath.Join(tempDir, "Normal-Title.md")
	if err := os.WriteFile(normalFile, []byte("# Normal"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run the merge function (should do nothing)
	if err := MergeSlashSplitFiles(tempDir); err != nil {
		t.Fatalf("MergeSlashSplitFiles failed: %v", err)
	}

	// Verify the file is unchanged
	if _, err := os.Stat(normalFile); os.IsNotExist(err) {
		t.Errorf("normal file should still exist: %s", normalFile)
	}
}

// TestMergeSlashSplitFiles_NestedPages tests merging with nested page structure
func TestMergeSlashSplitFiles_NestedPages(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-merge-nested-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	parentPageID := "parent1"
	// Create metadata with nested pages containing "/" in title
	metadata := SpaceMeta{
		ID:   "test-space",
		Name: "Test Space",
		Slug: "test",
		Pages: []*PageMeta{
			{
				ID:          "parent1",
				SlugID:      "parent123",
				Title:       "Parent",
				Position:    "a0",
				HasChildren: true,
				Children: []*PageMeta{
					{
						ID:           "child1",
						SlugID:       "child123",
						Title:        "인증/인가",
						Position:     "a0",
						ParentPageID: &parentPageID,
						HasChildren:  false,
					},
				},
			},
		},
		TotalPages: 2,
	}

	metaData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal metadata: %v", err)
	}

	metaPath := filepath.Join(tempDir, "_metadata.json")
	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		t.Fatalf("failed to write metadata: %v", err)
	}

	// Create the wrong structure inside parent directory
	parentDir := filepath.Join(tempDir, "Parent")
	wrongDir := filepath.Join(parentDir, "injeung")
	if err := os.MkdirAll(wrongDir, 0755); err != nil {
		t.Fatalf("failed to create wrong dir: %v", err)
	}

	wrongFile := filepath.Join(wrongDir, "inga.md")
	if err := os.WriteFile(wrongFile, []byte("# Nested Content"), 0644); err != nil {
		t.Fatalf("failed to create wrong file: %v", err)
	}

	// Run the merge function
	if err := MergeSlashSplitFiles(tempDir); err != nil {
		t.Fatalf("MergeSlashSplitFiles failed: %v", err)
	}

	// Verify the correct file was created
	correctFile := filepath.Join(parentDir, "injeung-inga.md")
	if _, err := os.Stat(correctFile); os.IsNotExist(err) {
		t.Errorf("expected merged file to exist: %s", correctFile)
	}
}

// TestMergeSlashSplitFiles_KoreanFilenames tests merging with original Korean filenames (before romanization)
func TestMergeSlashSplitFiles_KoreanFilenames(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-merge-korean-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	parentPageID := "planning1"
	// Create metadata matching the real-world case
	metadata := SpaceMeta{
		ID:   "test-space",
		Name: "Test Space",
		Slug: "test",
		Pages: []*PageMeta{
			{
				ID:          "planning1",
				SlugID:      "planning123",
				Title:       "Planning",
				Position:    "a0",
				HasChildren: true,
				Children: []*PageMeta{
					{
						ID:           "child1",
						SlugID:       "child123",
						Title:        "Security365 환경 인증/인가 관련 공통 에러 페이지",
						Position:     "a0",
						ParentPageID: &parentPageID,
						HasChildren:  false,
					},
				},
			},
		},
		TotalPages: 2,
	}

	metaData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal metadata: %v", err)
	}

	metaPath := filepath.Join(tempDir, "_metadata.json")
	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		t.Fatalf("failed to write metadata: %v", err)
	}

	// Create the wrong structure with Korean filenames (as docmost creates)
	planningDir := filepath.Join(tempDir, "Planning")
	wrongDir := filepath.Join(planningDir, "Security365 환경 인증")
	if err := os.MkdirAll(wrongDir, 0755); err != nil {
		t.Fatalf("failed to create wrong dir: %v", err)
	}

	wrongFile := filepath.Join(wrongDir, "인가 관련 공통 에러 페이지.md")
	if err := os.WriteFile(wrongFile, []byte("# Test Content"), 0644); err != nil {
		t.Fatalf("failed to create wrong file: %v", err)
	}

	// Run the merge function
	if err := MergeSlashSplitFiles(tempDir); err != nil {
		t.Fatalf("MergeSlashSplitFiles failed: %v", err)
	}

	// Verify the correct file was created (Korean merged filename)
	correctFile := filepath.Join(planningDir, "Security365 환경 인증인가 관련 공통 에러 페이지.md")
	if _, err := os.Stat(correctFile); os.IsNotExist(err) {
		t.Errorf("expected merged file to exist: %s", correctFile)
	}

	// Verify content preserved
	if _, err := os.Stat(correctFile); err == nil {
		content, err := os.ReadFile(correctFile)
		if err != nil {
			t.Fatalf("failed to read merged file: %v", err)
		}
		if string(content) != "# Test Content" {
			t.Errorf("content mismatch: got %q", string(content))
		}
	}

	// Verify the wrong file was removed
	if _, err := os.Stat(wrongFile); !os.IsNotExist(err) {
		t.Errorf("wrong file should be removed: %s", wrongFile)
	}

	// Verify the empty directory was removed
	if _, err := os.Stat(wrongDir); !os.IsNotExist(err) {
		t.Errorf("empty directory should be removed: %s", wrongDir)
	}
}

// TestHasSpaceBeforeExtension tests the hasSpaceBeforeExtension function
func TestHasSpaceBeforeExtension(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"no space before extension", "OIDC.md", false},
		{"single space before extension", "OIDC .md", true},
		{"multiple spaces before extension", "OIDC   .md", true},
		{"uppercase extension with space", "OIDC .MD", true},
		{"mixed case extension with space", "OIDC .Md", true},
		{"no extension", "OIDC", false},
		{"different extension", "OIDC .txt", false},
		{"space in middle only", "OID C.md", false},
		{"empty filename with extension", " .md", true},
		{"normal filename", "normal-file.md", false},
		{"korean filename with space", "인증 .md", true},
		{"korean filename without space", "인증.md", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := hasSpaceBeforeExtension(tc.input)
			if result != tc.expected {
				t.Errorf("hasSpaceBeforeExtension(%q) = %v, expected %v", tc.input, result, tc.expected)
			}
		})
	}
}

// TestRemoveSpaceBeforeExt tests the removeSpaceBeforeExt function
func TestRemoveSpaceBeforeExt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no space before extension", "OIDC.md", "OIDC.md"},
		{"single space before extension", "OIDC .md", "OIDC.md"},
		{"multiple spaces before extension", "OIDC   .md", "OIDC.md"},
		{"uppercase extension with space", "README .MD", "README.MD"},
		{"mixed case extension with space", "Doc .Md", "Doc.Md"},
		{"no extension", "OIDC", "OIDC"},
		{"different extension unchanged", "image .png", "image .png"},
		{"space in middle preserved", "OID C .md", "OID C.md"},
		{"korean filename with space", "인증 .md", "인증.md"},
		{"complex path component", "Authentication -and- Authorization Standards .md", "Authentication -and- Authorization Standards.md"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := removeSpaceBeforeExt(tc.input)
			if result != tc.expected {
				t.Errorf("removeSpaceBeforeExt(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestRemoveSpaceBeforeExtension_Basic tests basic functionality
func TestRemoveSpaceBeforeExtension_Basic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-space-ext-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create file with space before extension
	spaceFile := filepath.Join(tempDir, "OIDC .md")
	if err := os.WriteFile(spaceFile, []byte("# OIDC Content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run the function
	if err := RemoveSpaceBeforeExtension(tempDir); err != nil {
		t.Fatalf("RemoveSpaceBeforeExtension failed: %v", err)
	}

	// Verify file was renamed
	expectedFile := filepath.Join(tempDir, "OIDC.md")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", expectedFile)
	}

	// Verify original file is gone
	if _, err := os.Stat(spaceFile); !os.IsNotExist(err) {
		t.Errorf("original file %s should be removed", spaceFile)
	}

	// Verify content preserved
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != "# OIDC Content" {
		t.Errorf("file content mismatch: got %q", string(content))
	}
}

// TestRemoveSpaceBeforeExtension_NestedDirectories tests with nested directories
func TestRemoveSpaceBeforeExtension_NestedDirectories(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-space-ext-nested-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create nested structure matching the real-world case
	nestedDir := filepath.Join(tempDir, "SHIELD ID", "On-the-Job Training", "jh", "Authentication -and- Authorization Standards")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	// Create file with space before extension
	spaceFile := filepath.Join(nestedDir, "OIDC .md")
	if err := os.WriteFile(spaceFile, []byte("# OIDC"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run the function
	if err := RemoveSpaceBeforeExtension(tempDir); err != nil {
		t.Fatalf("RemoveSpaceBeforeExtension failed: %v", err)
	}

	// Verify file was renamed
	expectedFile := filepath.Join(nestedDir, "OIDC.md")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", expectedFile)
	}
}

// TestRemoveSpaceBeforeExtension_MultipleSpaces tests with multiple spaces before extension
func TestRemoveSpaceBeforeExtension_MultipleSpaces(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-space-ext-multi-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create file with multiple spaces before extension
	spaceFile := filepath.Join(tempDir, "Document   .md")
	if err := os.WriteFile(spaceFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run the function
	if err := RemoveSpaceBeforeExtension(tempDir); err != nil {
		t.Fatalf("RemoveSpaceBeforeExtension failed: %v", err)
	}

	// Verify file was renamed (all trailing spaces removed)
	expectedFile := filepath.Join(tempDir, "Document.md")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", expectedFile)
	}
}

// TestRemoveSpaceBeforeExtension_NoChangesNeeded tests when no files need changes
func TestRemoveSpaceBeforeExtension_NoChangesNeeded(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-space-ext-nochange-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create file without space before extension
	normalFile := filepath.Join(tempDir, "normal.md")
	if err := os.WriteFile(normalFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run the function
	if err := RemoveSpaceBeforeExtension(tempDir); err != nil {
		t.Fatalf("RemoveSpaceBeforeExtension failed: %v", err)
	}

	// Verify file is unchanged
	if _, err := os.Stat(normalFile); os.IsNotExist(err) {
		t.Errorf("file should still exist: %s", normalFile)
	}
}

// TestRemoveSpaceBeforeExtension_ConflictingNames tests when target name already exists
func TestRemoveSpaceBeforeExtension_ConflictingNames(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-space-ext-conflict-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create the target file first
	existingFile := filepath.Join(tempDir, "OIDC.md")
	if err := os.WriteFile(existingFile, []byte("existing content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Create file with space that would conflict
	spaceFile := filepath.Join(tempDir, "OIDC .md")
	if err := os.WriteFile(spaceFile, []byte("space content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run the function
	if err := RemoveSpaceBeforeExtension(tempDir); err != nil {
		t.Fatalf("RemoveSpaceBeforeExtension failed: %v", err)
	}

	// Verify existing file content is preserved
	content, err := os.ReadFile(existingFile)
	if err != nil {
		t.Fatalf("failed to read existing file: %v", err)
	}
	if string(content) != "existing content" {
		t.Errorf("existing file content should be preserved, got %q", string(content))
	}

	// Space file should still exist due to conflict
	if _, err := os.Stat(spaceFile); os.IsNotExist(err) {
		t.Errorf("space file should still exist due to conflict")
	}
}

// TestRemoveSpaceBeforeExtension_MultipleFiles tests with multiple files
func TestRemoveSpaceBeforeExtension_MultipleFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-space-ext-multiple-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create multiple files with space before extension
	files := []string{
		"Doc1 .md",
		"Doc2 .md",
		"Doc3 .MD",
	}

	for _, f := range files {
		path := filepath.Join(tempDir, f)
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", f, err)
		}
	}

	// Run the function
	if err := RemoveSpaceBeforeExtension(tempDir); err != nil {
		t.Fatalf("RemoveSpaceBeforeExtension failed: %v", err)
	}

	// Verify all files were renamed
	expectedFiles := []string{
		"Doc1.md",
		"Doc2.md",
		"Doc3.MD",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(tempDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file to exist: %s", path)
		}
	}
}

// TestRemoveSpaceBeforeExtension_NonMdFilesIgnored tests that non-.md files are ignored
func TestRemoveSpaceBeforeExtension_NonMdFilesIgnored(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-space-ext-nonmd-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create non-.md file with space before extension
	nonMdFile := filepath.Join(tempDir, "image .png")
	if err := os.WriteFile(nonMdFile, []byte("PNG"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run the function
	if err := RemoveSpaceBeforeExtension(tempDir); err != nil {
		t.Fatalf("RemoveSpaceBeforeExtension failed: %v", err)
	}

	// Verify non-.md file was NOT renamed
	if _, err := os.Stat(nonMdFile); os.IsNotExist(err) {
		t.Errorf("non-.md file should be preserved: %s", nonMdFile)
	}
}

// TestRemoveUntitledFiles_Basic tests basic removal of untitled placeholder files
func TestRemoveUntitledFiles_Basic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-untitled-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create an untitled.md file with "# untitled" content
	untitledFile := filepath.Join(tempDir, "untitled.md")
	if err := os.WriteFile(untitledFile, []byte("# untitled"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run the function
	if err := RemoveUntitledFiles(tempDir); err != nil {
		t.Fatalf("RemoveUntitledFiles failed: %v", err)
	}

	// Verify file was removed
	if _, err := os.Stat(untitledFile); !os.IsNotExist(err) {
		t.Errorf("untitled file should be removed: %s", untitledFile)
	}
}

// TestRemoveUntitledFiles_WithWhitespace tests removal with whitespace around content
func TestRemoveUntitledFiles_WithWhitespace(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-untitled-ws-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create untitled.md with whitespace around content
	untitledFile := filepath.Join(tempDir, "untitled.md")
	if err := os.WriteFile(untitledFile, []byte("  # untitled  \n\n"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run the function
	if err := RemoveUntitledFiles(tempDir); err != nil {
		t.Fatalf("RemoveUntitledFiles failed: %v", err)
	}

	// Verify file was removed
	if _, err := os.Stat(untitledFile); !os.IsNotExist(err) {
		t.Errorf("untitled file with whitespace should be removed: %s", untitledFile)
	}
}

// TestRemoveUntitledFiles_DifferentContent tests that files with different content are preserved
func TestRemoveUntitledFiles_DifferentContent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-untitled-diff-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create untitled.md with different content
	untitledFile := filepath.Join(tempDir, "untitled.md")
	if err := os.WriteFile(untitledFile, []byte("# untitled\n\nSome real content here."), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run the function
	if err := RemoveUntitledFiles(tempDir); err != nil {
		t.Fatalf("RemoveUntitledFiles failed: %v", err)
	}

	// Verify file was NOT removed (has actual content)
	if _, err := os.Stat(untitledFile); os.IsNotExist(err) {
		t.Errorf("untitled file with real content should be preserved: %s", untitledFile)
	}
}

// TestRemoveUntitledFiles_DifferentFilename tests that files with different names are preserved
func TestRemoveUntitledFiles_DifferentFilename(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-untitled-name-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file with "# untitled" content but different name
	otherFile := filepath.Join(tempDir, "other.md")
	if err := os.WriteFile(otherFile, []byte("# untitled"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run the function
	if err := RemoveUntitledFiles(tempDir); err != nil {
		t.Fatalf("RemoveUntitledFiles failed: %v", err)
	}

	// Verify file was NOT removed (different filename)
	if _, err := os.Stat(otherFile); os.IsNotExist(err) {
		t.Errorf("file with different name should be preserved: %s", otherFile)
	}
}

// TestRemoveUntitledFiles_NestedDirectories tests removal in nested directories
func TestRemoveUntitledFiles_NestedDirectories(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-untitled-nested-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create nested directories with untitled.md files
	nestedDir := filepath.Join(tempDir, "level1", "level2")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	// Create untitled.md at different levels
	untitledRoot := filepath.Join(tempDir, "untitled.md")
	untitledNested := filepath.Join(nestedDir, "untitled.md")

	if err := os.WriteFile(untitledRoot, []byte("# untitled"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if err := os.WriteFile(untitledNested, []byte("# untitled"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run the function
	if err := RemoveUntitledFiles(tempDir); err != nil {
		t.Fatalf("RemoveUntitledFiles failed: %v", err)
	}

	// Verify both files were removed
	if _, err := os.Stat(untitledRoot); !os.IsNotExist(err) {
		t.Errorf("untitled file at root should be removed: %s", untitledRoot)
	}
	if _, err := os.Stat(untitledNested); !os.IsNotExist(err) {
		t.Errorf("untitled file in nested dir should be removed: %s", untitledNested)
	}
}

// TestRemoveUntitledFiles_CaseInsensitive tests case-insensitive filename matching
func TestRemoveUntitledFiles_CaseInsensitive(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-untitled-case-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create files with different case variations
	files := []string{
		"Untitled.md",
		"UNTITLED.md",
		"UnTiTlEd.md",
	}

	for _, f := range files {
		path := filepath.Join(tempDir, f)
		if err := os.WriteFile(path, []byte("# untitled"), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", f, err)
		}
	}

	// Run the function
	if err := RemoveUntitledFiles(tempDir); err != nil {
		t.Fatalf("RemoveUntitledFiles failed: %v", err)
	}

	// Verify all files were removed
	for _, f := range files {
		path := filepath.Join(tempDir, f)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("untitled file should be removed (case insensitive): %s", path)
		}
	}
}

// TestRemoveUntitledFiles_EmptyDirectory tests with empty directory
func TestRemoveUntitledFiles_EmptyDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-untitled-empty-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Run on empty directory (should not fail)
	if err := RemoveUntitledFiles(tempDir); err != nil {
		t.Fatalf("RemoveUntitledFiles failed on empty dir: %v", err)
	}
}

// TestRemoveUntitledFiles_MixedFiles tests with a mix of files to keep and remove
func TestRemoveUntitledFiles_MixedFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-untitled-mixed-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Files to remove
	toRemove := []string{
		filepath.Join(tempDir, "untitled.md"),
		filepath.Join(tempDir, "subdir", "untitled.md"),
	}

	// Files to keep
	toKeep := []string{
		filepath.Join(tempDir, "real-content.md"),
		filepath.Join(tempDir, "untitled-but-has-content.md"),
		filepath.Join(tempDir, "subdir", "other.md"),
	}

	// Create directory structure
	if err := os.MkdirAll(filepath.Join(tempDir, "subdir"), 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Create files to remove
	for _, f := range toRemove {
		if err := os.WriteFile(f, []byte("# untitled"), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", f, err)
		}
	}

	// Create files to keep
	if err := os.WriteFile(toKeep[0], []byte("# Real Content\n\nThis is actual content."), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	// This file is named similarly but has different content, though filename doesn't match
	if err := os.WriteFile(toKeep[1], []byte("# untitled\n\nBut has more content."), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if err := os.WriteFile(toKeep[2], []byte("# Other"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run the function
	if err := RemoveUntitledFiles(tempDir); err != nil {
		t.Fatalf("RemoveUntitledFiles failed: %v", err)
	}

	// Verify files to remove are gone
	for _, f := range toRemove {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Errorf("file should be removed: %s", f)
		}
	}

	// Verify files to keep still exist
	for _, f := range toKeep {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("file should be preserved: %s", f)
		}
	}
}

// TestRemoveUntitledFiles_NumberedUntitled tests removal of "untitled 1.md", "untitled 2.md" etc.
func TestRemoveUntitledFiles_NumberedUntitled(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-untitled-numbered-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create numbered untitled files with "# untitled" content
	filesToRemove := []string{
		"untitled 1.md",
		"untitled 2.md",
		"untitled 10.md",
		"Untitled 3.md", // case insensitive
	}

	for _, f := range filesToRemove {
		path := filepath.Join(tempDir, f)
		if err := os.WriteFile(path, []byte("# untitled"), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", f, err)
		}
	}

	// Run the function
	if err := RemoveUntitledFiles(tempDir); err != nil {
		t.Fatalf("RemoveUntitledFiles failed: %v", err)
	}

	// Verify all numbered untitled files were removed
	for _, f := range filesToRemove {
		path := filepath.Join(tempDir, f)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("numbered untitled file should be removed: %s", path)
		}
	}
}

// TestRemoveUntitledFiles_NumberedWithContent tests that numbered untitled files with real content are preserved
func TestRemoveUntitledFiles_NumberedWithContent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-untitled-numbered-content-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create numbered untitled file but with additional real content
	untitledWithContent := filepath.Join(tempDir, "untitled 1.md")
	if err := os.WriteFile(untitledWithContent, []byte("# untitled\n\nThis has real content."), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run the function
	if err := RemoveUntitledFiles(tempDir); err != nil {
		t.Fatalf("RemoveUntitledFiles failed: %v", err)
	}

	// File should be removed because content starts with "# untitled"
	if _, err := os.Stat(untitledWithContent); !os.IsNotExist(err) {
		t.Errorf("numbered untitled file starting with '# untitled' should be removed: %s", untitledWithContent)
	}
}

// TestRemoveUntitledFiles_NumberedDifferentHeading tests that numbered untitled files with different heading are preserved
func TestRemoveUntitledFiles_NumberedDifferentHeading(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-untitled-numbered-diff-heading-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create numbered untitled file but with different heading
	untitledDiffHeading := filepath.Join(tempDir, "untitled 1.md")
	if err := os.WriteFile(untitledDiffHeading, []byte("# My Real Title\n\nContent here."), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run the function
	if err := RemoveUntitledFiles(tempDir); err != nil {
		t.Fatalf("RemoveUntitledFiles failed: %v", err)
	}

	// File should be preserved (different heading)
	if _, err := os.Stat(untitledDiffHeading); os.IsNotExist(err) {
		t.Errorf("numbered untitled file with different heading should be preserved: %s", untitledDiffHeading)
	}
}

// TestRemoveUntitledFiles_NumberedMixed tests mixed scenarios with numbered untitled files
func TestRemoveUntitledFiles_NumberedMixed(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-untitled-numbered-mixed-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Files to remove (numbered untitled with "# untitled" heading)
	toRemove := map[string]string{
		"untitled.md":    "# untitled",
		"untitled 1.md":  "# untitled",
		"untitled 2.md":  "# untitled\n",
		"untitled 99.md": "# untitled\n\nSome content after heading",
	}

	// Files to keep
	toKeep := map[string]string{
		"untitled 3.md":     "# Real Title\n\nContent",        // different heading
		"my-document.md":    "# untitled",                     // different filename
		"untitled-note.md":  "# untitled",                     // different filename pattern
		"untitledX.md":      "# untitled",                     // no space before number
		"untitled abc.md":   "# untitled",                     // not a number after space
	}

	for f, content := range toRemove {
		path := filepath.Join(tempDir, f)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", f, err)
		}
	}

	for f, content := range toKeep {
		path := filepath.Join(tempDir, f)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", f, err)
		}
	}

	// Run the function
	if err := RemoveUntitledFiles(tempDir); err != nil {
		t.Fatalf("RemoveUntitledFiles failed: %v", err)
	}

	// Verify files to remove are gone
	for f := range toRemove {
		path := filepath.Join(tempDir, f)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("file should be removed: %s", path)
		}
	}

	// Verify files to keep still exist
	for f := range toKeep {
		path := filepath.Join(tempDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("file should be preserved: %s", path)
		}
	}
}

// TestRemoveUntitledFiles_UntitledWithParenNumber tests removal of files with "# untitled (1)" content
func TestRemoveUntitledFiles_UntitledWithParenNumber(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-remove-untitled-paren-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Files to remove - content has "# untitled (N)" pattern
	toRemove := map[string]string{
		"untitled 1.md": "# untitled (1)",
		"untitled 2.md": "# untitled (2)\n",
		"untitled 3.md": "# untitled (3)\n\nSome content",
		"untitled.md":   "# untitled (1)", // also for non-numbered filename
	}

	// Files to keep
	toKeep := map[string]string{
		"untitled 4.md": "# Real Title",               // different heading
		"document.md":   "# untitled (1)",             // different filename
	}

	for f, content := range toRemove {
		path := filepath.Join(tempDir, f)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", f, err)
		}
	}

	for f, content := range toKeep {
		path := filepath.Join(tempDir, f)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", f, err)
		}
	}

	// Run the function
	if err := RemoveUntitledFiles(tempDir); err != nil {
		t.Fatalf("RemoveUntitledFiles failed: %v", err)
	}

	// Verify files to remove are gone
	for f := range toRemove {
		path := filepath.Join(tempDir, f)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("file should be removed: %s", path)
		}
	}

	// Verify files to keep still exist
	for f := range toKeep {
		path := filepath.Join(tempDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("file should be preserved: %s", path)
		}
	}
}

package postprocess

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMoveFilesIntoMatchingFolders_SingleLevel tests basic file moving at single level
func TestMoveFilesIntoMatchingFolders_SingleLevel(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test-move-files-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup structure:
	// ├── meomeideu.md
	// └── meomeideu/
	//     └── child.md

	// Create directory
	meomeideuDir := filepath.Join(tempDir, "meomeideu")
	if err := os.MkdirAll(meomeideuDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	// Create meomeideu.md at root level
	meomeideuFile := filepath.Join(tempDir, "meomeideu.md")
	if err := os.WriteFile(meomeideuFile, []byte("# Meomeideu Content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Create child.md inside meomeideu/
	childFile := filepath.Join(meomeideuDir, "child.md")
	if err := os.WriteFile(childFile, []byte("# Child Content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run function
	if err := MoveFilesIntoMatchingFolders(tempDir); err != nil {
		t.Fatalf("MoveFilesIntoMatchingFolders failed: %v", err)
	}

	// Verify: meomeideu.md should be moved to meomeideu/meomeideu.md
	movedFile := filepath.Join(meomeideuDir, "meomeideu.md")
	if _, err := os.Stat(movedFile); os.IsNotExist(err) {
		t.Errorf("expected file to be moved to %s, but it doesn't exist", movedFile)
	}

	// Verify: original meomeideu.md should not exist
	if _, err := os.Stat(meomeideuFile); !os.IsNotExist(err) {
		t.Errorf("expected original file %s to be removed, but it still exists", meomeideuFile)
	}

	// Verify: child.md should still exist
	if _, err := os.Stat(childFile); os.IsNotExist(err) {
		t.Errorf("child file %s should still exist", childFile)
	}

	// Verify content
	content, err := os.ReadFile(movedFile)
	if err != nil {
		t.Fatalf("failed to read moved file: %v", err)
	}
	if string(content) != "# Meomeideu Content" {
		t.Errorf("content mismatch: got %q", string(content))
	}
}

// TestMoveFilesIntoMatchingFolders_TwoLevels tests nested file moving at two levels
func TestMoveFilesIntoMatchingFolders_TwoLevels(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test-move-files-2level-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup structure:
	// ├── level1.md
	// └── level1/
	//     ├── level2.md
	//     └── level2/
	//         └── deep-child.md

	// Create directories
	level1Dir := filepath.Join(tempDir, "level1")
	level2Dir := filepath.Join(level1Dir, "level2")
	if err := os.MkdirAll(level2Dir, 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	// Create level1.md at root
	level1File := filepath.Join(tempDir, "level1.md")
	if err := os.WriteFile(level1File, []byte("# Level 1 Content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Create level2.md inside level1/
	level2File := filepath.Join(level1Dir, "level2.md")
	if err := os.WriteFile(level2File, []byte("# Level 2 Content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Create deep-child.md inside level2/
	deepChildFile := filepath.Join(level2Dir, "deep-child.md")
	if err := os.WriteFile(deepChildFile, []byte("# Deep Child Content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run function
	if err := MoveFilesIntoMatchingFolders(tempDir); err != nil {
		t.Fatalf("MoveFilesIntoMatchingFolders failed: %v", err)
	}

	// Verify: level1.md should be moved to level1/level1.md
	movedLevel1 := filepath.Join(level1Dir, "level1.md")
	if _, err := os.Stat(movedLevel1); os.IsNotExist(err) {
		t.Errorf("expected file to be moved to %s", movedLevel1)
	}

	// Verify: level2.md should be moved to level2/level2.md
	movedLevel2 := filepath.Join(level2Dir, "level2.md")
	if _, err := os.Stat(movedLevel2); os.IsNotExist(err) {
		t.Errorf("expected file to be moved to %s", movedLevel2)
	}

	// Verify: originals should not exist
	if _, err := os.Stat(level1File); !os.IsNotExist(err) {
		t.Errorf("original %s should be removed", level1File)
	}
	if _, err := os.Stat(level2File); !os.IsNotExist(err) {
		t.Errorf("original %s should be removed", level2File)
	}

	// Verify: deep-child.md should still exist
	if _, err := os.Stat(deepChildFile); os.IsNotExist(err) {
		t.Errorf("deep child file should still exist")
	}
}

// TestMoveFilesIntoMatchingFolders_ThreeLevels tests nested file moving at three levels
func TestMoveFilesIntoMatchingFolders_ThreeLevels(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test-move-files-3level-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup structure (before):
	// ├── category.md
	// ├── category/
	// │   ├── subcategory.md
	// │   └── subcategory/
	// │       ├── topic.md
	// │       └── topic/
	// │           └── article.md

	// Create directories
	categoryDir := filepath.Join(tempDir, "category")
	subcategoryDir := filepath.Join(categoryDir, "subcategory")
	topicDir := filepath.Join(subcategoryDir, "topic")
	if err := os.MkdirAll(topicDir, 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	// Create files
	categoryFile := filepath.Join(tempDir, "category.md")
	if err := os.WriteFile(categoryFile, []byte("# Category Content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	subcategoryFile := filepath.Join(categoryDir, "subcategory.md")
	if err := os.WriteFile(subcategoryFile, []byte("# Subcategory Content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	topicFile := filepath.Join(subcategoryDir, "topic.md")
	if err := os.WriteFile(topicFile, []byte("# Topic Content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	articleFile := filepath.Join(topicDir, "article.md")
	if err := os.WriteFile(articleFile, []byte("# Article Content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run function
	if err := MoveFilesIntoMatchingFolders(tempDir); err != nil {
		t.Fatalf("MoveFilesIntoMatchingFolders failed: %v", err)
	}

	// Expected structure (after):
	// ├── category/
	// │   ├── category.md          <- moved from root
	// │   └── subcategory/
	// │       ├── subcategory.md   <- moved from category/
	// │       └── topic/
	// │           ├── topic.md     <- moved from subcategory/
	// │           └── article.md

	// Verify all moves
	movedCategory := filepath.Join(categoryDir, "category.md")
	if _, err := os.Stat(movedCategory); os.IsNotExist(err) {
		t.Errorf("expected category.md to be moved to %s", movedCategory)
	}

	movedSubcategory := filepath.Join(subcategoryDir, "subcategory.md")
	if _, err := os.Stat(movedSubcategory); os.IsNotExist(err) {
		t.Errorf("expected subcategory.md to be moved to %s", movedSubcategory)
	}

	movedTopic := filepath.Join(topicDir, "topic.md")
	if _, err := os.Stat(movedTopic); os.IsNotExist(err) {
		t.Errorf("expected topic.md to be moved to %s", movedTopic)
	}

	// Verify originals are removed
	if _, err := os.Stat(categoryFile); !os.IsNotExist(err) {
		t.Errorf("original category.md should be removed")
	}
	if _, err := os.Stat(subcategoryFile); !os.IsNotExist(err) {
		t.Errorf("original subcategory.md should be removed")
	}
	if _, err := os.Stat(topicFile); !os.IsNotExist(err) {
		t.Errorf("original topic.md should be removed")
	}

	// Verify article.md still exists (no matching folder)
	if _, err := os.Stat(articleFile); os.IsNotExist(err) {
		t.Errorf("article.md should still exist in original location")
	}

	// Verify content of moved files
	content, _ := os.ReadFile(movedCategory)
	if string(content) != "# Category Content" {
		t.Errorf("category content mismatch")
	}

	content, _ = os.ReadFile(movedSubcategory)
	if string(content) != "# Subcategory Content" {
		t.Errorf("subcategory content mismatch")
	}

	content, _ = os.ReadFile(movedTopic)
	if string(content) != "# Topic Content" {
		t.Errorf("topic content mismatch")
	}
}

// TestMoveFilesIntoMatchingFolders_NoMatchingFolder tests that files without matching folders are not moved
func TestMoveFilesIntoMatchingFolders_NoMatchingFolder(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test-move-files-nomatch-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup structure:
	// ├── standalone.md
	// └── other-folder/
	//     └── child.md

	otherFolder := filepath.Join(tempDir, "other-folder")
	if err := os.MkdirAll(otherFolder, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	standaloneFile := filepath.Join(tempDir, "standalone.md")
	if err := os.WriteFile(standaloneFile, []byte("# Standalone"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	childFile := filepath.Join(otherFolder, "child.md")
	if err := os.WriteFile(childFile, []byte("# Child"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run function
	if err := MoveFilesIntoMatchingFolders(tempDir); err != nil {
		t.Fatalf("MoveFilesIntoMatchingFolders failed: %v", err)
	}

	// Verify: standalone.md should remain in original location
	if _, err := os.Stat(standaloneFile); os.IsNotExist(err) {
		t.Errorf("standalone.md should remain at %s", standaloneFile)
	}

	// Verify: child.md should remain in original location
	if _, err := os.Stat(childFile); os.IsNotExist(err) {
		t.Errorf("child.md should remain at %s", childFile)
	}
}

// TestMoveFilesIntoMatchingFolders_DestinationExists tests that files are not overwritten
func TestMoveFilesIntoMatchingFolders_DestinationExists(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test-move-files-exists-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup structure:
	// ├── conflict.md         <- should NOT be moved (destination exists)
	// └── conflict/
	//     └── conflict.md     <- already exists

	conflictDir := filepath.Join(tempDir, "conflict")
	if err := os.MkdirAll(conflictDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	sourceFile := filepath.Join(tempDir, "conflict.md")
	if err := os.WriteFile(sourceFile, []byte("# Source Content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	existingFile := filepath.Join(conflictDir, "conflict.md")
	if err := os.WriteFile(existingFile, []byte("# Existing Content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run function
	if err := MoveFilesIntoMatchingFolders(tempDir); err != nil {
		t.Fatalf("MoveFilesIntoMatchingFolders failed: %v", err)
	}

	// Verify: source file should still exist (not moved due to conflict)
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		t.Errorf("source file should remain at %s due to conflict", sourceFile)
	}

	// Verify: existing file content should be unchanged
	content, _ := os.ReadFile(existingFile)
	if string(content) != "# Existing Content" {
		t.Errorf("existing file content should not be overwritten, got: %q", string(content))
	}
}

// TestMoveFilesIntoMatchingFolders_MixedScenario tests a mixed scenario with various cases
func TestMoveFilesIntoMatchingFolders_MixedScenario(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test-move-files-mixed-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup structure (matching modify.md example):
	// ├── hangeul-gyeongro-haha.md   <- no matching folder, stays
	// ├── meomeideu.md               <- should move to meomeideu/
	// ├── meomeideu/
	// │   └── teseuteu-hangeul.md
	// ├── _metadata.json             <- no matching folder, stays
	// └── test.md                    <- no matching folder, stays

	meomeideuDir := filepath.Join(tempDir, "meomeideu")
	if err := os.MkdirAll(meomeideuDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	// Create files
	files := map[string]string{
		filepath.Join(tempDir, "hangeul-gyeongro-haha.md"):       "# Hangeul",
		filepath.Join(tempDir, "meomeideu.md"):                   "# Meomeideu",
		filepath.Join(meomeideuDir, "teseuteu-hangeul.md"):       "# Teseuteu",
		filepath.Join(tempDir, "_metadata.json"):                 "{}",
		filepath.Join(tempDir, "test.md"):                        "# Test",
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", path, err)
		}
	}

	// Run function
	if err := MoveFilesIntoMatchingFolders(tempDir); err != nil {
		t.Fatalf("MoveFilesIntoMatchingFolders failed: %v", err)
	}

	// Expected result:
	// ├── hangeul-gyeongro-haha.md   <- stays
	// ├── meomeideu/
	// │   ├── meomeideu.md           <- moved here
	// │   └── teseuteu-hangeul.md    <- stays
	// ├── _metadata.json             <- stays
	// └── test.md                    <- stays

	// Verify meomeideu.md moved
	movedFile := filepath.Join(meomeideuDir, "meomeideu.md")
	if _, err := os.Stat(movedFile); os.IsNotExist(err) {
		t.Errorf("meomeideu.md should be moved to %s", movedFile)
	}

	originalMeomeideu := filepath.Join(tempDir, "meomeideu.md")
	if _, err := os.Stat(originalMeomeideu); !os.IsNotExist(err) {
		t.Errorf("original meomeideu.md should be removed")
	}

	// Verify other files stay
	stayFiles := []string{
		filepath.Join(tempDir, "hangeul-gyeongro-haha.md"),
		filepath.Join(meomeideuDir, "teseuteu-hangeul.md"),
		filepath.Join(tempDir, "_metadata.json"),
		filepath.Join(tempDir, "test.md"),
	}

	for _, f := range stayFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("file should remain: %s", f)
		}
	}
}

// TestMoveFilesIntoMatchingFolders_EmptyDir tests with empty directory (no subdirs)
func TestMoveFilesIntoMatchingFolders_EmptyDir(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test-move-files-empty-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Run function on empty directory
	if err := MoveFilesIntoMatchingFolders(tempDir); err != nil {
		t.Fatalf("MoveFilesIntoMatchingFolders failed on empty dir: %v", err)
	}
}

// TestMergeKoreanFoldersIntoRomanized_Basic tests basic merging of Korean folder contents
func TestMergeKoreanFoldersIntoRomanized_Basic(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test-merge-korean-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup structure (the problem case from modify.md):
	// ├── meomeideu/
	// │   ├── meomeideu.md
	// │   └── teseuteu-hangeul.md
	// └── 머메이드/
	//     └── files/
	//         └── image.png

	meomeideuDir := filepath.Join(tempDir, "meomeideu")
	koreanDir := filepath.Join(tempDir, "머메이드")
	filesDir := filepath.Join(koreanDir, "files")

	if err := os.MkdirAll(meomeideuDir, 0755); err != nil {
		t.Fatalf("failed to create meomeideu dir: %v", err)
	}
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatalf("failed to create files dir: %v", err)
	}

	// Create files in romanized folder
	if err := os.WriteFile(filepath.Join(meomeideuDir, "meomeideu.md"), []byte("# Meomeideu"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(meomeideuDir, "teseuteu-hangeul.md"), []byte("# Test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Create files in Korean folder
	if err := os.WriteFile(filepath.Join(filesDir, "image.png"), []byte("PNG DATA"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run function
	if err := MergeKoreanFoldersIntoRomanized(tempDir); err != nil {
		t.Fatalf("MergeKoreanFoldersIntoRomanized failed: %v", err)
	}

	// Expected result:
	// ├── meomeideu/
	// │   ├── files/
	// │   │   └── image.png
	// │   ├── meomeideu.md
	// │   └── teseuteu-hangeul.md

	// Verify files folder was moved to romanized location
	movedFilesDir := filepath.Join(meomeideuDir, "files")
	if _, err := os.Stat(movedFilesDir); os.IsNotExist(err) {
		t.Errorf("files folder should be moved to %s", movedFilesDir)
	}

	// Verify image.png exists in new location
	movedImage := filepath.Join(movedFilesDir, "image.png")
	if _, err := os.Stat(movedImage); os.IsNotExist(err) {
		t.Errorf("image.png should exist at %s", movedImage)
	}

	// Verify content
	content, err := os.ReadFile(movedImage)
	if err != nil {
		t.Fatalf("failed to read moved image: %v", err)
	}
	if string(content) != "PNG DATA" {
		t.Errorf("image content mismatch")
	}

	// Verify Korean folder was removed
	if _, err := os.Stat(koreanDir); !os.IsNotExist(err) {
		t.Errorf("Korean folder %s should be removed after merge", koreanDir)
	}

	// Verify existing files in romanized folder still exist
	if _, err := os.Stat(filepath.Join(meomeideuDir, "meomeideu.md")); os.IsNotExist(err) {
		t.Errorf("meomeideu.md should still exist")
	}
	if _, err := os.Stat(filepath.Join(meomeideuDir, "teseuteu-hangeul.md")); os.IsNotExist(err) {
		t.Errorf("teseuteu-hangeul.md should still exist")
	}
}

// TestMergeKoreanFoldersIntoRomanized_MultipleFiles tests merging with multiple files
func TestMergeKoreanFoldersIntoRomanized_MultipleFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-merge-korean-multi-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup:
	// ├── meomeideu/
	// │   └── existing.md
	// └── 머메이드/
	//     ├── files/
	//     │   ├── image1.png
	//     │   └── image2.png
	//     └── assets/
	//         └── style.css

	meomeideuDir := filepath.Join(tempDir, "meomeideu")
	koreanDir := filepath.Join(tempDir, "머메이드")
	filesDir := filepath.Join(koreanDir, "files")
	assetsDir := filepath.Join(koreanDir, "assets")

	for _, dir := range []string{meomeideuDir, filesDir, assetsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	// Create files
	testFiles := map[string]string{
		filepath.Join(meomeideuDir, "existing.md"):  "# Existing",
		filepath.Join(filesDir, "image1.png"):       "PNG1",
		filepath.Join(filesDir, "image2.png"):       "PNG2",
		filepath.Join(assetsDir, "style.css"):       "body {}",
	}

	for path, content := range testFiles {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", path, err)
		}
	}

	// Run function
	if err := MergeKoreanFoldersIntoRomanized(tempDir); err != nil {
		t.Fatalf("MergeKoreanFoldersIntoRomanized failed: %v", err)
	}

	// Verify all files moved correctly
	expectedFiles := []string{
		filepath.Join(meomeideuDir, "existing.md"),
		filepath.Join(meomeideuDir, "files", "image1.png"),
		filepath.Join(meomeideuDir, "files", "image2.png"),
		filepath.Join(meomeideuDir, "assets", "style.css"),
	}

	for _, f := range expectedFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("expected file to exist: %s", f)
		}
	}

	// Verify Korean folder removed
	if _, err := os.Stat(koreanDir); !os.IsNotExist(err) {
		t.Errorf("Korean folder should be removed")
	}
}

// TestMergeKoreanFoldersIntoRomanized_NoRomanizedFolder tests when romanized folder doesn't exist
func TestMergeKoreanFoldersIntoRomanized_NoRomanizedFolder(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-merge-korean-noroman-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup: Only Korean folder exists, no romanized counterpart
	// └── 머메이드/
	//     └── files/
	//         └── image.png

	koreanDir := filepath.Join(tempDir, "머메이드")
	filesDir := filepath.Join(koreanDir, "files")

	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(filesDir, "image.png"), []byte("PNG"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Run function
	if err := MergeKoreanFoldersIntoRomanized(tempDir); err != nil {
		t.Fatalf("MergeKoreanFoldersIntoRomanized failed: %v", err)
	}

	// Korean folder should still exist (no romanized folder to merge into)
	if _, err := os.Stat(koreanDir); os.IsNotExist(err) {
		t.Errorf("Korean folder should remain when no romanized counterpart exists")
	}

	// Files should still exist in Korean folder
	if _, err := os.Stat(filepath.Join(filesDir, "image.png")); os.IsNotExist(err) {
		t.Errorf("image.png should still exist in Korean folder")
	}
}

// TestContainsKorean tests the containsKorean helper function
func TestContainsKorean(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"머메이드", true},
		{"한글", true},
		{"meomeideu", false},
		{"test123", false},
		{"mixed머메이드text", true},
		{"", false},
		{"日本語", false}, // Japanese should not match
		{"中文", false},   // Chinese should not match
	}

	for _, tc := range tests {
		result := containsKorean(tc.input)
		if result != tc.expected {
			t.Errorf("containsKorean(%q) = %v, expected %v", tc.input, result, tc.expected)
		}
	}
}

package tests

import (
	"os"
	"path/filepath"
	"testing"

	pepeunit "github.com/w7an1y4a/pepeunit_go_client"
)

func TestFileManagerFileExists(t *testing.T) {
	fm := pepeunit.NewFileManager()

	// Create a temporary test file
	tempFile, err := CreateTempTestFile(TestEnvData())
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer CleanupTempFile(tempFile)

	// Test existing file
	exists := fm.FileExists(tempFile)
	if !exists {
		t.Error("Expected temp file to exist")
	}

	// Test non-existing file
	exists = fm.FileExists("nonexistent.json")
	if exists {
		t.Error("Expected nonexistent.json to not exist")
	}
}

func TestFileManagerReadJSON(t *testing.T) {
	fm := pepeunit.NewFileManager()

	// Create a temporary test file
	tempFile, err := CreateTempTestFile(TestEnvData())
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer CleanupTempFile(tempFile)

	data, err := fm.ReadJSON(tempFile)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	expectedData := TestEnvData()
	if data["PEPEUNIT_URL"] != expectedData["PEPEUNIT_URL"] {
		t.Errorf("Expected PEPEUNIT_URL to be '%v', got '%v'", expectedData["PEPEUNIT_URL"], data["PEPEUNIT_URL"])
	}
}

func TestFileManagerReadJSONNonExistent(t *testing.T) {
	fm := pepeunit.NewFileManager()

	_, err := fm.ReadJSON("nonexistent.json")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestFileManagerWriteJSON(t *testing.T) {
	fm := pepeunit.NewFileManager()

	testData := map[string]interface{}{
		"test_key": "test_value",
		"number":   42,
	}

	tempFile := "test_write.json"
	defer os.Remove(tempFile)

	err := fm.WriteJSON(tempFile, testData)
	if err != nil {
		t.Fatalf("Failed to write JSON: %v", err)
	}

	// Verify by reading back
	data, err := fm.ReadJSON(tempFile)
	if err != nil {
		t.Fatalf("Failed to read back JSON: %v", err)
	}

	if data["test_key"] != "test_value" {
		t.Errorf("Expected test_key to be 'test_value', got '%v'", data["test_key"])
	}
}

func TestFileManagerCopyFile(t *testing.T) {
	fm := pepeunit.NewFileManager()

	// Create a temporary test file
	sourceFile, err := CreateTempTestFile(TestEnvData())
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer CleanupTempFile(sourceFile)

	destFile := "test_copy.json"
	defer os.Remove(destFile)

	err = fm.CopyFile(sourceFile, destFile)
	if err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}

	// Verify files are identical
	sourceData, err := fm.ReadJSON(sourceFile)
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}

	destData, err := fm.ReadJSON(destFile)
	if err != nil {
		t.Fatalf("Failed to read dest file: %v", err)
	}

	if sourceData["PEPEUNIT_URL"] != destData["PEPEUNIT_URL"] {
		t.Error("Copied file content doesn't match source")
	}
}

func TestFileManagerCopyDirectoryContents(t *testing.T) {
	fm := pepeunit.NewFileManager()

	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "test_source_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files in the temp directory
	envFile := filepath.Join(tempDir, "env.json")
	schemaFile := filepath.Join(tempDir, "schema.json")

	envData := TestEnvData()
	schemaData := TestSchemaData()

	// Write test data to files
	if err := fm.WriteJSON(envFile, envData); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}
	if err := fm.WriteJSON(schemaFile, schemaData); err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	destDir := "test_copy_dir"
	defer os.RemoveAll(destDir)

	err = fm.CopyDirectoryContents(tempDir, destDir)
	if err != nil {
		t.Fatalf("Failed to copy directory contents: %v", err)
	}

	// Verify files were copied
	destEnv := filepath.Join(destDir, "env.json")
	destSchema := filepath.Join(destDir, "schema.json")

	if !fm.FileExists(destEnv) {
		t.Error("Expected env.json to be copied")
	}
	if !fm.FileExists(destSchema) {
		t.Error("Expected schema.json to be copied")
	}

	// Verify content
	sourceData, err := fm.ReadJSON(envFile)
	if err != nil {
		t.Fatalf("Failed to read source env.json: %v", err)
	}

	destData, err := fm.ReadJSON(destEnv)
	if err != nil {
		t.Fatalf("Failed to read dest env.json: %v", err)
	}

	if sourceData["PEPEUNIT_URL"] != destData["PEPEUNIT_URL"] {
		t.Error("Copied env.json content doesn't match source")
	}
}

func TestFileManagerExtractTarGz(t *testing.T) {
	fm := pepeunit.NewFileManager()

	// Create a test tar.gz file
	testDir := "test_extract_source"
	testArchive := "test_extract.tar.gz"
	extractDir := "test_extract_dest"

	defer os.RemoveAll(testDir)
	defer os.RemoveAll(testArchive)
	defer os.RemoveAll(extractDir)

	// Create source directory with test files
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create tar.gz archive
	err = fm.CreateTarGz(testDir, testArchive)
	if err != nil {
		t.Fatalf("Failed to create tar.gz: %v", err)
	}

	// Extract archive
	err = fm.ExtractTarGz(testArchive, extractDir)
	if err != nil {
		t.Fatalf("Failed to extract tar.gz: %v", err)
	}

	// Verify extraction
	extractedFile := filepath.Join(extractDir, "test.txt")
	if !fm.FileExists(extractedFile) {
		t.Error("Expected test.txt to be extracted")
	}

	// Verify content
	content, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	if string(content) != "test content" {
		t.Errorf("Expected content 'test content', got '%s'", string(content))
	}
}

func TestFileManagerCreateTarGz(t *testing.T) {
	fm := pepeunit.NewFileManager()

	testDir := "test_create_source"
	testArchive := "test_create.tar.gz"

	defer os.RemoveAll(testDir)
	defer os.RemoveAll(testArchive)

	// Create source directory with test files
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create tar.gz archive
	err = fm.CreateTarGz(testDir, testArchive)
	if err != nil {
		t.Fatalf("Failed to create tar.gz: %v", err)
	}

	// Verify archive was created
	if !fm.FileExists(testArchive) {
		t.Error("Expected tar.gz archive to be created")
	}
}

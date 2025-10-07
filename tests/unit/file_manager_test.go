package pepeunit_test

import (
	"os"
	"path/filepath"
	"testing"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

func TestFileManager_AppendToJSONList_ArrayAndEntriesObject(t *testing.T) {
	dir := t.TempDir()
	pathArr := filepath.Join(dir, "arr.json")
	pathObj := filepath.Join(dir, "obj.json")

	fm := pepeunit.NewFileManager()

	// array path (non-existing -> create array)
	if err := fm.AppendToJSONList(pathArr, map[string]interface{}{"a": 1}); err != nil {
		t.Fatalf("append array: %v", err)
	}
	data, _ := os.ReadFile(pathArr)
	if string(data) == "" || data[0] != '[' {
		t.Fatalf("expected array json written")
	}

	// entries object path
	_ = os.WriteFile(pathObj, []byte("{\"entries\": []}"), 0644)
	if err := fm.AppendToJSONList(pathObj, map[string]interface{}{"b": 2}); err != nil {
		t.Fatalf("append entries object: %v", err)
	}
	data2, _ := os.ReadFile(pathObj)
	if len(data2) == 0 || data2[0] != '{' {
		t.Fatalf("expected object json written")
	}
}

func TestFileManager_CopyDirectory_TarRoundtrip(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	dstDir := filepath.Join(dir, "dst")
	_ = os.MkdirAll(srcDir, 0755)
	_ = os.MkdirAll(dstDir, 0755)

	file1 := filepath.Join(srcDir, "f1.txt")
	_ = os.WriteFile(file1, []byte("hello"), 0644)

	fm := pepeunit.NewFileManager()
	if err := fm.CopyDirectoryContents(srcDir, dstDir); err != nil {
		t.Fatalf("copy dir: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dstDir, "f1.txt"))
	if err != nil || string(b) != "hello" {
		t.Fatalf("copied file mismatch")
	}

	// tar.gz roundtrip
	archive := filepath.Join(dir, "a.tar.gz")
	if err := fm.CreateTarGz(srcDir, archive); err != nil {
		t.Fatalf("create tar.gz: %v", err)
	}
	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("mkdir out: %v", err)
	}
	if err := fm.ExtractTarGz(archive, outDir); err != nil {
		t.Fatalf("extract tar.gz: %v", err)
	}
	b2, err := os.ReadFile(filepath.Join(outDir, "f1.txt"))
	if err != nil || string(b2) != "hello" {
		t.Fatalf("extracted file mismatch")
	}
}

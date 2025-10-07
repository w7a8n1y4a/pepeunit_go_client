package pepeunit_test

import (
	"os"
	"path/filepath"
	"testing"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

func TestLogger_ResetLog_And_LoadEntriesObject(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "log.json")

	_ = os.WriteFile(logPath, []byte("{\"entries\": [{\"create_datetime\": \"t\", \"level\": \"Info\", \"text\": \"x\"}]}"), 0644)

	sm, _ := pepeunit.NewSchemaManager(filepath.Join(dir, "schema.json"))
	settings := pepeunit.NewSettings("")
	logger := pepeunit.NewLogger(logPath, nil, sm, settings)

	full := logger.GetFullLog()
	if len(full) != 1 {
		t.Fatalf("expected 1 entry loaded from entries object")
	}

	logger.ResetLog()
	full2 := logger.GetFullLog()
	if len(full2) != 0 {
		t.Fatalf("expected empty after reset")
	}
}

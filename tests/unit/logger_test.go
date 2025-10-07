package pepeunit_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

func TestLogger_WriteAndRead_LogLevel_FilterAndMQTTPublish(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "log.json")
	schemaPath := filepath.Join(dir, "schema.json")

	// schema with log topic
	schemaData := map[string]interface{}{
		"output_base_topic": map[string]interface{}{
			"log/pepeunit": []interface{}{"unit/abc/log"},
		},
	}
	fm := pepeunit.NewFileManager()
	if err := fm.WriteJSON(schemaPath, schemaData); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	sm, err := pepeunit.NewSchemaManager(schemaPath)
	if err != nil {
		t.Fatalf("schema: %v", err)
	}

	settings := pepeunit.NewSettings("")
	settings.MINIMAL_LOG_LEVEL = string(pepeunit.LogLevelInfo)

	mqtt := &mockMQTTClient{}
	logger := pepeunit.NewLogger(logPath, mqtt, sm, settings)

	logger.Debug("d")   // filtered out
	logger.Info("i1")   // published
	logger.Warning("w") // published

	// read file as array
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	var arr []map[string]interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 entries (info+warning), got %d", len(arr))
	}

	if len(mqtt.published) == 0 {
		t.Fatalf("expected mqtt publications")
	}

	full := logger.GetFullLog()
	if len(full) != 2 {
		t.Fatalf("expected 2 full entries")
	}
}

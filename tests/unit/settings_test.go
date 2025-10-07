package pepeunit_test

import (
	"os"
	"path/filepath"
	"testing"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

func TestSettings_LoadFromFile_And_All(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "env.json")

	content := map[string]interface{}{
		"PEPEUNIT_URL": "example.com",
		"MQTT_PORT":    1884,
		"EXTRA_KEY":    "extra",
	}
	if err := writeJSON(envPath, content); err != nil {
		t.Fatalf("write json: %v", err)
	}

	s := pepeunit.NewSettings(envPath)
	if s.PEPEUNIT_URL != "example.com" {
		t.Fatalf("expected url set, got %q", s.PEPEUNIT_URL)
	}
	if s.MQTT_PORT != 1884 {
		t.Fatalf("expected port 1884, got %d", s.MQTT_PORT)
	}
	if v, ok := s.Get("EXTRA_KEY"); !ok || v.(string) != "extra" {
		t.Fatalf("expected extra key present")
	}

	all := s.All()
	if all["PEPEUNIT_URL"].(string) != "example.com" {
		t.Fatalf("all should contain value")
	}
}

func TestSettings_Update_Set_Get(t *testing.T) {
	s := pepeunit.NewSettings("")
	_ = s.Update(map[string]interface{}{"PEPEUNIT_URL": "u", "MQTT_PORT": 1999})
	if s.PEPEUNIT_URL != "u" || s.MQTT_PORT != 1999 {
		t.Fatalf("update failed")
	}
	s.Set("EXTRA", 42)
	if v, ok := s.Get("EXTRA"); !ok || v.(int) != 42 {
		t.Fatalf("set/get extra failed")
	}
}

func TestSettings_UpdateEnvFile(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "env.json")
	_ = os.WriteFile(envPath, []byte("{}"), 0644)

	newEnvPath := filepath.Join(dir, "env_new.json")
	if err := writeJSON(newEnvPath, map[string]interface{}{"PEPEUNIT_URL": "u2"}); err != nil {
		t.Fatalf("write: %v", err)
	}

	s := pepeunit.NewSettings(envPath)
	if err := s.UpdateEnvFile(newEnvPath); err != nil {
		t.Fatalf("update env file: %v", err)
	}
	if s.PEPEUNIT_URL != "u2" {
		t.Fatalf("expected u2, got %q", s.PEPEUNIT_URL)
	}
}

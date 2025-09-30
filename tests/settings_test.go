package tests

import (
	"testing"

	pepeunit "github.com/w7an1y4a/pepeunit_go_client"
)

func TestNewSettings(t *testing.T) {
	// Create a temporary test file
	tempFile, err := CreateTempTestFile(TestEnvData())
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer CleanupTempFile(tempFile)

	// Test with valid file
	settings := pepeunit.NewSettings(tempFile)
	if settings == nil {
		t.Fatal("Expected settings instance, got nil")
	}

	// Check if values are loaded
	expectedData := TestEnvData()
	if settings.PEPEUNIT_URL != expectedData["PEPEUNIT_URL"] {
		t.Errorf("Expected PEPEUNIT_URL to be '%v', got '%s'", expectedData["PEPEUNIT_URL"], settings.PEPEUNIT_URL)
	}

	// Handle both int and float64 types from JSON
	var expectedPort int
	switch v := expectedData["MQTT_PORT"].(type) {
	case int:
		expectedPort = v
	case float64:
		expectedPort = int(v)
	default:
		t.Fatalf("Unexpected type for MQTT_PORT: %T", v)
	}

	if settings.MQTT_PORT != expectedPort {
		t.Errorf("Expected MQTT_PORT to be %d, got %d", expectedPort, settings.MQTT_PORT)
	}
}

func TestNewSettingsWithEmptyPath(t *testing.T) {
	settings := pepeunit.NewSettings("")
	if settings == nil {
		t.Fatal("Expected settings instance, got nil")
	}

	// Should have default values
	if settings.HTTP_TYPE != "https" {
		t.Errorf("Expected HTTP_TYPE to be 'https', got '%s'", settings.HTTP_TYPE)
	}

	if settings.MQTT_PORT != 1883 {
		t.Errorf("Expected MQTT_PORT to be 1883, got %d", settings.MQTT_PORT)
	}
}

func TestSettingsLoadFromFile(t *testing.T) {
	// Create a temporary test file
	tempFile, err := CreateTempTestFile(TestEnvData())
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer CleanupTempFile(tempFile)

	settings := &pepeunit.Settings{EnvFilePath: tempFile}

	err = settings.LoadFromFile()
	if err != nil {
		t.Fatalf("Failed to load from file: %v", err)
	}

	expectedData := TestEnvData()
	if settings.PEPEUNIT_URL != expectedData["PEPEUNIT_URL"] {
		t.Errorf("Expected PEPEUNIT_URL to be '%v', got '%s'", expectedData["PEPEUNIT_URL"], settings.PEPEUNIT_URL)
	}
}

func TestSettingsLoadFromNonExistentFile(t *testing.T) {
	settings := &pepeunit.Settings{EnvFilePath: "nonexistent.json"}

	err := settings.LoadFromFile()
	if err != nil {
		// This is expected behavior - should return error for non-existent file
		t.Logf("Expected error for non-existent file: %v", err)
	}
}

func TestSettingsGetEnvValues(t *testing.T) {
	// Create a temporary test file
	tempFile, err := CreateTempTestFile(TestEnvData())
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer CleanupTempFile(tempFile)

	settings := &pepeunit.Settings{EnvFilePath: tempFile}

	values, err := settings.GetEnvValues()
	if err != nil {
		t.Fatalf("Failed to get env values: %v", err)
	}

	expectedData := TestEnvData()
	if values["PEPEUNIT_URL"] != expectedData["PEPEUNIT_URL"] {
		t.Errorf("Expected PEPEUNIT_URL to be '%v', got '%v'", expectedData["PEPEUNIT_URL"], values["PEPEUNIT_URL"])
	}
}

func TestSettingsUpdate(t *testing.T) {
	settings := pepeunit.NewSettings("")

	updates := map[string]interface{}{
		"PEPEUNIT_URL": "updated.com",
		"MQTT_PORT":    8883,
	}

	err := settings.Update(updates)
	if err != nil {
		t.Fatalf("Failed to update settings: %v", err)
	}

	if settings.PEPEUNIT_URL != "updated.com" {
		t.Errorf("Expected PEPEUNIT_URL to be 'updated.com', got '%s'", settings.PEPEUNIT_URL)
	}

	if settings.MQTT_PORT != 8883 {
		t.Errorf("Expected MQTT_PORT to be 8883, got %d", settings.MQTT_PORT)
	}
}

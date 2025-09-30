package tests

import (
	"encoding/json"
	"os"
)

// TestEnvData returns sample environment data for testing
func TestEnvData() map[string]interface{} {
	return map[string]interface{}{
		"DELAY_PUB_MSG":              1,
		"PEPEUNIT_URL":               "devunit.pepeunit.com",
		"HTTP_TYPE":                  "https",
		"PEPEUNIT_APP_PREFIX":        "/pepeunit",
		"PEPEUNIT_API_ACTUAL_PREFIX": "/api/v1",
		"MQTT_URL":                   "devemqx.pepemoss.com",
		"MQTT_PORT":                  1884,
		"PEPEUNIT_TOKEN":             "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1dWlkIjoiYWQ0ZTQwMTQtOTg4ZC00ZjVlLTgzYTItMDgxZDNmZDllZTkwIiwidHlwZSI6IlVuaXQifQ.KPG7YFqI_VRmt-i80VeqSDpRFApLytrzdxKrvMU8Otc",
		"SYNC_ENCRYPT_KEY":           "OCsMR6ETVfZayCdBqEmxug==",
		"SECRET_KEY":                 "I4x4YwnNAUBQfIopSFCvvQ==",
		"PING_INTERVAL":              30,
		"STATE_SEND_INTERVAL":        2,
		"MINIMAL_LOG_LEVEL":          "Debug",
		"COMMIT_VERSION":             "d4218da85305798c783a51dd25944bfa0b40903b",
	}
}

// TestSchemaData returns sample schema data for testing
func TestSchemaData() map[string]interface{} {
	return map[string]interface{}{
		"input_base_topic": map[string]interface{}{
			"update/pepeunit": []interface{}{
				"devunit.pepeunit.com/input_base_topic/ad4e4014-988d-4f5e-83a2-081d3fd9ee90/update/pepeunit",
			},
			"env_update/pepeunit": []interface{}{
				"devunit.pepeunit.com/input_base_topic/ad4e4014-988d-4f5e-83a2-081d3fd9ee90/env_update/pepeunit",
			},
			"schema_update/pepeunit": []interface{}{
				"devunit.pepeunit.com/input_base_topic/ad4e4014-988d-4f5e-83a2-081d3fd9ee90/schema_update/pepeunit",
			},
			"log_sync/pepeunit": []interface{}{
				"devunit.pepeunit.com/input_base_topic/ad4e4014-988d-4f5e-83a2-081d3fd9ee90/log_sync/pepeunit",
			},
		},
		"output_base_topic": map[string]interface{}{
			"state/pepeunit": []interface{}{
				"devunit.pepeunit.com/output_base_topic/ad4e4014-988d-4f5e-83a2-081d3fd9ee90/state/pepeunit",
			},
			"log/pepeunit": []interface{}{
				"devunit.pepeunit.com/output_base_topic/ad4e4014-988d-4f5e-83a2-081d3fd9ee90/log/pepeunit",
			},
		},
		"input_topic": map[string]interface{}{
			"input/pepeunit": []interface{}{
				"devunit.pepeunit.com/486bfea4-a056-4c15-a361-7bbe5fde66ae/pepeunit",
			},
		},
		"output_topic": map[string]interface{}{
			"output/pepeunit": []interface{}{
				"devunit.pepeunit.com/895e0be1-7121-4e5c-9d84-b79d9cb8b1cb/pepeunit",
			},
		},
	}
}

// CreateTempTestFile creates a temporary file with the given data and returns its path
func CreateTempTestFile(data map[string]interface{}) (string, error) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test_*.json")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Write JSON data to the file
	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

// CleanupTempFile removes a temporary file
func CleanupTempFile(filePath string) {
	if filePath != "" {
		os.Remove(filePath)
	}
}

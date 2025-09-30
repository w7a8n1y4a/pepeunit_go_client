package pepeunit

import (
	"fmt"
	"strconv"
)

// Settings manages configuration settings
type Settings struct {
	EnvFilePath                string
	PEPEUNIT_URL               string
	PEPEUNIT_APP_PREFIX        string
	PEPEUNIT_API_ACTUAL_PREFIX string
	HTTP_TYPE                  string
	MQTT_URL                   string
	MQTT_PORT                  int
	PEPEUNIT_TOKEN             string
	SYNC_ENCRYPT_KEY           string
	SECRET_KEY                 string
	COMMIT_VERSION             string
	PING_INTERVAL              int
	STATE_SEND_INTERVAL        int
	MINIMAL_LOG_LEVEL          string
	DELAY_PUB_MSG              int
}

// NewSettings creates a new settings instance
func NewSettings(envFilePath string) *Settings {
	settings := &Settings{
		EnvFilePath:                envFilePath,
		PEPEUNIT_URL:               "",
		PEPEUNIT_APP_PREFIX:        "",
		PEPEUNIT_API_ACTUAL_PREFIX: "",
		HTTP_TYPE:                  "https",
		MQTT_URL:                   "",
		MQTT_PORT:                  1883,
		PEPEUNIT_TOKEN:             "",
		SYNC_ENCRYPT_KEY:           "",
		SECRET_KEY:                 "",
		COMMIT_VERSION:             "",
		PING_INTERVAL:              30,
		STATE_SEND_INTERVAL:        300,
		MINIMAL_LOG_LEVEL:          "Debug",
		DELAY_PUB_MSG:              300,
	}

	if envFilePath != "" {
		settings.LoadFromFile()
	}

	return settings
}

// LoadFromFile loads settings from the environment file
func (s *Settings) LoadFromFile() error {
	if s.EnvFilePath == "" {
		return nil
	}

	fm := NewFileManager()
	if !fm.FileExists(s.EnvFilePath) {
		return fmt.Errorf("environment file does not exist: %s", s.EnvFilePath)
	}

	envData, err := fm.ReadJSON(s.EnvFilePath)
	if err != nil {
		return err
	}

	return s.updateFromMap(envData)
}

// updateFromMap updates settings from a map of values
func (s *Settings) updateFromMap(data map[string]interface{}) error {
	for key, value := range data {
		switch key {
		case "PEPEUNIT_URL":
			s.PEPEUNIT_URL = toString(value)
		case "PEPEUNIT_APP_PREFIX":
			s.PEPEUNIT_APP_PREFIX = toString(value)
		case "PEPEUNIT_API_ACTUAL_PREFIX":
			s.PEPEUNIT_API_ACTUAL_PREFIX = toString(value)
		case "HTTP_TYPE":
			s.HTTP_TYPE = toString(value)
		case "MQTT_URL":
			s.MQTT_URL = toString(value)
		case "MQTT_PORT":
			s.MQTT_PORT = toInt(value)
		case "PEPEUNIT_TOKEN":
			s.PEPEUNIT_TOKEN = toString(value)
		case "SYNC_ENCRYPT_KEY":
			s.SYNC_ENCRYPT_KEY = toString(value)
		case "SECRET_KEY":
			s.SECRET_KEY = toString(value)
		case "COMMIT_VERSION":
			s.COMMIT_VERSION = toString(value)
		case "PING_INTERVAL":
			s.PING_INTERVAL = toInt(value)
		case "STATE_SEND_INTERVAL":
			s.STATE_SEND_INTERVAL = toInt(value)
		case "MINIMAL_LOG_LEVEL":
			s.MINIMAL_LOG_LEVEL = toString(value)
		case "DELAY_PUB_MSG":
			s.DELAY_PUB_MSG = toInt(value)
		}
	}
	return nil
}

// GetEnvValues returns all environment values as a map
func (s *Settings) GetEnvValues() (map[string]interface{}, error) {
	if s.EnvFilePath == "" {
		return map[string]interface{}{}, nil
	}

	fm := NewFileManager()
	if !fm.FileExists(s.EnvFilePath) {
		return map[string]interface{}{}, nil
	}

	return fm.ReadJSON(s.EnvFilePath)
}

// UpdateEnvFile updates the environment file with a new one
func (s *Settings) UpdateEnvFile(newEnvFilePath string) error {
	if s.EnvFilePath == "" {
		return fmt.Errorf("env file path not set")
	}

	fm := NewFileManager()
	err := fm.CopyFile(newEnvFilePath, s.EnvFilePath)
	if err != nil {
		return err
	}

	return s.LoadFromFile()
}

// Update updates specific settings
func (s *Settings) Update(updates map[string]interface{}) error {
	return s.updateFromMap(updates)
}

// Helper functions for type conversion
func toString(value interface{}) string {
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", value)
}

func toInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

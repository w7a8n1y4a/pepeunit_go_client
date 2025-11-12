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
	MIN_LOG_LEVEL              string
	MAX_LOG_LENGTH             int
	extras                     map[string]interface{}
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
		MIN_LOG_LEVEL:              "Debug",
		MAX_LOG_LENGTH:             64,
		extras:                     map[string]interface{}{},
	}

	if envFilePath != "" {
		settings.LoadFromFile()
	}

	return settings
}

func NewSettingsWith(envFilePath string, kwargs map[string]interface{}) *Settings {
	s := NewSettings(envFilePath)
	if len(kwargs) > 0 {
		_ = s.Update(kwargs)
	}
	return s
}

// LoadFromFile loads settings from the environment file
func (s *Settings) LoadFromFile() error {
	if s.EnvFilePath == "" {
		return nil
	}

	fm := NewFileManager()
	if !fm.FileExists(s.EnvFilePath) {
		// Align with Python client: missing env file is a no-op
		return nil
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
		case "MIN_LOG_LEVEL":
			s.MIN_LOG_LEVEL = toString(value)
		case "MINIMAL_LOG_LEVEL":
			s.MIN_LOG_LEVEL = toString(value)
		case "MAX_LOG_LENGTH":
			s.MAX_LOG_LENGTH = toInt(value)
		default:
			if s.extras == nil {
				s.extras = map[string]interface{}{}
			}
			s.extras[key] = value
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

func (s *Settings) Set(key string, value interface{}) {
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
	case "MIN_LOG_LEVEL":
		s.MIN_LOG_LEVEL = toString(value)
	case "MINIMAL_LOG_LEVEL":
		s.MIN_LOG_LEVEL = toString(value)
	case "MAX_LOG_LENGTH":
		s.MAX_LOG_LENGTH = toInt(value)
	default:
		if s.extras == nil {
			s.extras = map[string]interface{}{}
		}
		s.extras[key] = value
	}
}

func (s *Settings) Get(key string) (interface{}, bool) {
	switch key {
	case "PEPEUNIT_URL":
		return s.PEPEUNIT_URL, true
	case "PEPEUNIT_APP_PREFIX":
		return s.PEPEUNIT_APP_PREFIX, true
	case "PEPEUNIT_API_ACTUAL_PREFIX":
		return s.PEPEUNIT_API_ACTUAL_PREFIX, true
	case "HTTP_TYPE":
		return s.HTTP_TYPE, true
	case "MQTT_URL":
		return s.MQTT_URL, true
	case "MQTT_PORT":
		return s.MQTT_PORT, true
	case "PEPEUNIT_TOKEN":
		return s.PEPEUNIT_TOKEN, true
	case "SYNC_ENCRYPT_KEY":
		return s.SYNC_ENCRYPT_KEY, true
	case "SECRET_KEY":
		return s.SECRET_KEY, true
	case "COMMIT_VERSION":
		return s.COMMIT_VERSION, true
	case "PING_INTERVAL":
		return s.PING_INTERVAL, true
	case "STATE_SEND_INTERVAL":
		return s.STATE_SEND_INTERVAL, true
	case "MIN_LOG_LEVEL":
		return s.MIN_LOG_LEVEL, true
	case "MAX_LOG_LENGTH":
		return s.MAX_LOG_LENGTH, true
	default:
		if s.extras == nil {
			return nil, false
		}
		v, ok := s.extras[key]
		return v, ok
	}
}

func (s *Settings) GetString(key string) (string, bool) {
	v, ok := s.Get(key)
	if !ok {
		return "", false
	}
	return toString(v), true
}

func (s *Settings) GetInt(key string) (int, bool) {
	v, ok := s.Get(key)
	if !ok {
		return 0, false
	}
	return toInt(v), true
}

func (s *Settings) Has(key string) bool {
	_, ok := s.Get(key)
	return ok
}

func (s *Settings) All() map[string]interface{} {
	result := map[string]interface{}{
		"PEPEUNIT_URL":               s.PEPEUNIT_URL,
		"PEPEUNIT_APP_PREFIX":        s.PEPEUNIT_APP_PREFIX,
		"PEPEUNIT_API_ACTUAL_PREFIX": s.PEPEUNIT_API_ACTUAL_PREFIX,
		"HTTP_TYPE":                  s.HTTP_TYPE,
		"MQTT_URL":                   s.MQTT_URL,
		"MQTT_PORT":                  s.MQTT_PORT,
		"PEPEUNIT_TOKEN":             s.PEPEUNIT_TOKEN,
		"SYNC_ENCRYPT_KEY":           s.SYNC_ENCRYPT_KEY,
		"SECRET_KEY":                 s.SECRET_KEY,
		"COMMIT_VERSION":             s.COMMIT_VERSION,
		"PING_INTERVAL":              s.PING_INTERVAL,
		"STATE_SEND_INTERVAL":        s.STATE_SEND_INTERVAL,
		"MIN_LOG_LEVEL":              s.MIN_LOG_LEVEL,
		"MAX_LOG_LENGTH":             s.MAX_LOG_LENGTH,
	}
	for k, v := range s.extras {
		result[k] = v
	}
	return result
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

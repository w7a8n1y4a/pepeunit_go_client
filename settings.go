package pepeunit

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Settings manages configuration settings
type Settings struct {
	EnvFilePath            string
	PU_DOMAIN              string
	PU_APP_PREFIX          string
	PU_API_ACTUAL_PREFIX   string
	PU_HTTP_TYPE           string
	PU_MQTT_HOST           string
	PU_MQTT_PORT           int
	PU_AUTH_TOKEN          string
	PU_SECRET_KEY          string
	PU_ENCRYPT_KEY         string
	PU_COMMIT_VERSION      string
	PU_MQTT_PING_INTERVAL  int
	PU_STATE_SEND_INTERVAL int
	PU_MIN_LOG_LEVEL       string
	PU_MAX_LOG_LENGTH      int
	extras                 map[string]interface{}
}

// NewSettings creates a new settings instance
func NewSettings(envFilePath string) *Settings {
	settings := &Settings{
		EnvFilePath:            envFilePath,
		PU_DOMAIN:              "",
		PU_APP_PREFIX:          "",
		PU_API_ACTUAL_PREFIX:   "",
		PU_HTTP_TYPE:           "https",
		PU_MQTT_HOST:           "",
		PU_MQTT_PORT:           1883,
		PU_AUTH_TOKEN:          "",
		PU_SECRET_KEY:          "",
		PU_ENCRYPT_KEY:         "",
		PU_COMMIT_VERSION:      "",
		PU_MQTT_PING_INTERVAL:  30,
		PU_STATE_SEND_INTERVAL: 300,
		PU_MIN_LOG_LEVEL:       "Debug",
		PU_MAX_LOG_LENGTH:      64,
		extras:                 map[string]interface{}{},
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
		case "PU_DOMAIN":
			s.PU_DOMAIN = toString(value)
		case "PU_APP_PREFIX":
			s.PU_APP_PREFIX = toString(value)
		case "PU_API_ACTUAL_PREFIX":
			s.PU_API_ACTUAL_PREFIX = toString(value)
		case "PU_HTTP_TYPE":
			s.PU_HTTP_TYPE = toString(value)
		case "PU_MQTT_HOST":
			s.PU_MQTT_HOST = toString(value)
		case "PU_MQTT_PORT":
			s.PU_MQTT_PORT = toInt(value)
		case "PU_AUTH_TOKEN":
			s.PU_AUTH_TOKEN = toString(value)
		case "PU_SECRET_KEY":
			s.PU_SECRET_KEY = toString(value)
		case "PU_ENCRYPT_KEY":
			s.PU_ENCRYPT_KEY = toString(value)
		case "PU_COMMIT_VERSION":
			s.PU_COMMIT_VERSION = toString(value)
		case "PU_MQTT_PING_INTERVAL":
			s.PU_MQTT_PING_INTERVAL = toInt(value)
		case "PU_STATE_SEND_INTERVAL":
			s.PU_STATE_SEND_INTERVAL = toInt(value)
		case "PU_MIN_LOG_LEVEL":
			s.PU_MIN_LOG_LEVEL = toString(value)
		case "MINIMAL_LOG_LEVEL":
			s.PU_MIN_LOG_LEVEL = toString(value)
		case "PU_MAX_LOG_LENGTH":
			s.PU_MAX_LOG_LENGTH = toInt(value)
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
	case "PU_DOMAIN":
		s.PU_DOMAIN = toString(value)
	case "PU_APP_PREFIX":
		s.PU_APP_PREFIX = toString(value)
	case "PU_API_ACTUAL_PREFIX":
		s.PU_API_ACTUAL_PREFIX = toString(value)
	case "PU_HTTP_TYPE":
		s.PU_HTTP_TYPE = toString(value)
	case "PU_MQTT_HOST":
		s.PU_MQTT_HOST = toString(value)
	case "PU_MQTT_PORT":
		s.PU_MQTT_PORT = toInt(value)
	case "PU_AUTH_TOKEN":
		s.PU_AUTH_TOKEN = toString(value)
	case "PU_SECRET_KEY":
		s.PU_SECRET_KEY = toString(value)
	case "PU_ENCRYPT_KEY":
		s.PU_ENCRYPT_KEY = toString(value)
	case "PU_COMMIT_VERSION":
		s.PU_COMMIT_VERSION = toString(value)
	case "PU_MQTT_PING_INTERVAL":
		s.PU_MQTT_PING_INTERVAL = toInt(value)
	case "PU_STATE_SEND_INTERVAL":
		s.PU_STATE_SEND_INTERVAL = toInt(value)
	case "PU_MIN_LOG_LEVEL":
		s.PU_MIN_LOG_LEVEL = toString(value)
	case "MINIMAL_LOG_LEVEL":
		s.PU_MIN_LOG_LEVEL = toString(value)
	case "PU_MAX_LOG_LENGTH":
		s.PU_MAX_LOG_LENGTH = toInt(value)
	default:
		if s.extras == nil {
			s.extras = map[string]interface{}{}
		}
		s.extras[key] = value
	}
}

func (s *Settings) Get(key string) (interface{}, bool) {
	switch key {
	case "PU_DOMAIN":
		return s.PU_DOMAIN, true
	case "PU_APP_PREFIX":
		return s.PU_APP_PREFIX, true
	case "PU_API_ACTUAL_PREFIX":
		return s.PU_API_ACTUAL_PREFIX, true
	case "PU_HTTP_TYPE":
		return s.PU_HTTP_TYPE, true
	case "PU_MQTT_HOST":
		return s.PU_MQTT_HOST, true
	case "PU_MQTT_PORT":
		return s.PU_MQTT_PORT, true
	case "PU_AUTH_TOKEN":
		return s.PU_AUTH_TOKEN, true
	case "PU_SECRET_KEY":
		return s.PU_SECRET_KEY, true
	case "PU_ENCRYPT_KEY":
		return s.PU_ENCRYPT_KEY, true
	case "PU_COMMIT_VERSION":
		return s.PU_COMMIT_VERSION, true
	case "PU_MQTT_PING_INTERVAL":
		return s.PU_MQTT_PING_INTERVAL, true
	case "PU_STATE_SEND_INTERVAL":
		return s.PU_STATE_SEND_INTERVAL, true
	case "PU_MIN_LOG_LEVEL":
		return s.PU_MIN_LOG_LEVEL, true
	case "PU_MAX_LOG_LENGTH":
		return s.PU_MAX_LOG_LENGTH, true
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
		"PU_DOMAIN":              s.PU_DOMAIN,
		"PU_APP_PREFIX":          s.PU_APP_PREFIX,
		"PU_API_ACTUAL_PREFIX":   s.PU_API_ACTUAL_PREFIX,
		"PU_HTTP_TYPE":           s.PU_HTTP_TYPE,
		"PU_MQTT_HOST":           s.PU_MQTT_HOST,
		"PU_MQTT_PORT":           s.PU_MQTT_PORT,
		"PU_AUTH_TOKEN":          s.PU_AUTH_TOKEN,
		"PU_SECRET_KEY":          s.PU_SECRET_KEY,
		"PU_ENCRYPT_KEY":         s.PU_ENCRYPT_KEY,
		"PU_COMMIT_VERSION":      s.PU_COMMIT_VERSION,
		"PU_MQTT_PING_INTERVAL":  s.PU_MQTT_PING_INTERVAL,
		"PU_STATE_SEND_INTERVAL": s.PU_STATE_SEND_INTERVAL,
		"PU_MIN_LOG_LEVEL":       s.PU_MIN_LOG_LEVEL,
		"PU_MAX_LOG_LENGTH":      s.PU_MAX_LOG_LENGTH,
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

// UnitUUID extracts the unit UUID from the JWT token in settings
func (s *Settings) UnitUUID() (string, error) {
	tokenParts := strings.Split(s.PU_AUTH_TOKEN, ".")
	if len(tokenParts) != 3 {
		return "", fmt.Errorf("invalid JWT token format")
	}
	payload := tokenParts[1]
	for len(payload)%4 != 0 {
		payload += "="
	}
	decodedPayload, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("failed to decode JWT payload: %v", err)
	}
	var payloadData map[string]interface{}
	if err := json.Unmarshal(decodedPayload, &payloadData); err != nil {
		return "", fmt.Errorf("failed to parse JWT payload: %v", err)
	}
	uuidValue, ok := payloadData["uuid"]
	if !ok {
		return "", fmt.Errorf("UUID not found in JWT token")
	}
	uuidStr, ok := uuidValue.(string)
	if !ok {
		return "", fmt.Errorf("UUID is not a string")
	}
	return uuidStr, nil
}

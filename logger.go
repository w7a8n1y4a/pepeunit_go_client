package pepeunit

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp string `json:"create_datetime"`
	Level     string `json:"level"`
	Message   string `json:"text"`
}

// Logger handles logging operations
type Logger struct {
	logFilePath  string
	mqttClient   MQTTClient
	schema       *SchemaManager
	settings     *Settings
	logEntries   []LogEntry
	mutex        sync.RWMutex
	isPublishing bool // Flag to prevent recursive MQTT publishing
	fileManager  *FileManager
}

// NewLogger creates a new logger instance
func NewLogger(logFilePath string, mqttClient MQTTClient, schema *SchemaManager, settings *Settings) *Logger {
	logger := &Logger{
		logFilePath: logFilePath,
		mqttClient:  mqttClient,
		schema:      schema,
		settings:    settings,
		logEntries:  make([]LogEntry, 0),
		fileManager: NewFileManager(),
	}

	// Load existing log entries if file exists
	logger.loadExistingLogs()

	return logger
}

// loadExistingLogs loads existing log entries from file
func (l *Logger) loadExistingLogs() {
	if l.logFilePath == "" {
		return
	}

	if !l.fileManager.FileExists(l.logFilePath) {
		return
	}

	// Read raw JSON data to handle both array and object formats
	data, err := os.ReadFile(l.logFilePath)
	if err != nil {
		return
	}

	// Try to parse as array first (Python style)
	var directEntries []interface{}
	if err := json.Unmarshal(data, &directEntries); err == nil {
		l.loadEntriesFromArray(directEntries)
		return
	}

	// Try to parse as object with "entries" key (Go style)
	var logData map[string]interface{}
	if err := json.Unmarshal(data, &logData); err == nil {
		if wrappedEntries, ok := logData["entries"].([]interface{}); ok {
			l.loadEntriesFromArray(wrappedEntries)
		}
	}
}

// loadEntriesFromArray loads log entries from an array
func (l *Logger) loadEntriesFromArray(entries []interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	for _, entry := range entries {
		if entryMap, ok := entry.(map[string]interface{}); ok {
			// Handle different field names from Python vs Go
			var timestamp, level, message string

			// Try Python field names first (preferred format)
			if ts, exists := entryMap["create_datetime"]; exists {
				timestamp = toString(ts)
			} else if ts, exists := entryMap["timestamp"]; exists {
				timestamp = toString(ts)
			}

			if lvl, exists := entryMap["level"]; exists {
				level = toString(lvl)
			}

			if msg, exists := entryMap["text"]; exists {
				message = toString(msg)
			} else if msg, exists := entryMap["message"]; exists {
				message = toString(msg)
			}

			logEntry := LogEntry{
				Timestamp: timestamp,
				Level:     level,
				Message:   message,
			}
			l.logEntries = append(l.logEntries, logEntry)
		}
	}
}

// log writes a log entry with the specified level
func (l *Logger) log(level LogLevel, message string) {
	// Check if we should log this level first (like Python client)
	if !l.shouldPublishToMQTT(level) {
		return
	}

	// Use UTC time like Python client
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Create log entry with Python-compatible format
	logEntry := map[string]interface{}{
		"level":           string(level),
		"text":            message,
		"create_datetime": timestamp,
	}

	// Atomically append to file (like Python client)
	if l.logFilePath != "" {
		l.fileManager.AppendToJSONList(l.logFilePath, logEntry)
	}

	// Update in-memory cache
	entry := LogEntry{
		Timestamp: timestamp,
		Level:     string(level),
		Message:   message,
	}

	l.mutex.Lock()
	l.logEntries = append(l.logEntries, entry)
	l.mutex.Unlock()

	// Publish to MQTT if client and schema are available (like Python client)
	if l.mqttClient != nil && l.schema != nil {
		l.publishToMQTT(logEntry)
	}
}

// shouldPublishToMQTT checks if the log level should be published to MQTT
func (l *Logger) shouldPublishToMQTT(level LogLevel) bool {
	minLevel := LogLevel(l.settings.MINIMAL_LOG_LEVEL)
	return level.GetIntLevel() >= minLevel.GetIntLevel()
}

// publishToMQTT publishes log entry to MQTT
func (l *Logger) publishToMQTT(logEntry map[string]interface{}) {
	if l.schema == nil {
		return
	}

	// Prevent recursive publishing
	l.mutex.Lock()
	if l.isPublishing {
		l.mutex.Unlock()
		return
	}
	l.isPublishing = true
	l.mutex.Unlock()

	// Ensure we reset the flag when done
	defer func() {
		l.mutex.Lock()
		l.isPublishing = false
		l.mutex.Unlock()
	}()

	// Get output base topics and check if log topic exists
	outputBaseTopic := l.schema.GetOutputBaseTopic()
	if topics, ok := outputBaseTopic[string(BaseOutputTopicTypeLogPepeunit)]; ok && len(topics) > 0 {
		logJSON, err := json.Marshal(logEntry)
		if err != nil {
			return
		}

		// Publish to MQTT topic
		err = l.mqttClient.Publish(topics[0], string(logJSON))
		if err != nil {
			// Log error but don't fail the log operation
			// We can't use logger here as it would cause recursion
		}
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	l.log(LogLevelDebug, message)
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.log(LogLevelInfo, message)
}

// Warning logs a warning message
func (l *Logger) Warning(message string) {
	l.log(LogLevelWarning, message)
}

// Error logs an error message
func (l *Logger) Error(message string) {
	l.log(LogLevelError, message)
}

// Critical logs a critical message
func (l *Logger) Critical(message string) {
	l.log(LogLevelCritical, message)
}

// GetFullLog returns all log entries in Python-compatible format
func (l *Logger) GetFullLog() []map[string]interface{} {
	if l.logFilePath == "" {
		return []map[string]interface{}{}
	}

	if !l.fileManager.FileExists(l.logFilePath) {
		return []map[string]interface{}{}
	}

	// Read directly from file
	data, err := os.ReadFile(l.logFilePath)
	if err != nil {
		return []map[string]interface{}{}
	}

	var rawLogData []map[string]interface{}
	if err := json.Unmarshal(data, &rawLogData); err != nil {
		return []map[string]interface{}{}
	}

	// Convert to Python-compatible format
	result := make([]map[string]interface{}, 0, len(rawLogData))
	for _, entry := range rawLogData {
		// Create Python-compatible entry
		pythonEntry := make(map[string]interface{})

		// Handle text/message field
		if text, exists := entry["text"]; exists {
			pythonEntry["text"] = text
		} else if message, exists := entry["message"]; exists {
			pythonEntry["text"] = message
		}

		// Handle level field
		if level, exists := entry["level"]; exists {
			pythonEntry["level"] = level
		}

		// Handle timestamp field
		if createDatetime, exists := entry["create_datetime"]; exists {
			pythonEntry["create_datetime"] = createDatetime
		} else if timestamp, exists := entry["timestamp"]; exists {
			pythonEntry["create_datetime"] = timestamp
		}

		result = append(result, pythonEntry)
	}

	return result
}

// ResetLog clears all log entries
func (l *Logger) ResetLog() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.logEntries = make([]LogEntry, 0)
	if l.logFilePath != "" {
		l.fileManager.WriteJSON(l.logFilePath, []interface{}{})
	}
}

// SetMQTTClient sets the MQTT client for log publishing
func (l *Logger) SetMQTTClient(mqttClient MQTTClient) {
	l.mqttClient = mqttClient
}

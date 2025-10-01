package pepeunit

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
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
}

// NewLogger creates a new logger instance
func NewLogger(logFilePath string, mqttClient MQTTClient, schema *SchemaManager, settings *Settings) *Logger {
	logger := &Logger{
		logFilePath: logFilePath,
		mqttClient:  mqttClient,
		schema:      schema,
		settings:    settings,
		logEntries:  make([]LogEntry, 0),
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

	fm := NewFileManager()
	if !fm.FileExists(l.logFilePath) {
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

			// Try Go field names first
			if ts, exists := entryMap["timestamp"]; exists {
				timestamp = toString(ts)
			} else if ts, exists := entryMap["create_datetime"]; exists {
				timestamp = toString(ts)
			}

			if lvl, exists := entryMap["level"]; exists {
				level = toString(lvl)
			}

			if msg, exists := entryMap["message"]; exists {
				message = toString(msg)
			} else if msg, exists := entryMap["text"]; exists {
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

// saveLogs saves log entries to file
func (l *Logger) saveLogs() error {
	if l.logFilePath == "" {
		return nil
	}

	l.mutex.RLock()
	// Save as direct array like Python client, not wrapped in "entries"
	logData := make([]map[string]interface{}, len(l.logEntries))
	for i, entry := range l.logEntries {
		logData[i] = map[string]interface{}{
			"timestamp": entry.Timestamp,
			"level":     entry.Level,
			"message":   entry.Message,
		}
	}
	l.mutex.RUnlock()

	fm := NewFileManager()
	return fm.WriteJSON(l.logFilePath, logData)
}

// log writes a log entry with the specified level
func (l *Logger) log(level LogLevel, message string) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     string(level),
		Message:   message,
	}

	l.mutex.Lock()
	l.logEntries = append(l.logEntries, entry)
	l.mutex.Unlock()

	// Save to file
	l.saveLogs()

	// Publish to MQTT if client is available and log level is sufficient
	if l.mqttClient != nil && l.shouldPublishToMQTT(level) {
		l.publishToMQTT(entry)
	}
}

// shouldPublishToMQTT checks if the log level should be published to MQTT
func (l *Logger) shouldPublishToMQTT(level LogLevel) bool {
	minLevel := LogLevel(l.settings.MINIMAL_LOG_LEVEL)
	return level.GetIntLevel() >= minLevel.GetIntLevel()
}

// publishToMQTT publishes log entry to MQTT
func (l *Logger) publishToMQTT(entry LogEntry) {
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

	outputBaseTopic := l.schema.GetOutputBaseTopic()
	if topics, ok := outputBaseTopic[string(BaseOutputTopicTypeLogPepeunit)]; ok && len(topics) > 0 {
		// Format like Python client: use "create_datetime" and "text" fields
		logData := map[string]interface{}{
			"level":           entry.Level,
			"text":            entry.Message,
			"create_datetime": entry.Timestamp,
		}

		logJSON, err := json.Marshal(logData)
		if err != nil {
			return
		}

		l.mqttClient.Publish(topics[0], string(logJSON))
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

// GetFullLog returns all log entries
func (l *Logger) GetFullLog() []LogEntry {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	// Return a copy to prevent external modification
	logs := make([]LogEntry, len(l.logEntries))
	copy(logs, l.logEntries)
	return logs
}

// ResetLog clears all log entries
func (l *Logger) ResetLog() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.logEntries = make([]LogEntry, 0)
	l.saveLogs()
}

// SetMQTTClient sets the MQTT client for log publishing
func (l *Logger) SetMQTTClient(mqttClient MQTTClient) {
	l.mqttClient = mqttClient
}

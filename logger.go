package pepeunit

import (
	"encoding/json"
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
	logFilePath string
	mqttClient  MQTTClient
	schema      *SchemaManager
	settings    *Settings
	logEntries  []LogEntry
	mutex       sync.RWMutex
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

	logData, err := fm.ReadJSON(l.logFilePath)
	if err != nil {
		return
	}

	if entries, ok := logData["entries"].([]interface{}); ok {
		l.mutex.Lock()
		defer l.mutex.Unlock()

		for _, entry := range entries {
			if entryMap, ok := entry.(map[string]interface{}); ok {
				logEntry := LogEntry{
					Timestamp: toString(entryMap["timestamp"]),
					Level:     toString(entryMap["level"]),
					Message:   toString(entryMap["message"]),
				}
				l.logEntries = append(l.logEntries, logEntry)
			}
		}
	}
}

// saveLogs saves log entries to file
func (l *Logger) saveLogs() error {
	if l.logFilePath == "" {
		return nil
	}

	l.mutex.RLock()
	logData := map[string]interface{}{
		"entries": l.logEntries,
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

	outputBaseTopic := l.schema.GetOutputBaseTopic()
	if topics, ok := outputBaseTopic[string(BaseOutputTopicTypeLogPepeunit)]; ok && len(topics) > 0 {
		logData, err := json.Marshal(entry)
		if err != nil {
			return
		}

		l.mqttClient.Publish(topics[0], string(logData))
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

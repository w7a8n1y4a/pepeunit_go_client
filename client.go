package pepeunit

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// PepeunitClient is the main client for PepeUnit integration
type PepeunitClient struct {
	envFilePath         string
	schemaFilePath      string
	logFilePath         string
	enableMQTT          bool
	enableREST          bool
	cycleSpeed          time.Duration
	restartMode         RestartMode
	settings            *Settings
	schema              *SchemaManager
	logger              *Logger
	mqttClient          MQTTClient
	restClient          RESTClient
	inputHandler        MQTTInputHandler
	outputHandler       func(*PepeunitClient)
	customUpdateHandler func(*PepeunitClient, string) error
	running             bool
	lastStateSend       time.Time
	mutex               sync.RWMutex
}

// PepeunitClientConfig holds configuration for creating a PepeunitClient
type PepeunitClientConfig struct {
	EnvFilePath    string
	SchemaFilePath string
	LogFilePath    string
	EnableMQTT     bool
	EnableREST     bool
	CycleSpeed     time.Duration
	RestartMode    RestartMode
	MQTTClient     MQTTClient
	RESTClient     RESTClient
}

// NewPepeunitClient creates a new PepeUnit client
func NewPepeunitClient(config PepeunitClientConfig) (*PepeunitClient, error) {
	// Validate required paths
	if config.EnvFilePath == "" {
		return nil, fmt.Errorf("env file path is required")
	}
	if config.SchemaFilePath == "" {
		return nil, fmt.Errorf("schema file path is required")
	}
	if config.LogFilePath == "" {
		return nil, fmt.Errorf("log file path is required")
	}

	// Set defaults
	if config.CycleSpeed == 0 {
		config.CycleSpeed = 100 * time.Millisecond
	}
	if config.RestartMode == "" {
		config.RestartMode = RestartModeRestartExec
	}

	// Initialize components
	settings := NewSettings(config.EnvFilePath)
	schema, err := NewSchemaManager(config.SchemaFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create schema manager: %v", err)
	}

	logger := NewLogger(config.LogFilePath, nil, schema, settings)

	client := &PepeunitClient{
		envFilePath:    config.EnvFilePath,
		schemaFilePath: config.SchemaFilePath,
		logFilePath:    config.LogFilePath,
		enableMQTT:     config.EnableMQTT,
		enableREST:     config.EnableREST,
		cycleSpeed:     config.CycleSpeed,
		restartMode:    config.RestartMode,
		settings:       settings,
		schema:         schema,
		logger:         logger,
		running:        false,
	}

	// Initialize MQTT client
	if config.EnableMQTT {
		if config.MQTTClient != nil {
			client.mqttClient = config.MQTTClient
		} else {
			client.mqttClient = NewPepeunitMQTTClient(settings, schema, logger)
		}
		logger.SetMQTTClient(client.mqttClient)
	}

	// Initialize REST client
	if config.EnableREST {
		if config.RESTClient != nil {
			client.restClient = config.RESTClient
		} else {
			client.restClient = NewPepeunitRESTClient(settings)
		}
	}

	return client, nil
}

// GetUnitUUID extracts the unit UUID from the JWT token
func (c *PepeunitClient) GetUnitUUID() (string, error) {
	tokenParts := splitToken(c.settings.PEPEUNIT_TOKEN)
	if len(tokenParts) != 3 {
		return "", fmt.Errorf("invalid JWT token format")
	}

	payload := tokenParts[1]
	// Add padding if needed
	for len(payload)%4 != 0 {
		payload += "="
	}

	decodedPayload, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("failed to decode JWT payload: %v", err)
	}

	var payloadData map[string]interface{}
	err = json.Unmarshal(decodedPayload, &payloadData)
	if err != nil {
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

// splitToken splits a JWT token into its parts
func splitToken(token string) []string {
	return strings.Split(token, ".")
}

// SetCycleSpeed sets the main cycle execution speed
func (c *PepeunitClient) SetCycleSpeed(speed time.Duration) error {
	if speed <= 0 {
		return fmt.Errorf("cycle speed must be greater than 0")
	}
	c.mutex.Lock()
	c.cycleSpeed = speed
	c.mutex.Unlock()
	return nil
}

// UpdateDeviceProgram updates the device program from a tar.gz archive
func (c *PepeunitClient) UpdateDeviceProgram(ctx context.Context, archivePath string) error {
	tempExtractDir, err := os.MkdirTemp("", "pepeunit_update_*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempExtractDir)

	fm := NewFileManager()
	err = fm.ExtractTarGz(archivePath, tempExtractDir)
	if err != nil {
		return fmt.Errorf("failed to extract archive: %v", err)
	}
	c.logger.Info(fmt.Sprintf("Extracted archive to %s", tempExtractDir))

	var foundEnv string
	var foundSchema string
	walkErr := filepath.Walk(tempExtractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		name := filepath.Base(path)
		switch name {
		case filepath.Base(c.envFilePath):
			foundEnv = path
		case filepath.Base(c.schemaFilePath):
			foundSchema = path
		case "env.json":
			if foundEnv == "" {
				foundEnv = path
			}
		case "schema.json":
			if foundSchema == "" {
				foundSchema = path
			}
		}
		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("failed to scan extracted files: %v", walkErr)
	}

	if foundEnv != "" {
		if err := fm.CopyFile(foundEnv, c.envFilePath); err != nil {
			return fmt.Errorf("failed to update env: %v", err)
		}
	}
	if foundSchema != "" {
		if err := fm.CopyFile(foundSchema, c.schemaFilePath); err != nil {
			return fmt.Errorf("failed to update schema: %v", err)
		}
	}

	return c.updateEnvSchemaOnly(ctx)
}

// updateEnvSchemaOnly updates only environment and schema files
func (c *PepeunitClient) updateEnvSchemaOnly(ctx context.Context) error {
	err := c.settings.LoadFromFile()
	if err != nil {
		return fmt.Errorf("failed to reload settings: %v", err)
	}

	err = c.schema.UpdateFromFile()
	if err != nil {
		return fmt.Errorf("failed to reload schema: %v", err)
	}

	if c.enableMQTT && c.mqttClient != nil {
		err = c.SubscribeAllSchemaTopics(ctx)
		if err != nil {
			return fmt.Errorf("failed to resubscribe to topics: %v", err)
		}
	}

	c.logger.Info("Environment and schema updated successfully")
	return nil
}

// GetSystemState returns current system status information
func (c *PepeunitClient) GetSystemState() map[string]interface{} {
	state := map[string]interface{}{
		"millis":         time.Now().UnixMilli(),
		"mem_free":       0,
		"mem_alloc":      0,
		"freq":           0,
		"commit_version": c.settings.COMMIT_VERSION,
	}

	// Get memory information
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		state["mem_free"] = memInfo.Available
		state["mem_alloc"] = memInfo.Total - memInfo.Available
	}

	// Get CPU frequency
	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		state["freq"] = cpuInfo[0].Mhz
	}

	return state
}

// SetMQTTInputHandler sets the MQTT input message handler
func (c *PepeunitClient) SetMQTTInputHandler(handler MQTTInputHandler) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.inputHandler = handler
	if c.mqttClient != nil {
		// Create a combined handler that includes base functionality
		combinedHandler := func(msg MQTTMessage) {
			c.baseMQTTInputFunc(msg)
			if c.inputHandler != nil {
				c.inputHandler(msg)
			}
		}
		c.mqttClient.SetInputHandler(combinedHandler)
	}
}

// baseMQTTInputFunc handles base MQTT input functionality
func (c *PepeunitClient) baseMQTTInputFunc(msg MQTTMessage) {
	topic := msg.Topic
	payload := string(msg.Payload)

	inputBaseTopic := c.schema.GetInputBaseTopic()

	for topicKey, topics := range inputBaseTopic {
		for _, topicURL := range topics {
			if topic == topicURL {
				ctx := context.Background()
				switch topicKey {
				case string(BaseInputTopicTypeUpdatePepeunit):
					c.handleUpdate(ctx, payload)
				case string(BaseInputTopicTypeEnvUpdatePepeunit):
					c.handleEnvUpdate(ctx)
				case string(BaseInputTopicTypeSchemaUpdatePepeunit):
					c.handleSchemaUpdate(ctx)
				case string(BaseInputTopicTypeLogSyncPepeunit):
					c.handleLogSync(ctx)
				}
				return
			}
		}
	}
}

// handleUpdate handles update requests
func (c *PepeunitClient) handleUpdate(ctx context.Context, payload string) {
	c.logger.Info("Update request received via MQTT")
	if !c.enableREST || c.restClient == nil {
		c.logger.Warning("REST client not available for update")
		return
	}

	if c.customUpdateHandler != nil {
		if err := c.customUpdateHandler(c, payload); err != nil {
			c.logger.Error(fmt.Sprintf("Failed to perform custom update: %v", err))
		}
		return
	}

	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &meta); err != nil {
		c.logger.Error(fmt.Sprintf("Failed to parse payload: %v", err))
		return
	}

	linkVal, ok := meta["COMPILED_FIRMWARE_LINK"]
	if ok {
		if err := c.PerformUpdate(ctx); err != nil {
			c.logger.Error(fmt.Sprintf("Failed to update env/schema: %v", err))
			return
		}
		linkStr, _ := linkVal.(string)
		if linkStr != "" {
			if err := c.UpdateBinaryFromURL(ctx, linkStr); err != nil {
				c.logger.Error(fmt.Sprintf("Failed to update binary: %v", err))
				return
			}

			switch c.restartMode {
			case RestartModeRestartPopen:
				c.StopMainCycle()
				executable, err := os.Executable()
				if err != nil {
					c.logger.Error(fmt.Sprintf("Failed to get executable path: %v", err))
					return
				}
				cmd := exec.Command(executable, os.Args[1:]...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Stdin = os.Stdin
				if err := cmd.Start(); err != nil {
					c.logger.Error(fmt.Sprintf("Failed to start new process: %v", err))
					return
				}
				os.Exit(0)
			case RestartModeRestartExec:
				c.StopMainCycle()
				executable, err := os.Executable()
				if err != nil {
					c.logger.Error(fmt.Sprintf("Failed to get executable path: %v", err))
					return
				}
				if err := syscall.Exec(executable, os.Args, os.Environ()); err != nil {
					c.logger.Error(fmt.Sprintf("Failed to exec new process: %v", err))
					return
				}
			case RestartModeEnvSchemaOnly:
				return
			case RestartModeNoRestart:
				return
			}
		}
	}
}

func (c *PepeunitClient) UpdateBinaryFromURL(ctx context.Context, firmwareURL string) error {
	if !c.enableREST || c.restClient == nil {
		return fmt.Errorf("REST client is not enabled or available")
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	dir := filepath.Dir(executable)
	tempPath := filepath.Join(dir, filepath.Base(executable)+".new")

	if err := c.restClient.DownloadFileFromURL(ctx, firmwareURL, tempPath); err != nil {
		return err
	}

	if err := os.Chmod(tempPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %v", err)
	}

	if err := os.Rename(tempPath, executable); err != nil {
		_ = os.Remove(executable)
		if err2 := os.Rename(tempPath, executable); err2 != nil {
			return fmt.Errorf("failed to replace binary: %v", err2)
		}
	}

	c.logger.Info("Binary file downloaded successfully")
	return nil
}

// handleEnvUpdate handles environment update requests
func (c *PepeunitClient) handleEnvUpdate(ctx context.Context) {
	c.logger.Info("Env update request received via MQTT")
	if c.enableREST && c.restClient != nil {
		unitUUID, err := c.GetUnitUUID()
		if err != nil {
			c.logger.Error(fmt.Sprintf("Failed to get unit UUID: %v", err))
			return
		}

		err = c.restClient.DownloadEnv(ctx, unitUUID, c.envFilePath)
		if err != nil {
			c.logger.Error(fmt.Sprintf("Failed to update env: %v", err))
		} else {
			c.settings.LoadFromFile()
		}
	} else {
		c.logger.Warning("REST client not available for env update")
	}
}

// handleSchemaUpdate handles schema update requests
func (c *PepeunitClient) handleSchemaUpdate(ctx context.Context) {
	c.logger.Info("Schema update request received via MQTT")
	if c.enableREST && c.restClient != nil {
		unitUUID, err := c.GetUnitUUID()
		if err != nil {
			c.logger.Error(fmt.Sprintf("Failed to get unit UUID: %v", err))
			return
		}

		err = c.restClient.DownloadSchema(ctx, unitUUID, c.schemaFilePath)
		if err != nil {
			c.logger.Error(fmt.Sprintf("Failed to update schema: %v", err))
		} else {
			c.schema.UpdateFromFile()
			if c.enableMQTT && c.mqttClient != nil {
				c.SubscribeAllSchemaTopics(ctx)
			}
		}
	} else {
		c.logger.Warning("REST client not available for schema update")
	}
}

// handleLogSync handles log synchronization requests
func (c *PepeunitClient) handleLogSync(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Error(fmt.Sprintf("Error during log sync: %v", r))
		}
	}()

	outputBaseTopic := c.schema.GetOutputBaseTopic()
	if topics, ok := outputBaseTopic[string(BaseOutputTopicTypeLogPepeunit)]; ok && len(topics) > 0 {
		logData := c.logger.GetFullLog()
		logJSON, err := json.Marshal(logData)
		if err != nil {
			c.logger.Error(fmt.Sprintf("Error marshaling log data: %v", err))
			return
		}

		if c.mqttClient != nil {
			c.mqttClient.Publish(topics[0], string(logJSON))
		}
		c.logger.Info("Log sync completed")
	}
}

// DownloadUpdate downloads firmware update archive
func (c *PepeunitClient) DownloadUpdate(ctx context.Context, archivePath string) error {
	if !c.enableREST || c.restClient == nil {
		return fmt.Errorf("REST client is not enabled or available")
	}

	unitUUID, err := c.GetUnitUUID()
	if err != nil {
		return fmt.Errorf("failed to get unit UUID: %v", err)
	}

	err = c.restClient.DownloadUpdate(ctx, unitUUID, archivePath)
	if err != nil {
		return err
	}

	c.logger.Info(fmt.Sprintf("Update archive downloaded to %s", archivePath))
	return nil
}

// DownloadEnv downloads environment configuration
func (c *PepeunitClient) DownloadEnv(ctx context.Context, filePath string) error {
	if !c.enableREST || c.restClient == nil {
		return fmt.Errorf("REST client is not enabled or available")
	}

	unitUUID, err := c.GetUnitUUID()
	if err != nil {
		return fmt.Errorf("failed to get unit UUID: %v", err)
	}

	err = c.restClient.DownloadEnv(ctx, unitUUID, filePath)
	if err != nil {
		return err
	}

	c.settings.LoadFromFile()
	c.logger.Info(fmt.Sprintf("Environment file downloaded and updated from %s", filePath))
	return nil
}

// DownloadSchema downloads topic schema configuration
func (c *PepeunitClient) DownloadSchema(ctx context.Context, filePath string) error {
	if !c.enableREST || c.restClient == nil {
		return fmt.Errorf("REST client is not enabled or available")
	}

	unitUUID, err := c.GetUnitUUID()
	if err != nil {
		return fmt.Errorf("failed to get unit UUID: %v", err)
	}

	err = c.restClient.DownloadSchema(ctx, unitUUID, filePath)
	if err != nil {
		return err
	}

	c.schema.UpdateFromFile()
	c.logger.Info(fmt.Sprintf("Schema file downloaded and updated from %s", filePath))
	return nil
}

// SetStateStorage stores state data in PepeUnit storage
func (c *PepeunitClient) SetStateStorage(ctx context.Context, state map[string]interface{}) error {
	if !c.enableREST || c.restClient == nil {
		return fmt.Errorf("REST client is not enabled or available")
	}

	unitUUID, err := c.GetUnitUUID()
	if err != nil {
		return fmt.Errorf("failed to get unit UUID: %v", err)
	}

	err = c.restClient.SetStateStorage(ctx, unitUUID, state)
	if err != nil {
		return err
	}

	c.logger.Info("State uploaded to PepeUnit Unit Storage")
	return nil
}

// GetStateStorage retrieves state data from PepeUnit storage
func (c *PepeunitClient) GetStateStorage(ctx context.Context) (map[string]interface{}, error) {
	if !c.enableREST || c.restClient == nil {
		return nil, fmt.Errorf("REST client is not enabled or available")
	}

	unitUUID, err := c.GetUnitUUID()
	if err != nil {
		return nil, fmt.Errorf("failed to get unit UUID: %v", err)
	}

	state, err := c.restClient.GetStateStorage(ctx, unitUUID)
	if err != nil {
		return nil, err
	}

	c.logger.Info("State retrieved from PepeUnit Unit Storage")
	return state, nil
}

// PerformUpdate performs a complete update cycle
func (c *PepeunitClient) PerformUpdate(ctx context.Context) error {
	if !c.enableMQTT || !c.enableREST {
		return fmt.Errorf("both MQTT and REST clients must be enabled for perform_update")
	}

	tempDir := os.TempDir()
	unitUUID, err := c.GetUnitUUID()
	if err != nil {
		return fmt.Errorf("failed to get unit UUID: %v", err)
	}

	archivePath := filepath.Join(tempDir, fmt.Sprintf("update_%s.tar.gz", unitUUID))

	err = c.DownloadUpdate(ctx, archivePath)
	if err != nil {
		return fmt.Errorf("failed to download update: %v", err)
	}

	err = c.UpdateDeviceProgram(ctx, archivePath)
	if err != nil {
		return fmt.Errorf("failed to update device program: %v", err)
	}

	err = os.Remove(archivePath)
	if err != nil {
		c.logger.Warning(fmt.Sprintf("Failed to remove temporary archive: %v", err))
	}

	c.logger.Info("Schema and env updated successfully")
	return nil
}

// SubscribeAllSchemaTopics subscribes to all schema-defined topics
func (c *PepeunitClient) SubscribeAllSchemaTopics(ctx context.Context) error {
	if !c.enableMQTT || c.mqttClient == nil {
		return fmt.Errorf("MQTT client is not enabled or available")
	}

	topics := make([]string, 0)

	// Add input base topics
	inputBaseTopic := c.schema.GetInputBaseTopic()
	for _, topicList := range inputBaseTopic {
		topics = append(topics, topicList...)
	}

	// Add input topics
	inputTopic := c.schema.GetInputTopic()
	for _, topicList := range inputTopic {
		topics = append(topics, topicList...)
	}

	return c.mqttClient.SubscribeTopics(topics)
}

// PublishToTopics publishes a message to all topics with the given key
func (c *PepeunitClient) PublishToTopics(ctx context.Context, topicKey, message string) error {
	if !c.enableMQTT || c.mqttClient == nil {
		return fmt.Errorf("MQTT client is not enabled or available")
	}

	topics := make([]string, 0)

	// Check output topics first
	outputTopic := c.schema.GetOutputTopic()
	if topicList, ok := outputTopic[topicKey]; ok {
		topics = append(topics, topicList...)
	}

	// Check output base topics
	outputBaseTopic := c.schema.GetOutputBaseTopic()
	if topicList, ok := outputBaseTopic[topicKey]; ok {
		topics = append(topics, topicList...)
	}

	// Publish to all matching topics
	for _, topic := range topics {
		err := c.mqttClient.Publish(topic, message)
		if err != nil {
			c.logger.Error(fmt.Sprintf("Failed to publish to topic %s: %v", topic, err))
		}
	}

	return nil
}

// baseMQTTOutputHandler handles base MQTT output functionality
func (c *PepeunitClient) baseMQTTOutputHandler(ctx context.Context) {
	currentTime := time.Now()

	outputBaseTopic := c.schema.GetOutputBaseTopic()
	if topics, ok := outputBaseTopic[string(BaseOutputTopicTypeStatePepeunit)]; ok && len(topics) > 0 {
		c.mutex.RLock()
		shouldSend := currentTime.Sub(c.lastStateSend) >= time.Duration(c.settings.STATE_SEND_INTERVAL)*time.Second
		c.mutex.RUnlock()

		if shouldSend {
			stateData := c.GetSystemState()
			stateJSON, err := json.Marshal(stateData)
			if err != nil {
				c.logger.Error(fmt.Sprintf("Failed to marshal state data: %v", err))
				return
			}

			if c.mqttClient != nil {
				err = c.mqttClient.Publish(topics[0], string(stateJSON))
				if err != nil {
					c.logger.Error(fmt.Sprintf("Failed to publish state: %v", err))
				} else {
					c.mutex.Lock()
					c.lastStateSend = currentTime
					c.mutex.Unlock()
				}
			}
		}
	}
}

// RunMainCycle starts the main application loop
func (c *PepeunitClient) RunMainCycle(ctx context.Context, outputHandler func(*PepeunitClient)) {
	c.mutex.Lock()
	c.running = true
	c.outputHandler = outputHandler
	c.mutex.Unlock()

	defer func() {
		c.mutex.Lock()
		c.running = false
		c.mutex.Unlock()
	}()

	ticker := time.NewTicker(c.cycleSpeed)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Main cycle stopped by context")
			return
		case <-ticker.C:
			if !c.isRunning() {
				return
			}

			// Handle base MQTT output
			c.baseMQTTOutputHandler(ctx)

			// Handle custom output
			if c.outputHandler != nil {
				c.outputHandler(c)
			}
		}
	}
}

// SetOutputHandler sets the custom output message handler
func (c *PepeunitClient) SetOutputHandler(outputHandler func(*PepeunitClient)) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.outputHandler = outputHandler
}

// SetCustomUpdateHandler sets a custom handler for program updates triggered via MQTT
func (c *PepeunitClient) SetCustomUpdateHandler(handler func(*PepeunitClient, string) error) {
	c.mutex.Lock()
	c.customUpdateHandler = handler
	c.mutex.Unlock()
}

// StopMainCycle stops the main application loop
func (c *PepeunitClient) StopMainCycle() {
	c.logger.Info("Stop main cycle")
	c.mutex.Lock()
	c.running = false
	c.mutex.Unlock()
}

// isRunning checks if the main cycle is running
func (c *PepeunitClient) isRunning() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.running
}

// GetSettings returns the settings manager
func (c *PepeunitClient) GetSettings() *Settings {
	return c.settings
}

// GetSchema returns the schema manager
func (c *PepeunitClient) GetSchema() *SchemaManager {
	return c.schema
}

// GetLogger returns the logger
func (c *PepeunitClient) GetLogger() *Logger {
	return c.logger
}

// GetMQTTClient returns the MQTT client
func (c *PepeunitClient) GetMQTTClient() MQTTClient {
	return c.mqttClient
}

// GetRESTClient returns the REST client
func (c *PepeunitClient) GetRESTClient() RESTClient {
	return c.restClient
}

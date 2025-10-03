package pepeunit

import "context"

// MQTTMessage represents an MQTT message
type MQTTMessage struct {
	Topic   string
	Payload []byte
}

// MQTTInputHandler is a function type for handling incoming MQTT messages
type MQTTInputHandler func(msg MQTTMessage)

// MQTTClient interface for MQTT operations
type MQTTClient interface {
	// Connect connects to the MQTT broker
	Connect(ctx context.Context) error

	// Disconnect disconnects from the MQTT broker
	Disconnect(ctx context.Context) error

	// SubscribeTopics subscribes to a list of MQTT topics
	SubscribeTopics(topics []string) error

	// Publish publishes a message to a specific topic
	Publish(topic, message string) error

	// SetInputHandler sets the handler for incoming messages
	SetInputHandler(handler MQTTInputHandler)
}

// RESTClient interface for REST API operations
type RESTClient interface {
	// DownloadUpdate downloads firmware update archive
	DownloadUpdate(ctx context.Context, unitUUID, filePath string) error

	// DownloadEnv downloads environment configuration
	DownloadEnv(ctx context.Context, unitUUID, filePath string) error

	// DownloadSchema downloads topic schema configuration
	DownloadSchema(ctx context.Context, unitUUID, filePath string) error

	// DownloadFileFromURL downloads a file from an external URL
	DownloadFileFromURL(ctx context.Context, url, filePath string) error

	// SetStateStorage stores state data in PepeUnit storage
	SetStateStorage(ctx context.Context, unitUUID string, state map[string]interface{}) error

	// GetStateStorage retrieves state data from PepeUnit storage
	GetStateStorage(ctx context.Context, unitUUID string) (map[string]interface{}, error)
}

// AbstractMQTTClient is an abstract base for MQTT clients
type AbstractMQTTClient struct {
	Settings      *Settings
	SchemaManager *SchemaManager
	Logger        *Logger
}

// NewAbstractMQTTClient creates a new abstract MQTT client
func NewAbstractMQTTClient(settings *Settings, schemaManager *SchemaManager, logger *Logger) *AbstractMQTTClient {
	return &AbstractMQTTClient{
		Settings:      settings,
		SchemaManager: schemaManager,
		Logger:        logger,
	}
}

// AbstractRESTClient is an abstract base for REST clients
type AbstractRESTClient struct {
	Settings *Settings
}

// NewAbstractRESTClient creates a new abstract REST client
func NewAbstractRESTClient(settings *Settings) *AbstractRESTClient {
	return &AbstractRESTClient{
		Settings: settings,
	}
}

// GetAuthHeaders returns authentication headers for API requests
func (c *AbstractRESTClient) GetAuthHeaders() map[string]string {
	return map[string]string{
		"accept":       "application/json",
		"x-auth-token": c.Settings.PEPEUNIT_TOKEN,
	}
}

// GetBaseURL returns the base URL for PepeUnit API
func (c *AbstractRESTClient) GetBaseURL() string {
	return c.Settings.HTTP_TYPE + "://" + c.Settings.PEPEUNIT_URL + c.Settings.PEPEUNIT_APP_PREFIX + c.Settings.PEPEUNIT_API_ACTUAL_PREFIX
}

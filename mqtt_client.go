package pepeunit

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// PepeunitMQTTClient implements MQTTClient interface using paho.mqtt.golang
type PepeunitMQTTClient struct {
	*AbstractMQTTClient
	client    mqtt.Client
	handler   MQTTInputHandler
	connected bool
}

// NewPepeunitMQTTClient creates a new MQTT client
func NewPepeunitMQTTClient(settings *Settings, schemaManager *SchemaManager, logger *Logger) *PepeunitMQTTClient {
	return &PepeunitMQTTClient{
		AbstractMQTTClient: NewAbstractMQTTClient(settings, schemaManager, logger),
		connected:          false,
	}
}

// Connect connects to the MQTT broker
func (c *PepeunitMQTTClient) Connect(ctx context.Context) error {
	// Generate unique client ID like Python client
	clientID := c.generateUniqueClientID()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", c.Settings.MQTT_URL, c.Settings.MQTT_PORT))
	opts.SetClientID(clientID)
	opts.SetUsername(c.Settings.PEPEUNIT_TOKEN) // Use PEPEUNIT_TOKEN as username like Python client
	opts.SetPassword("")                        // Empty password like Python client
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetKeepAlive(time.Duration(c.Settings.PING_INTERVAL) * time.Second)

	// Set up TLS if needed
	if c.Settings.HTTP_TYPE == "https" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
		}
		opts.SetTLSConfig(tlsConfig)
		// Update broker URL to use SSL
		opts.AddBroker(fmt.Sprintf("ssl://%s:%d", c.Settings.MQTT_URL, c.Settings.MQTT_PORT))
	}

	// Set connection lost handler
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		c.connected = false
		c.Logger.Error(fmt.Sprintf("MQTT connection lost: %v", err))
	})

	// Set on connect handler
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		c.connected = true
		c.Logger.Info("MQTT connected successfully")
	})

	// Set reconnect handler
	opts.SetReconnectingHandler(func(client mqtt.Client, options *mqtt.ClientOptions) {
		c.Logger.Info("MQTT reconnecting...")
	})

	c.client = mqtt.NewClient(opts)

	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %v", token.Error())
	}

	c.connected = true
	c.Logger.Info("MQTT client connected successfully")
	return nil
}

// Disconnect disconnects from the MQTT broker
func (c *PepeunitMQTTClient) Disconnect(ctx context.Context) error {
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250) // Wait 250ms for disconnect
		c.connected = false
		c.Logger.Info("MQTT client disconnected")
	}
	return nil
}

// SubscribeTopics subscribes to a list of MQTT topics
func (c *PepeunitMQTTClient) SubscribeTopics(topics []string) error {
	if c.client == nil || !c.client.IsConnected() {
		return fmt.Errorf("MQTT client is not connected")
	}

	for _, topic := range topics {
		if token := c.client.Subscribe(topic, 1, c.messageHandler); token.Wait() && token.Error() != nil {
			c.Logger.Error(fmt.Sprintf("Failed to subscribe to topic %s: %v", topic, token.Error()))
			return token.Error()
		}
	}

	return nil
}

// Publish publishes a message to a specific topic
func (c *PepeunitMQTTClient) Publish(topic, message string) error {
	if c.client == nil || !c.client.IsConnected() {
		return fmt.Errorf("MQTT client is not connected")
	}

	token := c.client.Publish(topic, 1, false, message)
	if token.Wait() && token.Error() != nil {
		c.Logger.Error(fmt.Sprintf("Failed to publish to topic %s: %v", topic, token.Error()))
		return token.Error()
	}

	return nil
}

// SetInputHandler sets the handler for incoming messages
func (c *PepeunitMQTTClient) SetInputHandler(handler MQTTInputHandler) {
	c.handler = handler
}

// messageHandler handles incoming MQTT messages
func (c *PepeunitMQTTClient) messageHandler(client mqtt.Client, msg mqtt.Message) {
	if c.handler != nil {
		mqttMsg := MQTTMessage{
			Topic:   msg.Topic(),
			Payload: msg.Payload(),
		}
		c.handler(mqttMsg)
	}
}

// IsConnected returns whether the client is connected
func (c *PepeunitMQTTClient) IsConnected() bool {
	return c.connected && c.client != nil && c.client.IsConnected()
}

// GetClient returns the underlying MQTT client
func (c *PepeunitMQTTClient) GetClient() mqtt.Client {
	return c.client
}

// generateUniqueClientID generates a unique client ID like Python's uuid.uuid4()
func (c *PepeunitMQTTClient) generateUniqueClientID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)

	// Format like UUID4: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x",
		bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])

	return uuid
}

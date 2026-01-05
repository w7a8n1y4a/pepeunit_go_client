package pepeunit

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// PepeunitMQTTClient implements MQTTClient interface using paho.mqtt.golang
type PepeunitMQTTClient struct {
	*AbstractMQTTClient
	client          mqtt.Client
	handler         MQTTInputHandler
	connected       bool
	subscriptionsMu sync.RWMutex
	subscriptions   map[string]byte
	connectMu       sync.Mutex
}

// NewPepeunitMQTTClient creates a new MQTT client
func NewPepeunitMQTTClient(settings *Settings, schemaManager *SchemaManager, logger *Logger) *PepeunitMQTTClient {
	return &PepeunitMQTTClient{
		AbstractMQTTClient: NewAbstractMQTTClient(settings, schemaManager, logger),
		connected:          false,
		subscriptions:      make(map[string]byte),
	}
}

// Connect connects to the MQTT broker
func (c *PepeunitMQTTClient) Connect(ctx context.Context) error {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()
	return c.connectLocked()
}

func (c *PepeunitMQTTClient) connectLocked() error {
	// Generate unique client ID like Python client
	clientID := c.generateUniqueClientID()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", c.Settings.PU_MQTT_HOST, c.Settings.PU_MQTT_PORT))
	opts.SetClientID(clientID)
	opts.SetUsername(c.Settings.PU_AUTH_TOKEN) // Use PU_AUTH_TOKEN as username like Python client
	opts.SetPassword("")                       // Empty password like Python client
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetConnectRetryInterval(1 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetKeepAlive(time.Duration(c.Settings.PU_MQTT_PING_INTERVAL) * time.Second)
	opts.SetWriteTimeout(10 * time.Second)

	// Set connection lost handler
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		c.connected = false
		c.Logger.Error(fmt.Sprintf("MQTT connection lost: %v", err))
	})

	// Set on connect handler
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		c.connected = true
		c.Logger.Info("Connected to MQTT Broker")
		go c.resubscribeAll()
	})

	// Set reconnect handler
	opts.SetReconnectingHandler(func(client mqtt.Client, options *mqtt.ClientOptions) {
		c.Logger.Info("MQTT reconnecting...")
	})

	c.client = mqtt.NewClient(opts)

	token := c.client.Connect()
	if !token.WaitTimeout(15 * time.Second) {
		return fmt.Errorf("failed to connect to MQTT broker: timeout")
	}
	if token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %v", token.Error())
	}

	c.connected = true
	c.Logger.Info("MQTT client connected successfully")
	return nil
}

// Disconnect disconnects from the MQTT broker
func (c *PepeunitMQTTClient) Disconnect(ctx context.Context) error {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250) // Wait 250ms for disconnect
		c.connected = false
		c.Logger.Info("Disconnected from MQTT Broker", true)
	}
	return nil
}

// SubscribeTopics subscribes to a list of MQTT topics
func (c *PepeunitMQTTClient) SubscribeTopics(topics []string) error {
	if len(topics) == 0 {
		return nil
	}

	c.subscriptionsMu.Lock()
	for _, topic := range topics {
		c.subscriptions[topic] = 1
	}
	c.subscriptionsMu.Unlock()

	if c.client == nil || !c.client.IsConnected() {
		if err := c.forceReconnect(); err != nil {
			return err
		}
	}

	for _, topic := range topics {
		token := c.client.Subscribe(topic, 1, c.messageHandler)
		if !token.WaitTimeout(5 * time.Second) {
			_ = c.forceReconnect()
			token = c.client.Subscribe(topic, 1, c.messageHandler)
			if !token.WaitTimeout(5 * time.Second) {
				return fmt.Errorf("failed to subscribe to topic %s: timeout", topic)
			}
		}
		if token.Error() != nil {
			c.Logger.Error(fmt.Sprintf("Failed to subscribe to topic %s: %v", topic, token.Error()))
			_ = c.forceReconnect()
			token2 := c.client.Subscribe(topic, 1, c.messageHandler)
			if !token2.WaitTimeout(5 * time.Second) {
				return fmt.Errorf("failed to subscribe to topic %s: timeout", topic)
			}
			if token2.Error() != nil {
				return token2.Error()
			}
		}
	}

	c.Logger.Info(fmt.Sprintf("Success subscribed to %d topics", len(topics)))
	return nil
}

// UnsubscribeTopics unsubscribes from a list of MQTT topics
func (c *PepeunitMQTTClient) UnsubscribeTopics(topics []string) error {
	if len(topics) == 0 {
		return nil
	}

	c.subscriptionsMu.Lock()
	for _, topic := range topics {
		delete(c.subscriptions, topic)
	}
	c.subscriptionsMu.Unlock()

	if c.client == nil || !c.client.IsConnected() {
		if err := c.forceReconnect(); err != nil {
			return err
		}
	}
	token := c.client.Unsubscribe(topics...)
	if !token.WaitTimeout(5 * time.Second) {
		_ = c.forceReconnect()
		token = c.client.Unsubscribe(topics...)
		if !token.WaitTimeout(5 * time.Second) {
			return fmt.Errorf("failed to unsubscribe from topics: timeout")
		}
	}
	if token.Error() != nil {
		c.Logger.Error(fmt.Sprintf("Failed to unsubscribe from topics: %v", token.Error()))
		_ = c.forceReconnect()
		token2 := c.client.Unsubscribe(topics...)
		if !token2.WaitTimeout(5 * time.Second) {
			return fmt.Errorf("failed to unsubscribe from topics: timeout")
		}
		if token2.Error() != nil {
			return token2.Error()
		}
	}
	return nil
}

// Publish publishes a message to a specific topic
func (c *PepeunitMQTTClient) Publish(topic, message string) error {
	if c.client == nil || !c.client.IsConnected() {
		if err := c.forceReconnect(); err != nil {
			return err
		}
	}

	token := c.client.Publish(topic, 1, false, message)
	if !token.WaitTimeout(5 * time.Second) {
		_ = c.forceReconnect()
		token = c.client.Publish(topic, 1, false, message)
		if !token.WaitTimeout(5 * time.Second) {
			return fmt.Errorf("failed to publish to topic %s: timeout", topic)
		}
	}
	if token.Error() != nil {
		c.Logger.Error(fmt.Sprintf("Failed to publish to topic %s: %v", topic, token.Error()))
		_ = c.forceReconnect()
		token2 := c.client.Publish(topic, 1, false, message)
		if !token2.WaitTimeout(5 * time.Second) {
			return fmt.Errorf("failed to publish to topic %s: timeout", topic)
		}
		if token2.Error() != nil {
			return token2.Error()
		}
	}

	return nil
}

// SetInputHandler sets the handler for incoming messages
func (c *PepeunitMQTTClient) SetInputHandler(handler MQTTInputHandler) {
	c.handler = handler
}

// messageHandler handles incoming MQTT messages
func (c *PepeunitMQTTClient) messageHandler(client mqtt.Client, msg mqtt.Message) {
	defer func() {
		if r := recover(); r != nil {
			c.Logger.Error(fmt.Sprintf("Error processing MQTT message: %v", r))
		}
	}()
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

func (c *PepeunitMQTTClient) forceReconnect() error {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	if c.client != nil {
		c.client.Disconnect(250)
	}
	c.connected = false
	return c.connectLocked()
}

func (c *PepeunitMQTTClient) resubscribeAll() {
	c.subscriptionsMu.RLock()
	if len(c.subscriptions) == 0 {
		c.subscriptionsMu.RUnlock()
		return
	}
	topics := make([]string, 0, len(c.subscriptions))
	for topic := range c.subscriptions {
		topics = append(topics, topic)
	}
	c.subscriptionsMu.RUnlock()

	if c.client == nil || !c.client.IsConnected() {
		return
	}

	for _, topic := range topics {
		token := c.client.Subscribe(topic, 1, c.messageHandler)
		if !token.WaitTimeout(5 * time.Second) {
			c.Logger.Error(fmt.Sprintf("Timeout resubscribing to topic %s", topic))
			continue
		}
		if token.Error() != nil {
			c.Logger.Error(fmt.Sprintf("Failed to resubscribe to topic %s: %v", topic, token.Error()))
		}
	}
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

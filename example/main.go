package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

// Global variable to track last message send time
var lastOutputSendTime time.Time

func handleInputMessages(client *pepeunit.PepeunitClient, msg pepeunit.MQTTMessage) {
	topicParts := strings.Split(msg.Topic, "/")

	// topic with format domain.com/+/pepeunit
	if len(topicParts) == 3 {
		// find topic name in schema, by topic with struct domain.com/+/pepeunit or domain.com/+
		topicName, err := client.GetSchema().FindTopicByUnitNode(
			msg.Topic, pepeunit.SearchTopicTypeFullName, pepeunit.SearchScopeInput,
		)

		if err != nil {
			client.GetLogger().Error(fmt.Sprintf("Error finding topic: %v", err))
			return
		}

		if topicName == "input/pepeunit" {
			value := string(msg.Payload)
			intValue, err := strconv.Atoi(value)
			if err != nil {
				client.GetLogger().Error(fmt.Sprintf("Value is not a number: %s", value))
				return
			}

			// example logic
			if intValue == 0 {
				client.GetLogger().Info(fmt.Sprintf("Get message from input/pepeunit topics %d", intValue))
			} else {
				// send value to all topic by name
				ctx := context.Background()
				err = client.PublishToTopics(ctx, "output/pepeunit", strconv.Itoa(intValue))
				if err != nil {
					client.GetLogger().Error(fmt.Sprintf("Failed to publish message: %v", err))
				}
			}
		}
	}
}

func handleOutputMessages(client *pepeunit.PepeunitClient) {
	currentTime := time.Now()

	// Send data every DELAY_PUB_MSG from extras (fallback to STATE_SEND_INTERVAL)
	delay, ok := client.GetSettings().GetInt("DELAY_PUB_MSG")
	if !ok || delay <= 0 {
		delay = client.GetSettings().STATE_SEND_INTERVAL
	}
	if currentTime.Sub(lastOutputSendTime) >= time.Duration(delay)*time.Second {
		// message example
		message := "12.45"

		client.GetLogger().Info(fmt.Sprintf("Send message to output/pepeunit topics: %s", message))

		// Try to publish to sensor output topics
		ctx := context.Background()
		err := client.PublishToTopics(ctx, "output/pepeunit", message)
		if err != nil {
			client.GetLogger().Error(fmt.Sprintf("Failed to publish message: %v", err))
		}

		// Update the last message send time
		lastOutputSendTime = currentTime
	}
}

func main() {
	// Initialize the PepeUnit client
	client, err := pepeunit.NewPepeunitClient(pepeunit.PepeunitClientConfig{
		EnvFilePath:    "env.json",
		SchemaFilePath: "schema.json",
		LogFilePath:    "log.json",
		EnableMQTT:     true,
		EnableREST:     true,
		CycleSpeed:     1 * time.Second, // 1 second cycle
		RestartMode:    pepeunit.RestartModeRestartExec,
	})

	if err != nil {
		log.Fatalf("Failed to create PepeUnit client: %v", err)
	}

	// Log startup
	client.GetLogger().Debug("PepeUnit client created")

	unitUUID, err := client.GetUnitUUID()
	if err != nil {
		client.GetLogger().Error(fmt.Sprintf("Failed to get unit UUID: %v", err))
	} else {
		client.GetLogger().Debug(fmt.Sprintf("Device UUID: %s", unitUUID))
	}

	// Set up message handlers
	client.SetMQTTInputHandler(func(msg pepeunit.MQTTMessage) {
		handleInputMessages(client, msg)
	})

	// Connect to mqtt broker
	ctx := context.Background()
	err = client.GetMQTTClient().Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
	}

	// Subscribe to all input topics from schema, be sure to after connecting with the broker
	err = client.SubscribeAllSchemaTopics(ctx)
	if err != nil {
		log.Fatalf("Failed to subscribe to topics: %v", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a cancellable context for the main cycle
	cycleCtx, cycleCancel := context.WithCancel(context.Background())

	// Run the main cycle with set output handler in a goroutine
	go func() {
		client.RunMainCycle(cycleCtx, handleOutputMessages)
	}()

	// Wait for shutdown signal
	<-sigChan
	client.GetLogger().Info("Shutting down...")

	// Stop the main cycle by canceling the context
	cycleCancel()

	// Disconnect from MQTT
	if client.GetMQTTClient() != nil {
		err = client.GetMQTTClient().Disconnect(ctx)
		if err != nil {
			client.GetLogger().Error(fmt.Sprintf("Failed to disconnect from MQTT: %v", err))
		}
	}

	client.GetLogger().Info("Shutdown complete")
}

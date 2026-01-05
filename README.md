# Client Framework for Pepeunit

<div align="center">
    <img align="center" src="https://pepeunit.com/pepeunit-og.jpg"  width="640" height="320">
</div>

A cross-platform Go library for integrating with the PepeUnit IoT platform. This client mirrors the Python implementation, providing MQTT and REST functionality for device communications, configuration updates, and state management. The only public entry point is `PepeunitClient`.

## Installation

```bash
# Add module to your project
go get github.com/w7a8n1y4a/pepeunit_go_client

# (optional) tidy deps
go mod tidy
```

## Usage Example

```go
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

// Basic PepeUnit Client Example

// To use this example, simply create a Pepeunit Unit based on the repository https://git.pepemoss.com/pepe/pepeunit/units/universal_test_unit on any instance.

// The resulting schema.json and env.json files should be added to the example directory.

// This example demonstrates basic usage of the PepeUnit client with both MQTT and REST functionality.
// It shows how to:
// - Initialize the client with configuration files
// - Set up message handlers
// - Subscribe to topics
// - Run the main application cycle
// - Storage api
// - Units Nodes api

// Global variable to track last message send time
var lastOutputSendTime time.Time
var inc int

func testSetGetStorage(client *pepeunit.PepeunitClient) {
	ctx := context.Background()
	if client.GetRESTClient() == nil {
		client.GetLogger().Warning("REST client not enabled, skip storage test")
		return
	}
	state := "This line is saved in Pepeunit Instance"
	if err := client.GetRESTClient().SetStateStorage(ctx, state); err != nil {
		client.GetLogger().Error(fmt.Sprintf("Test set state failed: %v", err))
		return
	}
	client.GetLogger().Info("Success set state")
	val, err := client.GetRESTClient().GetStateStorage(ctx)
	if err != nil {
		client.GetLogger().Error(fmt.Sprintf("Test get state failed: %v", err))
		return
	}
	client.GetLogger().Info(fmt.Sprintf("Success get state: %s", val))
}

func testGetUnits(client *pepeunit.PepeunitClient) {
	ctx := context.Background()
	if client.GetRESTClient() == nil {
		client.GetLogger().Warning("REST client not enabled, skip units query test")
		return
	}
	outputTopics := client.GetSchema().GetOutputTopic()
	topicURLs := outputTopics["output/pepeunit"]
	if len(topicURLs) == 0 {
		client.GetLogger().Warning("No output/pepeunit topics found in schema")
		return
	}
	topicURL := topicURLs[0]
	client.GetLogger().Info(fmt.Sprintf("Querying input unit nodes for topic: %s", topicURL))

	unitNodesResp, err := client.GetRESTClient().GetInputByOutput(ctx, topicURL, 100, 0)
	if err != nil {
		client.GetLogger().Warning(fmt.Sprintf("REST get_input_by_output failed: %v", err))
		return
	}
	count := 0
	if v, ok := unitNodesResp["count"].(float64); ok {
		count = int(v)
	}
	client.GetLogger().Info(fmt.Sprintf("Found %d unit nodes", count))

	unitNodeUUIDs := make([]string, 0)
	if arr, ok := unitNodesResp["unit_nodes"].([]interface{}); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				if uuid, ok := m["uuid"].(string); ok && uuid != "" {
					unitNodeUUIDs = append(unitNodeUUIDs, uuid)
				}
			}
		}
	}
	if len(unitNodeUUIDs) == 0 {
		return
	}

	unitsResp, err := client.GetRESTClient().GetUnitsByNodes(ctx, unitNodeUUIDs, 100, 0)
	if err != nil {
		client.GetLogger().Warning(fmt.Sprintf("REST get_units_by_nodes failed: %v", err))
		return
	}
	unitCount := 0
	if v, ok := unitsResp["count"].(float64); ok {
		unitCount = int(v)
	}
	client.GetLogger().Info(fmt.Sprintf("Found %d units", unitCount))
	if arr, ok := unitsResp["units"].([]interface{}); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				name, _ := m["name"].(string)
				uuid, _ := m["uuid"].(string)
				client.GetLogger().Info(fmt.Sprintf("Unit: %s (UUID: %s)", name, uuid))
			}
		}
	}
}

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

			client.GetLogger().Debug(fmt.Sprintf("Get from input/pepeunit: %d", intValue), true)
		}
	}
}

func handleOutputMessages(client *pepeunit.PepeunitClient) {
	currentTime := time.Now()

	// Send data every DELAY_PUB_MSG from extras (fallback to PU_STATE_SEND_INTERVAL)
	delay, ok := client.GetSettings().GetInt("DELAY_PUB_MSG")
	if !ok || delay <= 0 {
		delay = client.GetSettings().PU_STATE_SEND_INTERVAL
	}
	if currentTime.Sub(lastOutputSendTime) >= time.Duration(delay)*time.Second {
		message := inc
		client.GetLogger().Debug(fmt.Sprintf("Send to output/pepeunit: %d", message), true)

		// Try to publish to sensor output topics
		ctx := context.Background()
		err := client.PublishToTopics(ctx, "output/pepeunit", strconv.Itoa(message))
		if err != nil {
			client.GetLogger().Error(fmt.Sprintf("Failed to publish message: %v", err))
		}

		// Update the last message send time
		lastOutputSendTime = currentTime
		inc++
	}
}

func main() {
	// Initialize the PepeUnit client
	client, err := pepeunit.NewPepeunitClient(pepeunit.PepeunitClientConfig{
		EnvFilePath:      "env.json",
		SchemaFilePath:   "schema.json",
		LogFilePath:      "log.json",
		EnableMQTT:       true,
		EnableREST:       true,
		CycleSpeed:       1 * time.Second, // 1 second cycle
		RestartMode:      pepeunit.RestartModeRestartExec,
		FFVersionCheckEnable: true,
	})

	if err != nil {
		log.Fatalf("Failed to create PepeUnit client: %v", err)
	}

	// Test work pepeunit storage
	testSetGetStorage(client)

	// Test get edged units by output topic
	testGetUnits(client)

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

```


## API Reference

### PepeunitClient

Main client for interacting with the PepeUnit platform.

| Method | Description |
|--------|-------------|
| `GetUnitUUID()` | Extracts unit UUID from the auth token. |
| `SetCycleSpeed(speed)` | Sets the main cycle execution interval. |
| `UpdateDeviceProgram(ctx, archivePath)` | Extracts and applies update archive to the unit directory. |
| `GetSystemState()` | Returns system state (timestamp, memory, CPU freq, version). |
| `SetMQTTInputHandler(handler)` | Sets a combined MQTT input handler (base + custom). |
| `UpdateBinaryFromURL(ctx, firmwareURL)` | Downloads new binary and atomically replaces the current executable. |
| `DownloadUpdate(ctx, archivePath)` | Downloads firmware update archive via REST. |
| `DownloadEnv(ctx, filePath)` | Downloads environment config and reloads settings. |
| `DownloadSchema(ctx, filePath)` | Downloads schema and resubscribes MQTT topics. |
| `SetStateStorage(ctx, state)` | Saves state to server storage. |
| `GetStateStorage(ctx)` | Retrieves state from server storage. |
| `PerformUpdate(ctx)` | Full update cycle: download, apply, cleanup. |
| `SubscribeAllSchemaTopics(ctx)` | Subscribes to all input topics defined in schema. |
| `PublishToTopics(ctx, topicKey, message)` | Publishes a message to all topics by key. |
| `RunMainCycle(ctx, outputHandler)` | Runs the main loop with periodic output handling. |
| `SetOutputHandler(outputHandler)` | Sets custom output handler for the main loop. |
| `SetCustomUpdateHandler(handler)` | Sets custom handler for program update via MQTT. |
| `StopMainCycle()` | Stops the main loop. |
| `GetSettings()` | Returns the settings manager. |
| `GetSchema()` | Returns the schema manager. |
| `GetLogger()` | Returns the logger. |
| `GetMQTTClient()` | Returns the MQTT client. |
| `GetRESTClient()` | Returns the REST client. |

### SchemaManager

| Method | Description |
|--------|-------------|
| `UpdateFromFile()` | Reloads schema data from file. |
| `UpdateSchema(schemaDict)` | Replaces schema data and writes to file. |
| `GetInputBaseTopic()` | Returns base input topics mapping. |
| `GetOutputBaseTopic()` | Returns base output topics mapping. |
| `GetInputTopic()` | Returns input topics mapping. |
| `GetOutputTopic()` | Returns output topics mapping. |
| `FindTopicByUnitNode(searchValue, searchType, searchScope)` | Finds topic key by unit node UUID or full topic URL within a scope. |

### Settings

| Method | Description |
|--------|-------------|
| `LoadFromFile()` | Loads settings from env file (no-op if missing). |
| `GetEnvValues()` | Returns parsed env values as a map. |
| `UpdateEnvFile(newEnvFilePath)` | Replaces env file and reloads settings. |
| `Update(updates)` | Updates settings from a map. |
| `Set(key, value)` | Sets a setting or extra value. |
| `Get(key)` | Gets a setting or extra value with presence flag. |
| `GetString(key)` | Gets a setting as string with presence flag. |
| `GetInt(key)` | Gets a setting as int with presence flag. |
| `Has(key)` | Checks if a key exists (including extras). |
| `All()` | Returns all settings including extras. |
| `UnitUUID()` | Extracts unit UUID from JWT in `PU_AUTH_TOKEN`. |

### Cipher (AES-GCM)

The keyB64 can be 16, 24, or 32 bits long.

| Method | Description |
|--------|-------------|
| `AESGCMEncode(data, keyB64)` | Encrypts UTF-8 `data` using AES-GCM with 12-byte random nonce. Returns `base64(nonce).base64(ciphertext)` joined by `.`. `keyB64` must decode to 16/24/32 bytes. |
| `AESGCMDecode(encoded, keyB64)` | Decrypts `encoded` string in format `base64(nonce).base64(ciphertext)` using AES-GCM and returns UTF-8 plaintext. |

### Logger

| Method | Description |
|--------|-------------|
| `Debug(message, fileOnly...)` | Logs a debug-level message. |
| `Info(message, fileOnly...)` | Logs an info-level message. |
| `Warning(message, fileOnly...)` | Logs a warning-level message. |
| `Error(message, fileOnly...)` | Logs an error-level message. |
| `Critical(message, fileOnly...)` | Logs a critical-level message. |
| `GetFullLog()` | Returns all log entries in NDJSON-compatible format. |
| `ResetLog()` | Clears log file contents. |
| `SetMQTTClient(mqttClient)` | Sets MQTT client used for log publishing. |

### FileManager

| Method | Description |
|--------|-------------|
| `FileExists(filePath)` | Checks if a file exists. |
| `ReadJSON(filePath)` | Reads JSON or JSON-encoded string file to map. |
| `WriteJSON(filePath, data)` | Writes JSON with indentation (atomic). |
| `CopyFile(srcPath, destPath)` | Copies file contents. |
| `CopyDirectoryContents(srcDir, destDir)` | Recursively copies directory contents. |
| `ExtractTarGz(archivePath, destDir)` | Extracts `.tar.gz` or zlib-compressed tar. |
| `AppendToJSONList(filePath, item)` | Appends item to JSON array file (supports wrapped format). |
| `AppendNDJSONWithLimit(filePath, item, maxLines)` | Appends to NDJSON with max lines trim. |
| `IterNDJSON(filePath)` | Iterates NDJSON file to slice of objects. |
| `TrimNDJSON(filePath, maxLines)` | Trims NDJSON to last `max_lines` lines. |
| `CreateTarGz(sourceDir, archivePath)` | Creates `.tar.gz` from directory. |

### PepeunitMQTTClient

| Method | Description |
|--------|-------------|
| `Connect(ctx)` | Connects to MQTT broker with auto-reconnect. |
| `Disconnect(ctx)` | Disconnects from MQTT broker. |
| `SubscribeTopics(topics)` | Subscribes to multiple topics. |
| `UnsubscribeTopics(topics)` | Unsubscribes from multiple topics. |
| `Publish(topic, message)` | Publishes a message (QoS 1, non-retained). |
| `SetInputHandler(handler)` | Sets handler for incoming messages. |
| `IsConnected()` | Returns connection state. |
| `GetClient()` | Returns underlying `paho.mqtt.golang` client. |

### PepeunitRESTClient

| Method | Description |
|--------|-------------|
| `DownloadUpdate(ctx, filePath)` | Downloads firmware update archive. |
| `DownloadEnv(ctx, filePath)` | Downloads environment configuration file. |
| `DownloadSchema(ctx, filePath)` | Downloads schema configuration file. |
| `DownloadFileFromURL(ctx, url, filePath)` | Downloads file from external URL. |
| `SetStateStorage(ctx, state)` | Saves state to server storage. |
| `GetStateStorage(ctx)` | Retrieves state from server storage. |
| `GetInputByOutput(ctx, topic, limit, offset)` | Gets input unit nodes linked to an output topic URL. |
| `GetUnitsByNodes(ctx, unitNodeUUIDs, limit, offset)` | Gets units by unit node UUIDs. |
| `SetHTTPClient(httpClient)` | Sets custom HTTP client. |
| `GetHTTPClient()` | Returns current HTTP client. |

### Enums

| Entity | Key | Description |
|--------|-----|-------------|
| `LogLevel` | `Debug` | Debug level logging. |
| `LogLevel` | `Info` | Information level logging. |
| `LogLevel` | `Warning` | Warning level logging. |
| `LogLevel` | `Error` | Error level logging. |
| `LogLevel` | `Critical` | Critical level logging. |
| `SearchTopicType` | `unit_node_uuid` | Search by unit node UUID. |
| `SearchTopicType` | `full_name` | Search by full topic name. |
| `SearchScope` | `all` | Search in all topics. |
| `SearchScope` | `input` | Search only input topics. |
| `SearchScope` | `output` | Search only output topics. |
| `DestinationTopicType` | `input_base_topic` | Base input topics section. |
| `DestinationTopicType` | `output_base_topic` | Base output topics section. |
| `DestinationTopicType` | `input_topic` | Regular input topics section. |
| `DestinationTopicType` | `output_topic` | Regular output topics section. |
| `BaseInputTopicType` | `update/pepeunit` | Update command topic. |
| `BaseInputTopicType` | `env_update/pepeunit` | Environment update command topic. |
| `BaseInputTopicType` | `schema_update/pepeunit` | Schema update command topic. |
| `BaseInputTopicType` | `log_sync/pepeunit` | Log synchronization command topic. |
| `BaseOutputTopicType` | `log/pepeunit` | Log output topic. |
| `BaseOutputTopicType` | `state/pepeunit` | State output topic. |
| `RestartMode` | `restart_popen` | Restart using separate process (`exec.Command`). |
| `RestartMode` | `restart_exec` | Replace current process using `syscall.Exec`. |
| `RestartMode` | `env_schema_only` | Update only env and schema without restart. |
| `RestartMode` | `no_restart` | Extract archive without restart or updates. |

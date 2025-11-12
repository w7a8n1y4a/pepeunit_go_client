## PepeUnit Go Client

A cross-platform Go library for integrating with the PepeUnit IoT platform. This client mirrors the Python implementation, providing MQTT and REST functionality for device communications, configuration updates, and state management. The only public entry point is `PepeunitClient`.

### Installation

```bash
# Add module to your project
go get github.com/w7a8n1y4a/pepeunit_go_client

# (optional) tidy deps
go mod tidy
```

### Examples

#### Basic Usage

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

// Global variable to track last message send time
var lastOutputSendTime time.Time
var inc int

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

			ctx := context.Background()
			if intValue < 10 {
				// Store simple state string, matching Python client behavior
				state := "This line is saved in Pepeunit Instance"
				if client.GetRESTClient() != nil {
					if err := client.GetRESTClient().SetStateStorage(ctx, state); err != nil {
						client.GetLogger().Error(fmt.Sprintf("Failed set state: %v", err))
					} else {
						client.GetLogger().Info("Success set state")
					}
				}
			}

			if intValue > 10 && intValue < 20 {
				if client.GetRESTClient() != nil {
					if state, err := client.GetRESTClient().GetStateStorage(ctx); err == nil {
						client.GetLogger().Info(fmt.Sprintf("Success get state: %s", state))
					} else {
						client.GetLogger().Error(fmt.Sprintf("Failed get state: %v", err))
					}
				}
			}

			client.GetLogger().Debug(fmt.Sprintf("Get from input/pepeunit: %d", intValue), true)
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
		SkipVersionCheck: true,
	})

	if err != nil {
		log.Fatalf("Failed to create PepeUnit client: %v", err)
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

```

#### Advanced Usage: Restart Modes

```go
productionCfg := pepeunit.PepeunitClientConfig{
    EnvFilePath: "env.json", SchemaFilePath: "schema.json", LogFilePath: "log.json",
    EnableMQTT: true, EnableREST: true, RestartMode: pepeunit.RestartModeRestartExec,
}

reliableCfg := pepeunit.PepeunitClientConfig{
    EnvFilePath: "env.json", SchemaFilePath: "schema.json", LogFilePath: "log.json",
    EnableMQTT: true, EnableREST: true, RestartMode: pepeunit.RestartModeRestartPopen,
}

configOnlyCfg := pepeunit.PepeunitClientConfig{
    EnvFilePath: "env.json", SchemaFilePath: "schema.json", LogFilePath: "log.json",
    EnableMQTT: true, EnableREST: true, RestartMode: pepeunit.RestartModeEnvSchemaOnly,
}

manualCfg := pepeunit.PepeunitClientConfig{
    EnvFilePath: "env.json", SchemaFilePath: "schema.json", LogFilePath: "log.json",
    EnableMQTT: true, EnableREST: true, RestartMode: pepeunit.RestartModeNoRestart,
}
```

### API Reference

#### PepeunitClient

The main client providing PepeUnit integration.

Constructor:

```go
client, err := pepeunit.NewPepeunitClient(pepeunit.PepeunitClientConfig{
    EnvFilePath:    "env.json",
    SchemaFilePath: "schema.json",
    LogFilePath:    "log.json",
    EnableMQTT:     true,
    EnableREST:     true,
    CycleSpeed:     100 * time.Millisecond,
    RestartMode:    pepeunit.RestartModeRestartExec,
    SkipVersionCheck: false, // optional
})
```

Properties/methods (selected):

- `GetUnitUUID() (string, error)`: Extract device UUID from JWT token
- `SetCycleSpeed(speed time.Duration) error`: Set main loop cadence
- `GetSystemState() map[string]interface{}`: System status (memory, CPU, version)
- `UpdateDeviceProgram(ctx, archivePath string) error`: Apply update from tar.gz
- `UpdateBinaryFromURL(ctx context.Context, firmwareURL string) error`: Replace running binary
- `RunMainCycle(ctx, outputHandler func(*PepeunitClient))`: Start main loop
- `StopMainCycle()`: Stop loop
- `SetOutputHandler(handler func(*PepeunitClient))`
- `SetCustomUpdateHandler(handler func(*PepeunitClient, string) error)`

Restart modes (`RestartMode`):

- `RestartModeRestartPopen`: Spawn a new process and exit current
- `RestartModeRestartExec` (default): Replace current process via exec
- `RestartModeEnvSchemaOnly`: Update env/schema without restarting
- `RestartModeNoRestart`: Only extract/update artifacts

Usage:

```go
client, _ := pepeunit.NewPepeunitClient(pepeunit.PepeunitClientConfig{
    EnvFilePath: "env.json", SchemaFilePath: "schema.json", LogFilePath: "log.json",
    RestartMode: pepeunit.RestartModeRestartExec,
})
```

#### MQTT Methods (EnableMQTT=true)

- `SetMQTTInputHandler(handler MQTTInputHandler)`
- `SubscribeAllSchemaTopics(ctx context.Context) error`
- `PublishToTopics(ctx context.Context, topicKey, message string) error`

#### REST Methods (EnableREST=true)

- `DownloadUpdate(ctx, archivePath string) error`
- `DownloadEnv(ctx, filePath string) error`
- `DownloadSchema(ctx, filePath string) error`
- `SetStateStorage(ctx context.Context, state string) error`
- `GetStateStorage(ctx context.Context) (string, error)`

#### Combined (MQTT + REST)

- `PerformUpdate(ctx context.Context) error`

#### MQTT Client (`PepeunitClient.GetMQTTClient()`)

Implements `MQTTClient` interface:

- `Connect(ctx context.Context) error`
- `Disconnect(ctx context.Context) error`
- `SubscribeTopics(topics []string) error`
- `UnsubscribeTopics(topics []string) error`
- `Publish(topic, message string) error`
- `SetInputHandler(handler MQTTInputHandler)`

#### REST Client (`PepeunitClient.GetRESTClient()`)

Implements `RESTClient` interface:

- `DownloadUpdate(ctx context.Context, filePath string) error`
- `DownloadEnv(ctx context.Context, filePath string) error`
- `DownloadSchema(ctx context.Context, filePath string) error`
- `DownloadFileFromURL(ctx context.Context, url, filePath string) error`
- `SetStateStorage(ctx context.Context, state string) error`
- `GetStateStorage(ctx context.Context) (string, error)`

Dependency injection (optional):

```go
customClient, _ := pepeunit.NewPepeunitClient(pepeunit.PepeunitClientConfig{
    EnvFilePath: "env.json", SchemaFilePath: "schema.json", LogFilePath: "log.json",
    EnableMQTT: true, EnableREST: true,
    MQTTClient: myMQTTImpl,    // implements pepeunit.MQTTClient
    RESTClient: myRESTImpl,    // implements pepeunit.RESTClient
})
```

#### Settings

Environment-backed configuration loaded from `env.json`:

- `PEPEUNIT_URL`, `PEPEUNIT_APP_PREFIX`, `PEPEUNIT_API_ACTUAL_PREFIX`, `HTTP_TYPE`
- `MQTT_URL`, `MQTT_PORT`, `PEPEUNIT_TOKEN`
- `SYNC_ENCRYPT_KEY`, `SECRET_KEY`, `COMMIT_VERSION`
- `PING_INTERVAL`, `STATE_SEND_INTERVAL`, `MINIMAL_LOG_LEVEL`, `MIN_LOG_LEVEL`
- `MAX_LOG_LENGTH`

Helpers:

- `LoadFromFile() error`
- `GetEnvValues() (map[string]interface{}, error)`
- `UpdateEnvFile(newEnvFilePath string) error`
- `Update(updates map[string]interface{}) error`

#### Schema

Schema manager for MQTT topic configuration loaded from `schema.json`:

- `GetInputBaseTopic() map[string][]string`
- `GetOutputBaseTopic() map[string][]string`
- `GetInputTopic() map[string][]string`
- `GetOutputTopic() map[string][]string`
- `FindTopicByUnitNode(searchValue string, searchType SearchTopicType, searchScope SearchScope) (string, error)`

Search enums:

- `SearchTopicTypeUnitNodeUUID`, `SearchTopicTypeFullName`
- `SearchScopeAll`, `SearchScopeInput`, `SearchScopeOutput`

#### Logger

File-backed logger with optional MQTT publishing:

- Levels: `Debug`, `Info`, `Warning`, `Error`, `Critical`
- Methods: `Debug`, `Info`, `Warning`, `Error`, `Critical`, `GetFullLog() []map[string]interface{}`, `ResetLog()`

### Error Handling

The library uses standard Go `error` returns and does not hide failures. Typical errors:

- `invalid JWT token format`, `failed to decode/parse JWT payload`
- `MQTT client is not enabled or available`, `REST client is not enabled or available`
- HTTP errors with non-200 status codes from REST endpoints

### Dependencies

- MQTT: `github.com/eclipse/paho.mqtt.golang`
- System stats: `github.com/shirou/gopsutil/v3`

### License

GNU Affero General Public License v3 (AGPL-3.0-or-later)

### Links

- Homepage: `https://git.pepemoss.com/pepe/pepeunit/libs/pepeunit_go_client`
- Issues: `https://git.pepemoss.com/pepe/pepeunit/libs/pepeunit_go_client/-/issues`


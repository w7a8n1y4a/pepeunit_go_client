# PepeUnit Go Client

A cross-platform Go library for integrating with the PepeUnit IoT platform. This library provides MQTT and REST client functionality for managing device communications, configurations, and state management.

## Installation

```bash
go get github.com/w7an1y4a/pepeunit_go_client
```

## Features

- **MQTT Client**: Full MQTT broker integration with automatic reconnection
- **REST Client**: HTTP-based API client for device management
- **Configuration Management**: JSON-based settings and schema management
- **Logging System**: File and MQTT-based logging with configurable levels
- **Device Updates**: Support for OTA updates with multiple restart modes
- **State Management**: Device state synchronization with PepeUnit storage
- **Cross-platform**: Works on Windows, Linux, macOS, and other Go-supported platforms

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    pepeunit "github.com/w7an1y4a/pepeunit_go_client"
)

func main() {
    // Create client configuration
    config := pepeunit.PepeunitClientConfig{
        EnvFilePath:    "env.json",
        SchemaFilePath: "schema.json", 
        LogFilePath:    "log.json",
        EnableMQTT:     true,
        EnableREST:     true,
        CycleSpeed:     1 * time.Second,
        RestartMode:    pepeunit.RestartModeRestartExec,
    }

    // Initialize client
    client, err := pepeunit.NewPepeunitClient(config)
    if err != nil {
        log.Fatal(err)
    }

    // Set up MQTT input handler
    client.SetMQTTInputHandler(func(msg pepeunit.MQTTMessage) {
        // Handle incoming MQTT messages
        client.GetLogger().Info("Received message: " + string(msg.Payload))
    })

    // Connect to MQTT broker
    ctx := context.Background()
    err = client.GetMQTTClient().Connect(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Subscribe to topics
    err = client.SubscribeAllSchemaTopics(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Run main cycle
    client.RunMainCycle(ctx, func(client *pepeunit.PepeunitClient) {
        // Custom output handler
        client.GetLogger().Info("Main cycle running")
    })
}
```

### Advanced Usage

#### Different Restart Modes

```go
// Fast restart (default) - replaces current process
client := pepeunit.NewPepeunitClient(pepeunit.PepeunitClientConfig{
    EnvFilePath:    "env.json",
    SchemaFilePath: "schema.json",
    LogFilePath:    "log.json",
    RestartMode:    pepeunit.RestartModeRestartExec,
})

// Subprocess restart - creates new process
client := pepeunit.NewPepeunitClient(pepeunit.PepeunitClientConfig{
    EnvFilePath:    "env.json", 
    SchemaFilePath: "schema.json",
    LogFilePath:    "log.json",
    RestartMode:    pepeunit.RestartModeRestartPopen,
})

// Configuration-only updates (no downtime)
client := pepeunit.NewPepeunitClient(pepeunit.PepeunitClientConfig{
    EnvFilePath:    "env.json",
    SchemaFilePath: "schema.json", 
    LogFilePath:    "log.json",
    RestartMode:    pepeunit.RestartModeEnvSchemaOnly,
})

// Manual update control
client := pepeunit.NewPepeunitClient(pepeunit.PepeunitClientConfig{
    EnvFilePath:    "env.json",
    SchemaFilePath: "schema.json",
    LogFilePath:    "log.json", 
    RestartMode:    pepeunit.RestartModeNoRestart,
})
```

#### Custom MQTT and REST Clients

```go
// Create custom MQTT client
type CustomMQTTClient struct {
    *pepeunit.AbstractMQTTClient
}

func (c *CustomMQTTClient) Connect(ctx context.Context) error {
    // Custom connection logic
    return nil
}

// Create custom REST client  
type CustomRESTClient struct {
    *pepeunit.AbstractRESTClient
}

func (c *CustomRESTClient) DownloadUpdate(ctx context.Context, unitUUID, filePath string) error {
    // Custom download logic
    return nil
}

// Use custom clients
config := pepeunit.PepeunitClientConfig{
    EnvFilePath: "env.json",
    SchemaFilePath: "schema.json",
    LogFilePath: "log.json",
    EnableMQTT: true,
    EnableREST: true,
    MQTTClient: &CustomMQTTClient{},
    RESTClient: &CustomRESTClient{},
}
```

## Configuration Files

### Environment Configuration (env.json)

```json
{
  "PEPEUNIT_URL": "your.pepeunit.com",
  "PEPEUNIT_APP_PREFIX": "/api",
  "PEPEUNIT_API_ACTUAL_PREFIX": "/v1",
  "HTTP_TYPE": "https",
  "MQTT_URL": "mqtt.pepeunit.com",
  "MQTT_PORT": 1883,
  "PEPEUNIT_TOKEN": "your-jwt-token",
  "SYNC_ENCRYPT_KEY": "your-encrypt-key",
  "SECRET_KEY": "your-secret-key",
  "COMMIT_VERSION": "v1.0.0",
  "PING_INTERVAL": 30,
  "STATE_SEND_INTERVAL": 300,
  "MINIMAL_LOG_LEVEL": "Debug",
  "DELAY_PUB_MSG": 300
}
```

### Schema Configuration (schema.json)

```json
{
  "input_base_topic": {
    "update/pepeunit": ["domain.com/uuid/update/pepeunit"],
    "env_update/pepeunit": ["domain.com/uuid/env_update/pepeunit"]
  },
  "output_base_topic": {
    "log/pepeunit": ["domain.com/uuid/log/pepeunit"],
    "state/pepeunit": ["domain.com/uuid/state/pepeunit"]
  },
  "input_topic": {
    "input/pepeunit": ["domain.com/uuid/input/pepeunit"]
  },
  "output_topic": {
    "output/pepeunit": ["domain.com/uuid/output/pepeunit"]
  }
}
```

## API Reference

### PepeunitClient

The main client class providing all functionality for PepeUnit integration.

#### Constructor

```go
func NewPepeunitClient(config PepeunitClientConfig) (*PepeunitClient, error)
```

#### Core Methods

- **`GetUnitUUID() (string, error)`**: Get device UUID from JWT token
- **`SetCycleSpeed(speed time.Duration) error`**: Set main cycle execution speed
- **`GetSystemState() map[string]interface{}`**: Get current system status
- **`UpdateDeviceProgram(ctx context.Context, archivePath string) error`**: Update device from tar.gz archive
- **`RunMainCycle(ctx context.Context, outputHandler func(*PepeunitClient))`**: Start main application loop
- **`StopMainCycle()`**: Stop main application loop

#### MQTT Methods

- **`SetMQTTInputHandler(handler MQTTInputHandler)`**: Set input message handler
- **`SubscribeAllSchemaTopics(ctx context.Context) error`**: Subscribe to all schema topics
- **`PublishToTopics(ctx context.Context, topicKey, message string) error`**: Publish message to topic group

#### REST Methods

- **`DownloadUpdate(ctx context.Context, archivePath string) error`**: Download firmware update
- **`DownloadEnv(ctx context.Context, filePath string) error`**: Download environment config
- **`DownloadSchema(ctx context.Context, filePath string) error`**: Download schema config
- **`SetStateStorage(ctx context.Context, state map[string]interface{}) error`**: Upload state to storage
- **`GetStateStorage(ctx context.Context) (map[string]interface{}, error)`**: Retrieve state from storage

#### Combined Methods

- **`PerformUpdate(ctx context.Context) error`**: Complete update cycle (download + extract + apply)

### Restart Modes

The `RestartMode` controls how the device behaves when `UpdateDeviceProgram()` is called:

- **`RestartModeRestartPopen`**: Creates new process via `exec.Command().Start()`, then exits current process
  - ✅ Full process isolation and reliable restart
  - ⚠️ Brief gap between old and new process
  
- **`RestartModeRestartExec`** (default): Replaces current process using `syscall.Exec()`
  - ✅ Fast restart, preserves process ID
  - ⚠️ May not release all resources properly
  
- **`RestartModeEnvSchemaOnly`**: Updates configuration without restarting
  - ✅ No downtime, fast configuration updates
  - ⚠️ Code changes not applied, only env.json and schema.json
  
- **`RestartModeNoRestart`**: Only extracts archive, no updates or restarts
  - ✅ Full control over update process
  - ⚠️ Manual intervention required

### Logging

The logger provides file and MQTT-based logging with configurable levels:

```go
client.GetLogger().Debug("Debug message")
client.GetLogger().Info("Info message") 
client.GetLogger().Warning("Warning message")
client.GetLogger().Error("Error message")
client.GetLogger().Critical("Critical message")

// Get full log history
logs := client.GetLogger().GetFullLog()

// Reset logs
client.GetLogger().ResetLog()
```

### System Monitoring

The client automatically monitors system resources:

```go
state := client.GetSystemState()
// Returns: {
//   "millis": 1640995200000,
//   "mem_free": 8589934592,
//   "mem_alloc": 4294967296, 
//   "freq": 2400,
//   "commit_version": "v1.0.0"
// }
```

## Error Handling

The library uses standard Go error handling:

```go
client, err := pepeunit.NewPepeunitClient(config)
if err != nil {
    log.Fatal(err)
}

err = client.GetMQTTClient().Connect(ctx)
if err != nil {
    log.Printf("MQTT connection failed: %v", err)
    // Handle connection error
}
```

## Dependencies

### Core Dependencies
- `github.com/eclipse/paho.mqtt.golang v1.4.3` - MQTT client
- `github.com/google/uuid v1.6.0` - UUID handling
- `github.com/shirou/gopsutil/v3 v3.23.12` - System monitoring

### Optional Dependencies
- Standard library packages: `net/http`, `encoding/json`, `os`, `time`, etc.

## License

GNU Affero General Public License v3 (AGPL-3.0-or-later)

## Links

- [Homepage](https://git.pepemoss.com/pepe/pepeunit/libs/pepeunit_go_client)
- [Issues](https://git.pepemoss.com/pepe/pepeunit/libs/pepeunit_go_client/-/issues)
- [Documentation](https://git.pepemoss.com/pepe/pepeunit/libs/pepeunit_go_client/-/wikis)

## Example

See the `example/` directory for a complete working example with configuration files.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## Changelog

### v0.1.0
- Initial release
- MQTT and REST client support
- Configuration management
- Logging system
- Device update functionality
- Cross-platform support

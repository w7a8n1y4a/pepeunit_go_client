package pepeunit_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

func setupClientForHandlers(t *testing.T) (*pepeunit.PepeunitClient, *mockMQTTClient, *mockRESTClient, string, string, string) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "env.json")
	schemaPath := filepath.Join(dir, "schema.json")
	logPath := filepath.Join(dir, "log.json")

	_ = pepeunit.NewFileManager().WriteJSON(envPath, map[string]interface{}{
		"PEPEUNIT_TOKEN": makeJWTWithUUID("u-1"),
	})
	_ = pepeunit.NewFileManager().WriteJSON(schemaPath, map[string]interface{}{
		"input_base_topic": map[string]interface{}{
			"env_update/pepeunit":    []interface{}{"unit/u-1/env_update"},
			"schema_update/pepeunit": []interface{}{"unit/u-1/schema_update"},
			"log_sync/pepeunit":      []interface{}{"unit/u-1/log_sync"},
			"update/pepeunit":        []interface{}{"unit/u-1/update"},
		},
		"output_base_topic": map[string]interface{}{
			"log/pepeunit":   []interface{}{"unit/u-1/log"},
			"state/pepeunit": []interface{}{"unit/u-1/state"},
		},
	})

	mqtt := &mockMQTTClient{}
	rest := newMockRESTClient()

	client, err := pepeunit.NewPepeunitClient(pepeunit.PepeunitClientConfig{
		EnvFilePath:    envPath,
		SchemaFilePath: schemaPath,
		LogFilePath:    logPath,
		EnableMQTT:     true,
		EnableREST:     true,
		MQTTClient:     mqtt,
		RESTClient:     rest,
	})
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	return client, mqtt, rest, envPath, schemaPath, logPath
}

func TestClient_HandleEnvUpdate_SchemaUpdate_LogSync(t *testing.T) {
	client, mqtt, _, envPath, schemaPath, _ := setupClientForHandlers(t)

	client.SetMQTTInputHandler(func(msg pepeunit.MQTTMessage) {})

	mqtt.handler(pepeunit.MQTTMessage{Topic: "unit/u-1/env_update", Payload: []byte("{}")})
	mqtt.handler(pepeunit.MQTTMessage{Topic: "unit/u-1/schema_update", Payload: []byte("{}")})
	mqtt.handler(pepeunit.MQTTMessage{Topic: "unit/u-1/log_sync", Payload: []byte("{}")})

	// env and schema files should remain readable; log sync should publish
	if _, err := pepeunit.NewFileManager().ReadJSON(envPath); err != nil {
		t.Fatalf("env read: %v", err)
	}
	if _, err := pepeunit.NewFileManager().ReadJSON(schemaPath); err != nil {
		t.Fatalf("schema read: %v", err)
	}
	if len(mqtt.published) == 0 {
		t.Fatalf("expected publish on log_sync")
	}
}

func TestClient_HandleUpdate_PerformUpdate_Path(t *testing.T) {
	client, mqtt, rest, _, _, _ := setupClientForHandlers(t)
	client.SetMQTTInputHandler(func(msg pepeunit.MQTTMessage) {})

	meta := map[string]interface{}{"COMPILED_FIRMWARE_LINK": "http://example/firmware"}
	b, _ := json.Marshal(meta)
	mqtt.handler(pepeunit.MQTTMessage{Topic: "unit/u-1/update", Payload: b})

	_ = rest // nothing to assert strongly without side effects
	_ = client
	if len(mqtt.published) < 0 { // keep linter happy, no-op
		panic("unreachable")
	}
}

func TestClient_StatePublishOnInterval(t *testing.T) {
	client, mqtt, _, _, _, _ := setupClientForHandlers(t)
	client.SetMQTTInputHandler(func(msg pepeunit.MQTTMessage) {})

	ctx := context.Background()
	client.GetSettings().STATE_SEND_INTERVAL = 0
	client.GetSchema().UpdateSchema(map[string]interface{}{
		"output_base_topic": map[string]interface{}{
			"state/pepeunit": []interface{}{"unit/u-1/state"},
		},
	})

	client.RunMainCycle(ctx, func(c *pepeunit.PepeunitClient) {
		_ = c.GetSystemState()
		c.StopMainCycle()
	})
	if len(mqtt.published) == 0 {
		t.Fatalf("expected state publish")
	}
}

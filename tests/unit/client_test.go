package pepeunit_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

func makeEnvSchema(dir string) (string, string) {
	envPath := filepath.Join(dir, "env.json")
	schemaPath := filepath.Join(dir, "schema.json")

	_ = pepeunit.NewFileManager().WriteJSON(envPath, map[string]interface{}{
		"PEPEUNIT_TOKEN":      makeJWTWithUUID("unit-uuid-1"),
		"STATE_SEND_INTERVAL": 1,
	})

	_ = pepeunit.NewFileManager().WriteJSON(schemaPath, map[string]interface{}{
		"output_base_topic": map[string]interface{}{
			"state/pepeunit": []interface{}{"unit/unit-uuid-1/state"},
			"log/pepeunit":   []interface{}{"unit/unit-uuid-1/log"},
		},
		"input_base_topic": map[string]interface{}{
			"env_update/pepeunit":    []interface{}{"unit/unit-uuid-1/env_update"},
			"schema_update/pepeunit": []interface{}{"unit/unit-uuid-1/schema_update"},
			"log_sync/pepeunit":      []interface{}{"unit/unit-uuid-1/log_sync"},
		},
		"input_topic":  map[string]interface{}{},
		"output_topic": map[string]interface{}{},
	})
	return envPath, schemaPath
}

func TestClient_UUID_Subscribe_Publish(t *testing.T) {
	dir := t.TempDir()
	envPath, schemaPath := makeEnvSchema(dir)
	logPath := filepath.Join(dir, "log.json")

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
		t.Fatalf("new client: %v", err)
	}

	uuid, err := client.GetUnitUUID()
	if err != nil || uuid != "unit-uuid-1" {
		t.Fatalf("uuid parse failed: %v %s", err, uuid)
	}

	ctx := context.Background()
	if err := client.SubscribeAllSchemaTopics(ctx); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if len(mqtt.subscribed) == 3 { // 3 input base topics
		// ok
	} else {
		t.Fatalf("expected 3 subscribed topics, got %d", len(mqtt.subscribed))
	}

	if err := client.PublishToTopics(ctx, "state/pepeunit", "{}"); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if len(mqtt.published) == 0 {
		t.Fatalf("expected mqtt publish to state topic")
	}

	// Ensure main cycle can be started and stopped via context
	runCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	client.RunMainCycle(runCtx, func(c *pepeunit.PepeunitClient) {})
}

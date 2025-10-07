package pepeunit_test

import (
	"context"
	"path/filepath"
	"testing"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

func TestClient_StorageAndDownloads(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "env.json")
	schemaPath := filepath.Join(dir, "schema.json")
	logPath := filepath.Join(dir, "log.json")

	_ = pepeunit.NewFileManager().WriteJSON(envPath, map[string]interface{}{
		"PEPEUNIT_TOKEN": makeJWTWithUUID("u-2"),
	})
	_ = pepeunit.NewFileManager().WriteJSON(schemaPath, map[string]interface{}{
		"input_base_topic":  map[string]interface{}{},
		"output_base_topic": map[string]interface{}{},
	})

	client, err := pepeunit.NewPepeunitClient(pepeunit.PepeunitClientConfig{
		EnvFilePath:    envPath,
		SchemaFilePath: schemaPath,
		LogFilePath:    logPath,
		EnableMQTT:     false,
		EnableREST:     true,
		RESTClient:     newMockRESTClient(),
	})
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	ctx := context.Background()

	if err := client.DownloadEnv(ctx, envPath); err != nil {
		t.Fatalf("download env: %v", err)
	}
	if err := client.DownloadSchema(ctx, schemaPath); err != nil {
		t.Fatalf("download schema: %v", err)
	}

	state := map[string]interface{}{"a": 1}
	if err := client.SetStateStorage(ctx, state); err != nil {
		t.Fatalf("set state: %v", err)
	}
	got, err := client.GetStateStorage(ctx)
	if err != nil || got["a"].(int) != 1 {
		t.Fatalf("get state mismatch: %v %v", err, got)
	}
}

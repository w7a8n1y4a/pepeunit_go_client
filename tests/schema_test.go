package tests

import (
	"strings"
	"testing"

	pepeunit "github.com/w7an1y4a/pepeunit_go_client"
)

func TestNewSchemaManager(t *testing.T) {
	// Create a temporary test file
	tempFile, err := CreateTempTestFile(TestSchemaData())
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer CleanupTempFile(tempFile)

	schema, err := pepeunit.NewSchemaManager(tempFile)
	if err != nil {
		t.Fatalf("Failed to create schema manager: %v", err)
	}

	if schema == nil {
		t.Fatal("Expected schema manager instance, got nil")
	}
}

func TestSchemaManagerGetInputBaseTopic(t *testing.T) {
	// Create a temporary test file
	tempFile, err := CreateTempTestFile(TestSchemaData())
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer CleanupTempFile(tempFile)

	schema, err := pepeunit.NewSchemaManager(tempFile)
	if err != nil {
		t.Fatalf("Failed to create schema manager: %v", err)
	}

	inputBaseTopic := schema.GetInputBaseTopic()
	if len(inputBaseTopic) == 0 {
		t.Error("Expected input base topics, got empty map")
	}

	if topics, ok := inputBaseTopic["update/pepeunit"]; !ok || len(topics) == 0 {
		t.Error("Expected update/pepeunit topic, not found")
	}
}

func TestSchemaManagerGetOutputBaseTopic(t *testing.T) {
	// Create a temporary test file
	tempFile, err := CreateTempTestFile(TestSchemaData())
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer CleanupTempFile(tempFile)

	schema, err := pepeunit.NewSchemaManager(tempFile)
	if err != nil {
		t.Fatalf("Failed to create schema manager: %v", err)
	}

	outputBaseTopic := schema.GetOutputBaseTopic()
	if len(outputBaseTopic) == 0 {
		t.Error("Expected output base topics, got empty map")
	}

	if topics, ok := outputBaseTopic["log/pepeunit"]; !ok || len(topics) == 0 {
		t.Error("Expected log/pepeunit topic, not found")
	}
}

func TestSchemaManagerGetInputTopic(t *testing.T) {
	// Create a temporary test file
	tempFile, err := CreateTempTestFile(TestSchemaData())
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer CleanupTempFile(tempFile)

	schema, err := pepeunit.NewSchemaManager(tempFile)
	if err != nil {
		t.Fatalf("Failed to create schema manager: %v", err)
	}

	inputTopic := schema.GetInputTopic()
	if len(inputTopic) == 0 {
		t.Error("Expected input topics, got empty map")
	}

	if topics, ok := inputTopic["input/pepeunit"]; !ok || len(topics) == 0 {
		t.Error("Expected input/pepeunit topic, not found")
	}
}

func TestSchemaManagerGetOutputTopic(t *testing.T) {
	// Create a temporary test file
	tempFile, err := CreateTempTestFile(TestSchemaData())
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer CleanupTempFile(tempFile)

	schema, err := pepeunit.NewSchemaManager(tempFile)
	if err != nil {
		t.Fatalf("Failed to create schema manager: %v", err)
	}

	outputTopic := schema.GetOutputTopic()
	if len(outputTopic) == 0 {
		t.Error("Expected output topics, got empty map")
	}

	if topics, ok := outputTopic["output/pepeunit"]; !ok || len(topics) == 0 {
		t.Error("Expected output/pepeunit topic, not found")
	}
}

func TestSchemaManagerFindTopicByUnitNode(t *testing.T) {
	// Create a temporary test file
	tempFile, err := CreateTempTestFile(TestSchemaData())
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer CleanupTempFile(tempFile)

	schema, err := pepeunit.NewSchemaManager(tempFile)
	if err != nil {
		t.Fatalf("Failed to create schema manager: %v", err)
	}

	// Test finding by full name
	schemaData := TestSchemaData()
	inputTopics := schemaData["input_topic"].(map[string]interface{})
	expectedTopic := inputTopics["input/pepeunit"].([]interface{})[0].(string)

	topicName, err := schema.FindTopicByUnitNode(
		expectedTopic,
		pepeunit.SearchTopicTypeFullName,
		pepeunit.SearchScopeInput,
	)
	if err != nil {
		t.Fatalf("Failed to find topic by full name: %v", err)
	}

	if topicName != "input/pepeunit" {
		t.Errorf("Expected topic name 'input/pepeunit', got '%s'", topicName)
	}

	// Test finding by UUID (extract UUID from the topic)
	expectedTopic = inputTopics["input/pepeunit"].([]interface{})[0].(string)
	// Extract UUID from topic like "devunit.pepeunit.com/486bfea4-a056-4c15-a361-7bbe5fde66ae/pepeunit"
	parts := strings.Split(expectedTopic, "/")
	if len(parts) < 2 {
		t.Fatalf("Invalid topic format: %s", expectedTopic)
	}
	uuid := parts[1]

	topicName, err = schema.FindTopicByUnitNode(
		uuid,
		pepeunit.SearchTopicTypeUnitNodeUUID,
		pepeunit.SearchScopeInput,
	)
	if err != nil {
		t.Fatalf("Failed to find topic by UUID: %v", err)
	}

	if topicName != "input/pepeunit" {
		t.Errorf("Expected topic name 'input/pepeunit', got '%s'", topicName)
	}
}

func TestSchemaManagerFindTopicByUnitNodeNotFound(t *testing.T) {
	// Create a temporary test file
	tempFile, err := CreateTempTestFile(TestSchemaData())
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer CleanupTempFile(tempFile)

	schema, err := pepeunit.NewSchemaManager(tempFile)
	if err != nil {
		t.Fatalf("Failed to create schema manager: %v", err)
	}

	// Test finding non-existent topic
	_, err = schema.FindTopicByUnitNode(
		"nonexistent-topic",
		pepeunit.SearchTopicTypeFullName,
		pepeunit.SearchScopeInput,
	)
	if err == nil {
		t.Error("Expected error for non-existent topic, got nil")
	}
}

func TestSchemaManagerUpdateFromFile(t *testing.T) {
	// Create a temporary test file
	tempFile, err := CreateTempTestFile(TestSchemaData())
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer CleanupTempFile(tempFile)

	schema, err := pepeunit.NewSchemaManager(tempFile)
	if err != nil {
		t.Fatalf("Failed to create schema manager: %v", err)
	}

	err = schema.UpdateFromFile()
	if err != nil {
		t.Fatalf("Failed to update from file: %v", err)
	}
}

func TestSchemaManagerUpdateSchema(t *testing.T) {
	// Create a temporary test file
	tempFile, err := CreateTempTestFile(TestSchemaData())
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer CleanupTempFile(tempFile)

	schema, err := pepeunit.NewSchemaManager(tempFile)
	if err != nil {
		t.Fatalf("Failed to create schema manager: %v", err)
	}

	newSchema := map[string]interface{}{
		"input_topic": map[string]interface{}{
			"test_topic": []interface{}{"test.domain.com/uuid/test_topic"},
		},
	}

	err = schema.UpdateSchema(newSchema)
	if err != nil {
		t.Fatalf("Failed to update schema: %v", err)
	}

	// Verify the update
	inputTopic := schema.GetInputTopic()
	if _, ok := inputTopic["test_topic"]; !ok {
		t.Error("Expected test_topic to be present after update")
	}
}

package pepeunit_test

import (
	"path/filepath"
	"testing"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

func TestSchema_Load_Getters_FindTopic(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.json")

	schemaData := map[string]interface{}{
		"input_base_topic": map[string]interface{}{
			"update/pepeunit": []interface{}{"unit/123/update"},
		},
		"output_base_topic": map[string]interface{}{
			"state/pepeunit": []interface{}{"unit/123/state"},
		},
		"input_topic": map[string]interface{}{
			"t_in": []interface{}{"in/123/x"},
		},
		"output_topic": map[string]interface{}{
			"t_out": []interface{}{"out/123/x"},
		},
	}

	fm := pepeunit.NewFileManager()
	if err := fm.WriteJSON(schemaPath, schemaData); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sm, err := pepeunit.NewSchemaManager(schemaPath)
	if err != nil {
		t.Fatalf("new schema: %v", err)
	}

	if len(sm.GetInputBaseTopic()["update/pepeunit"]) != 1 {
		t.Fatalf("getter failed")
	}

	name, err := sm.FindTopicByUnitNode("123", pepeunit.SearchTopicTypeUnitNodeUUID, pepeunit.SearchScopeAll)
	if err != nil || name == "" {
		t.Fatalf("find by uuid failed")
	}

	name2, err := sm.FindTopicByUnitNode("out/123/x", pepeunit.SearchTopicTypeFullName, pepeunit.SearchScopeOutput)
	if err != nil || name2 != "t_out" {
		t.Fatalf("find by name failed")
	}
}

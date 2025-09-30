package pepeunit

import (
	"fmt"
	"strings"
)

// SchemaManager manages MQTT topic schema configuration
type SchemaManager struct {
	schemaFilePath string
	schemaData     map[string]interface{}
}

// NewSchemaManager creates a new schema manager
func NewSchemaManager(schemaFilePath string) (*SchemaManager, error) {
	sm := &SchemaManager{
		schemaFilePath: schemaFilePath,
	}

	err := sm.loadSchema()
	if err != nil {
		return nil, err
	}

	return sm, nil
}

// loadSchema loads the schema from the file
func (sm *SchemaManager) loadSchema() error {
	fm := NewFileManager()
	schemaData, err := fm.ReadJSON(sm.schemaFilePath)
	if err != nil {
		return err
	}

	sm.schemaData = schemaData
	return nil
}

// UpdateFromFile reloads the schema from the file
func (sm *SchemaManager) UpdateFromFile() error {
	return sm.loadSchema()
}

// UpdateSchema updates the schema with new data and saves to file
func (sm *SchemaManager) UpdateSchema(schemaDict map[string]interface{}) error {
	sm.schemaData = schemaDict
	fm := NewFileManager()
	return fm.WriteJSON(sm.schemaFilePath, schemaDict)
}

// GetInputBaseTopic returns the input base topics configuration
func (sm *SchemaManager) GetInputBaseTopic() map[string][]string {
	if data, ok := sm.schemaData[string(DestinationTopicTypeInputBaseTopic)].(map[string]interface{}); ok {
		result := make(map[string][]string)
		for key, value := range data {
			if topics, ok := value.([]interface{}); ok {
				topicStrings := make([]string, len(topics))
				for i, topic := range topics {
					topicStrings[i] = fmt.Sprintf("%v", topic)
				}
				result[key] = topicStrings
			}
		}
		return result
	}
	return make(map[string][]string)
}

// GetOutputBaseTopic returns the output base topics configuration
func (sm *SchemaManager) GetOutputBaseTopic() map[string][]string {
	if data, ok := sm.schemaData[string(DestinationTopicTypeOutputBaseTopic)].(map[string]interface{}); ok {
		result := make(map[string][]string)
		for key, value := range data {
			if topics, ok := value.([]interface{}); ok {
				topicStrings := make([]string, len(topics))
				for i, topic := range topics {
					topicStrings[i] = fmt.Sprintf("%v", topic)
				}
				result[key] = topicStrings
			}
		}
		return result
	}
	return make(map[string][]string)
}

// GetInputTopic returns the input topics configuration
func (sm *SchemaManager) GetInputTopic() map[string][]string {
	if data, ok := sm.schemaData[string(DestinationTopicTypeInputTopic)].(map[string]interface{}); ok {
		result := make(map[string][]string)
		for key, value := range data {
			if topics, ok := value.([]interface{}); ok {
				topicStrings := make([]string, len(topics))
				for i, topic := range topics {
					topicStrings[i] = fmt.Sprintf("%v", topic)
				}
				result[key] = topicStrings
			}
		}
		return result
	}
	return make(map[string][]string)
}

// GetOutputTopic returns the output topics configuration
func (sm *SchemaManager) GetOutputTopic() map[string][]string {
	if data, ok := sm.schemaData[string(DestinationTopicTypeOutputTopic)].(map[string]interface{}); ok {
		result := make(map[string][]string)
		for key, value := range data {
			if topics, ok := value.([]interface{}); ok {
				topicStrings := make([]string, len(topics))
				for i, topic := range topics {
					topicStrings[i] = fmt.Sprintf("%v", topic)
				}
				result[key] = topicStrings
			}
		}
		return result
	}
	return make(map[string][]string)
}

// FindTopicByUnitNode finds a topic by unit node UUID or full name
func (sm *SchemaManager) FindTopicByUnitNode(searchValue string, searchType SearchTopicType, searchScope SearchScope) (string, error) {
	sections := sm.getSectionsByScope(searchScope)

	for _, section := range sections {
		var result string
		var err error

		switch searchType {
		case SearchTopicTypeUnitNodeUUID:
			result, err = sm.searchUUIDInTopicSection(section, searchValue)
		case SearchTopicTypeFullName:
			result, err = sm.searchTopicNameInSection(section, searchValue)
		default:
			continue
		}

		if err != nil {
			return "", err
		}
		if result != "" {
			return result, nil
		}
	}

	return "", fmt.Errorf("topic not found")
}

// getSectionsByScope returns the sections to search based on scope
func (sm *SchemaManager) getSectionsByScope(searchScope SearchScope) []string {
	switch searchScope {
	case SearchScopeAll:
		return []string{string(DestinationTopicTypeInputTopic), string(DestinationTopicTypeOutputTopic)}
	case SearchScopeInput:
		return []string{string(DestinationTopicTypeInputTopic)}
	case SearchScopeOutput:
		return []string{string(DestinationTopicTypeOutputTopic)}
	default:
		return []string{}
	}
}

// searchUUIDInTopicSection searches for a UUID in a specific topic section
func (sm *SchemaManager) searchUUIDInTopicSection(section string, uuid string) (string, error) {
	topicSection, ok := sm.schemaData[section].(map[string]interface{})
	if !ok {
		return "", nil
	}

	for topicName, topicList := range topicSection {
		if topics, ok := topicList.([]interface{}); ok {
			for _, topicURL := range topics {
				if topicURLStr, ok := topicURL.(string); ok {
					if sm.extractUUIDFromTopic(topicURLStr) == uuid {
						return topicName, nil
					}
				}
			}
		}
	}

	return "", nil
}

// extractUUIDFromTopic extracts UUID from topic URL
func (sm *SchemaManager) extractUUIDFromTopic(topicURL string) string {
	parts := strings.Split(topicURL, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// searchTopicNameInSection searches for a topic name in a specific section
func (sm *SchemaManager) searchTopicNameInSection(section string, topicName string) (string, error) {
	topicSection, ok := sm.schemaData[section].(map[string]interface{})
	if !ok {
		return "", nil
	}

	for topicKey, topicList := range topicSection {
		if topics, ok := topicList.([]interface{}); ok {
			for _, topicURL := range topics {
				if topicURLStr, ok := topicURL.(string); ok {
					if topicURLStr == topicName {
						return topicKey, nil
					}
				}
			}
		}
	}

	return "", nil
}

package pepeunit_test

import (
	"context"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

type mockMQTTClient struct {
	connected  bool
	subscribed []string
	published  [][2]string
	handler    pepeunit.MQTTInputHandler
}

func (m *mockMQTTClient) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

func (m *mockMQTTClient) Disconnect(ctx context.Context) error {
	m.connected = false
	return nil
}

func (m *mockMQTTClient) SubscribeTopics(topics []string) error {
	m.subscribed = append(m.subscribed, topics...)
	return nil
}

func (m *mockMQTTClient) Publish(topic, message string) error {
	m.published = append(m.published, [2]string{topic, message})
	return nil
}

func (m *mockMQTTClient) SetInputHandler(handler pepeunit.MQTTInputHandler) {
	m.handler = handler
}

type mockRESTClient struct {
	updates     map[string]string
	envFiles    map[string]string
	schemaFiles map[string]string
	stateByUnit map[string]map[string]interface{}
}

func newMockRESTClient() *mockRESTClient {
	return &mockRESTClient{
		updates:     map[string]string{},
		envFiles:    map[string]string{},
		schemaFiles: map[string]string{},
		stateByUnit: map[string]map[string]interface{}{},
	}
}

func (m *mockRESTClient) DownloadUpdate(ctx context.Context, unitUUID, filePath string) error {
	return nil
}

func (m *mockRESTClient) DownloadEnv(ctx context.Context, unitUUID, filePath string) error {
	data := map[string]interface{}{"KEY": "VALUE"}
	fm := pepeunit.NewFileManager()
	return fm.WriteJSON(filePath, data)
}

func (m *mockRESTClient) DownloadSchema(ctx context.Context, unitUUID, filePath string) error {
	data := map[string]interface{}{
		"output_base_topic": map[string]interface{}{
			"log/pepeunit":   []interface{}{"unit/abc/log"},
			"state/pepeunit": []interface{}{"unit/abc/state"},
		},
	}
	fm := pepeunit.NewFileManager()
	return fm.WriteJSON(filePath, data)
}

func (m *mockRESTClient) DownloadFileFromURL(ctx context.Context, url, filePath string) error {
	return nil
}

func (m *mockRESTClient) SetStateStorage(ctx context.Context, unitUUID string, state map[string]interface{}) error {
	m.stateByUnit[unitUUID] = state
	return nil
}

func (m *mockRESTClient) GetStateStorage(ctx context.Context, unitUUID string) (map[string]interface{}, error) {
	return m.stateByUnit[unitUUID], nil
}

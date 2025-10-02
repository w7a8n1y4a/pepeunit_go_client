package pepeunit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// PepeunitRESTClient implements RESTClient interface
type PepeunitRESTClient struct {
	*AbstractRESTClient
	httpClient *http.Client
}

// NewPepeunitRESTClient creates a new REST client
func NewPepeunitRESTClient(settings *Settings) *PepeunitRESTClient {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &PepeunitRESTClient{
		AbstractRESTClient: NewAbstractRESTClient(settings),
		httpClient:         httpClient,
	}
}

// DownloadUpdate downloads firmware update archive
func (c *PepeunitRESTClient) DownloadUpdate(ctx context.Context, unitUUID, filePath string) error {
	url := c.GetBaseURL() + "/units/" + unitUUID + "/update"
	return c.downloadFile(ctx, url, filePath)
}

// DownloadEnv downloads environment configuration
func (c *PepeunitRESTClient) DownloadEnv(ctx context.Context, unitUUID, filePath string) error {
	url := c.GetBaseURL() + "/units/env/" + unitUUID
	return c.downloadJSONFile(ctx, url, filePath)
}

// DownloadSchema downloads topic schema configuration
func (c *PepeunitRESTClient) DownloadSchema(ctx context.Context, unitUUID, filePath string) error {
	url := c.GetBaseURL() + "/units/get_current_schema/" + unitUUID
	return c.downloadJSONFile(ctx, url, filePath)
}

// SetStateStorage stores state data in PepeUnit storage
func (c *PepeunitRESTClient) SetStateStorage(ctx context.Context, unitUUID string, state map[string]interface{}) error {
	url := c.GetBaseURL() + "/units/" + unitUUID + "/state"

	jsonData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state data: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, io.NopCloser(strings.NewReader(string(jsonData))))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	headers := c.GetAuthHeaders()
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetStateStorage retrieves state data from PepeUnit storage
func (c *PepeunitRESTClient) GetStateStorage(ctx context.Context, unitUUID string) (map[string]interface{}, error) {
	url := c.GetBaseURL() + "/units/" + unitUUID + "/state"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	headers := c.GetAuthHeaders()
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return result, nil
}

// downloadFile downloads a file from the given URL to the specified file path
func (c *PepeunitRESTClient) downloadFile(ctx context.Context, url, filePath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	headers := c.GetAuthHeaders()
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Create the destination file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", filePath, err)
	}
	defer file.Close()

	// Copy the response body to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %v", filePath, err)
	}

	return nil
}

// downloadJSONFile downloads JSON data from the given URL and writes it to the specified file path
func (c *PepeunitRESTClient) downloadJSONFile(ctx context.Context, url, filePath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	headers := c.GetAuthHeaders()
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse JSON; also handle the case when API returns a JSON-encoded string
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return fmt.Errorf("failed to parse JSON response: %v", err)
	}
	if str, ok := jsonData.(string); ok {
		var nested interface{}
		if err := json.Unmarshal([]byte(str), &nested); err == nil {
			jsonData = nested
		}
	}

	// Create the destination file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", filePath, err)
	}
	defer file.Close()

	// Write the JSON data to the file with proper formatting
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	err = encoder.Encode(jsonData)
	if err != nil {
		return fmt.Errorf("failed to write JSON file %s: %v", filePath, err)
	}

	return nil
}

// SetHTTPClient sets a custom HTTP client
func (c *PepeunitRESTClient) SetHTTPClient(httpClient *http.Client) {
	c.httpClient = httpClient
}

// GetHTTPClient returns the current HTTP client
func (c *PepeunitRESTClient) GetHTTPClient() *http.Client {
	return c.httpClient
}

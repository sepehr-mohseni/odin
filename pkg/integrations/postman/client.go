package postman

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	baseURL        = "https://api.getpostman.com"
	defaultTimeout = 30 * time.Second
)

// Client represents a Postman API client
type Client struct {
	apiKey     string
	httpClient *http.Client
	logger     *logrus.Logger
	baseURL    string
}

// NewClient creates a new Postman API client
func NewClient(apiKey string, logger *logrus.Logger) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		logger:  logger,
		baseURL: baseURL,
	}
}

// SetTimeout sets the HTTP client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	c.logger.WithFields(logrus.Fields{
		"method": method,
		"url":    url,
	}).Debug("Making Postman API request")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Check for API errors
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		var errorResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("API error (status %d): failed to decode error response", resp.StatusCode)
		}
		return nil, fmt.Errorf("API error: %s - %s", errorResp.Error.Name, errorResp.Error.Message)
	}

	return resp, nil
}

// Collections

// ListCollections retrieves all collections in the workspace
func (c *Client) ListCollections(ctx context.Context) ([]CollectionSummary, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/collections", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result CollectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.WithField("count", len(result.Collections)).Info("Listed Postman collections")
	return result.Collections, nil
}

// GetCollection retrieves a specific collection by UID
func (c *Client) GetCollection(ctx context.Context, uid string) (*PostmanCollection, error) {
	path := fmt.Sprintf("/collections/%s", uid)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result CollectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"uid":  uid,
		"name": result.Collection.Info.Name,
	}).Info("Retrieved Postman collection")

	return &result.Collection, nil
}

// CreateCollection creates a new collection
func (c *Client) CreateCollection(ctx context.Context, collection *PostmanCollection) (*PostmanCollection, error) {
	body := map[string]interface{}{
		"collection": collection,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/collections", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result CollectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.WithField("name", result.Collection.Info.Name).Info("Created Postman collection")
	return &result.Collection, nil
}

// UpdateCollection updates an existing collection
func (c *Client) UpdateCollection(ctx context.Context, uid string, collection *PostmanCollection) (*PostmanCollection, error) {
	path := fmt.Sprintf("/collections/%s", uid)
	body := map[string]interface{}{
		"collection": collection,
	}

	resp, err := c.doRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result CollectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"uid":  uid,
		"name": result.Collection.Info.Name,
	}).Info("Updated Postman collection")

	return &result.Collection, nil
}

// DeleteCollection deletes a collection
func (c *Client) DeleteCollection(ctx context.Context, uid string) error {
	path := fmt.Sprintf("/collections/%s", uid)
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.logger.WithField("uid", uid).Info("Deleted Postman collection")
	return nil
}

// Environments

// ListEnvironments retrieves all environments
func (c *Client) ListEnvironments(ctx context.Context) ([]Environment, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/environments", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result EnvironmentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.WithField("count", len(result.Environments)).Info("Listed Postman environments")
	return result.Environments, nil
}

// GetEnvironment retrieves a specific environment
func (c *Client) GetEnvironment(ctx context.Context, uid string) (*Environment, error) {
	path := fmt.Sprintf("/environments/%s", uid)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result EnvironmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"uid":  uid,
		"name": result.Environment.Name,
	}).Info("Retrieved Postman environment")

	return &result.Environment, nil
}

// CreateEnvironment creates a new environment
func (c *Client) CreateEnvironment(ctx context.Context, env *Environment) (*Environment, error) {
	body := map[string]interface{}{
		"environment": env,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/environments", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result EnvironmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.WithField("name", result.Environment.Name).Info("Created Postman environment")
	return &result.Environment, nil
}

// UpdateEnvironment updates an existing environment
func (c *Client) UpdateEnvironment(ctx context.Context, uid string, env *Environment) (*Environment, error) {
	path := fmt.Sprintf("/environments/%s", uid)
	body := map[string]interface{}{
		"environment": env,
	}

	resp, err := c.doRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result EnvironmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"uid":  uid,
		"name": result.Environment.Name,
	}).Info("Updated Postman environment")

	return &result.Environment, nil
}

// DeleteEnvironment deletes an environment
func (c *Client) DeleteEnvironment(ctx context.Context, uid string) error {
	path := fmt.Sprintf("/environments/%s", uid)
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.logger.WithField("uid", uid).Info("Deleted Postman environment")
	return nil
}

// Workspaces

// ListWorkspaces retrieves all workspaces
func (c *Client) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/workspaces", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result WorkspacesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.WithField("count", len(result.Workspaces)).Info("Listed Postman workspaces")
	return result.Workspaces, nil
}

// GetWorkspace retrieves a specific workspace
func (c *Client) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	path := fmt.Sprintf("/workspaces/%s", id)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result WorkspaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"id":   id,
		"name": result.Workspace.Name,
	}).Info("Retrieved Postman workspace")

	return &result.Workspace, nil
}

// Monitors

// ListMonitors retrieves all monitors
func (c *Client) ListMonitors(ctx context.Context) ([]Monitor, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/monitors", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result MonitorsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.WithField("count", len(result.Monitors)).Info("Listed Postman monitors")
	return result.Monitors, nil
}

// Mock Servers

// ListMockServers retrieves all mock servers
func (c *Client) ListMockServers(ctx context.Context) ([]MockServer, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/mocks", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result MockServersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.WithField("count", len(result.Mocks)).Info("Listed Postman mock servers")
	return result.Mocks, nil
}

// Utility methods

// TestConnection tests the API connection
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.ListWorkspaces(ctx)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	c.logger.Info("Postman API connection test successful")
	return nil
}

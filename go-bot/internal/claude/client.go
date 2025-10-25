package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// QueryClient is an interface for both HTTP and SDK clients
type QueryClient interface {
	Query(ctx context.Context, req QueryRequest) (<-chan StreamResponse, <-chan error)
	Health(ctx context.Context) error
}

// Client represents a Claude bridge HTTP client
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// QueryRequest represents a request to Claude
type QueryRequest struct {
	Prompt          string   `json:"prompt"`
	SessionID       string   `json:"sessionId,omitempty"`
	Model           string   `json:"model,omitempty"`
	Workspace       string   `json:"workspace,omitempty"`
	PermissionMode  string   `json:"permissionMode,omitempty"`
	AllowedTools    []string `json:"allowedTools,omitempty"`
}

// StreamResponse represents a response from Claude
type StreamResponse struct {
	Type  string          `json:"type"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
	Code  string          `json:"code,omitempty"`
}

// NewClient creates a new Claude bridge client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{},
	}
}

// Query sends a query to Claude and streams responses
func (c *Client) Query(ctx context.Context, req QueryRequest) (<-chan StreamResponse, <-chan error) {
	responseChan := make(chan StreamResponse, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		// Marshal request
		body, err := json.Marshal(req)
		if err != nil {
			errorChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		// Create HTTP request
		httpReq, err := http.NewRequestWithContext(
			ctx,
			"POST",
			c.baseURL+"/api/query",
			bytes.NewReader(body),
		)
		if err != nil {
			errorChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")

		// Execute request
		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			errorChan <- fmt.Errorf("failed to execute request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errorChan <- fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
			return
		}

		// Parse SSE stream
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			// SSE format: "data: {...}"
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "" {
				continue
			}

			var streamResp StreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				log.Printf("Failed to parse SSE message: %v", err)
				continue
			}

			select {
			case responseChan <- streamResp:
			case <-ctx.Done():
				return
			}

			// Stop if we got "done" or "error"
			if streamResp.Type == "done" || streamResp.Type == "error" {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			errorChan <- fmt.Errorf("error reading stream: %w", err)
		}
	}()

	return responseChan, errorChan
}

// Health checks if the Claude bridge is healthy
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

package claude

import (
	"context"
	"encoding/json"
)

// QueryClient is an interface for Claude clients
type QueryClient interface {
	Query(ctx context.Context, req QueryRequest) (<-chan StreamResponse, <-chan error)
	Health(ctx context.Context) error
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

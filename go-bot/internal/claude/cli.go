package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
)

// CLIClient wraps the Claude CLI for executing queries
type CLIClient struct {
	model          string
	permissionMode string
}

// NewCLIClient creates a new CLI client
func NewCLIClient(model, permissionMode string) *CLIClient {
	return &CLIClient{
		model:          model,
		permissionMode: permissionMode,
	}
}

// Query executes a Claude query using the CLI directly
func (c *CLIClient) Query(ctx context.Context, req QueryRequest) (<-chan StreamResponse, <-chan error) {
	responseChan := make(chan StreamResponse, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		// Build CLI arguments
		args := []string{
			"--print",
			"--output-format", "stream-json",
			"--verbose", // Required for stream-json format
			"--permission-mode", c.permissionMode,
			// Allow common development tools
			"--allowed-tools", "Bash", "Read", "Write", "Edit", "Glob", "Grep",
		}

		// Add model if specified
		if req.Model != "" {
			args = append(args, "--model", req.Model)
		} else if c.model != "" {
			args = append(args, "--model", c.model)
		}

		// Add session ID if provided (use --resume to continue existing session)
		if req.SessionID != "" {
			args = append(args, "--resume", req.SessionID)
		}

		// Add workspace/cwd if provided
		if req.Workspace != "" {
			// Claude CLI uses the current working directory
			// We'll set it via cmd.Dir instead of a flag
		}

		// Add the prompt as the last argument
		args = append(args, req.Prompt)

		log.Printf("[Claude CLI] Executing: claude %v", args)

		// Execute Claude CLI
		cmd := exec.CommandContext(ctx, "claude", args...)
		if req.Workspace != "" {
			cmd.Dir = req.Workspace
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errorChan <- fmt.Errorf("failed to create stdout pipe: %w", err)
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			errorChan <- fmt.Errorf("failed to create stderr pipe: %w", err)
			return
		}

		if err := cmd.Start(); err != nil {
			errorChan <- fmt.Errorf("failed to start claude process: %w", err)
			return
		}

		// Read stderr in background
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				log.Printf("[Claude CLI stderr]: %s", scanner.Text())
			}
		}()

		// Parse stdout for JSON messages
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			// Parse the stream-json format
			var cliMessage map[string]interface{}
			if err := json.Unmarshal([]byte(line), &cliMessage); err != nil {
				log.Printf("Failed to parse CLI response: %v (line: %s)", err, line)
				continue
			}

			// Convert CLI format to our StreamResponse format
			response := c.convertCLIMessage(cliMessage)
			if response != nil {
				select {
				case responseChan <- *response:
				case <-ctx.Done():
					cmd.Process.Kill()
					return
				}
			}
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			errorChan <- fmt.Errorf("error reading CLI output: %w", err)
		}

		cmd.Wait()

		// Send done signal
		select {
		case responseChan <- StreamResponse{Type: "done"}:
		case <-ctx.Done():
			return
		}
	}()

	return responseChan, errorChan
}

// convertCLIMessage converts Claude CLI stream-json format to our StreamResponse
func (c *CLIClient) convertCLIMessage(cliMsg map[string]interface{}) *StreamResponse {
	// Claude CLI stream-json format is the same as SDK format
	// Just wrap it in our response type
	data, err := json.Marshal(cliMsg)
	if err != nil {
		log.Printf("Failed to marshal CLI message: %v", err)
		return nil
	}

	return &StreamResponse{
		Type: "claude_message",
		Data: json.RawMessage(data),
	}
}

// Health checks if Claude CLI is available
func (c *CLIClient) Health(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "claude", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("claude CLI not available: %w (output: %s)", err, string(output))
	}
	return nil
}

package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/drew/omnik-bot/internal/claude"
	"github.com/drew/omnik-bot/internal/session"
)

// Bot represents the Telegram bot
type Bot struct {
	api            *tgbotapi.BotAPI
	claudeClient   claude.QueryClient // Interface for both HTTP and SDK clients
	sessionManager *session.Manager
	authorizedUID  int64
	workingDir     string // Current working directory for debugging
}

// Config holds bot configuration
type Config struct {
	TelegramToken   string
	AuthorizedUID   int64
	ClaudeBridgeURL string // For HTTP mode (legacy)
	UseSDK          bool   // Use SDK client instead of HTTP
	ClaudeModel     string // Model to use (sonnet, opus, etc)
}

// New creates a new bot instance
func New(cfg Config) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	log.Printf("Authorized on account %s", api.Self.UserName)

	// Create appropriate Claude client
	var claudeClient claude.QueryClient
	if cfg.UseSDK {
		log.Printf("Using Claude CLI client (model: %s)", cfg.ClaudeModel)
		claudeClient = claude.NewCLIClient(cfg.ClaudeModel, "bypassPermissions")
	} else {
		log.Printf("Using Claude HTTP client (bridge: %s)", cfg.ClaudeBridgeURL)
		claudeClient = claude.NewClient(cfg.ClaudeBridgeURL)
	}

	// Check Claude health
	ctx := context.Background()
	if err := claudeClient.Health(ctx); err != nil {
		log.Printf("WARNING: Claude health check failed: %v", err)
	} else {
		log.Printf("✓ Claude is healthy")
	}

	// Initialize session manager
	sessionManager, err := session.NewManager("/workspace/.omnik-sessions.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	// Create default session if none exists
	if len(sessionManager.List()) == 0 {
		_, err := sessionManager.Create("default", "Default session", "/workspace")
		if err != nil {
			return nil, fmt.Errorf("failed to create default session: %w", err)
		}
		log.Printf("Created default session")
	}

	// Get current session's working directory
	currentSession := sessionManager.Current()
	workingDir := "/workspace"
	if currentSession != nil {
		workingDir = currentSession.WorkingDir
	}

	return &Bot{
		api:            api,
		claudeClient:   claudeClient,
		sessionManager: sessionManager,
		authorizedUID:  cfg.AuthorizedUID,
		workingDir:     workingDir,
	}, nil
}

// Start starts the bot
func (b *Bot) Start(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	log.Println("🤖 Bot started, waiting for messages...")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update := <-updates:
			if update.Message == nil {
				continue
			}

			b.handleMessage(ctx, update.Message)
		}
	}
}

// handleMessage processes incoming messages
func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	// Check authorization
	if msg.From.ID != b.authorizedUID {
		log.Printf("Unauthorized access attempt from user %d", msg.From.ID)
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Unauthorized")
		b.api.Send(reply)
		return
	}

	// Handle commands
	if msg.IsCommand() {
		b.handleCommand(ctx, msg)
		return
	}

	// Forward text message to Claude
	if msg.Text != "" {
		b.forwardToClaude(ctx, msg)
		return
	}
}

// handleCommand handles bot commands
func (b *Bot) handleCommand(ctx context.Context, msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		reply := tgbotapi.NewMessage(msg.Chat.ID,
			"Welcome to omnik - Claude Code on Telegram\n\n"+
				"Send me any message and I'll forward it to Claude!\n\n"+
				"File Navigation:\n"+
				"/pwd - Show current working directory\n"+
				"/ls - List files (ls -lah)\n"+
				"/cd <path> - Change directory\n"+
				"/cat <file> - Show file contents\n"+
				"/exec <cmd> - Execute bash command\n\n"+
				"Session Management:\n"+
				"/sessions - List all sessions\n"+
				"/newsession <name> [description] - Create new session\n"+
				"/switch <name> - Switch to session\n"+
				"/delsession <name> - Delete session\n"+
				"/status - Show current session status")
		b.api.Send(reply)

	case "status":
		currentSession := b.sessionManager.Current()
		var status string
		if currentSession == nil {
			status = "No active session\n\nUse /newsession to create one"
		} else {
			status = fmt.Sprintf(
				"Current Session\n\n"+
					"Name: %s\n"+
					"Description: %s\n"+
					"Working Dir: %s\n"+
					"Created: %s\n"+
					"Last Used: %s\n"+
					"Session ID: %s",
				currentSession.Name,
				currentSession.Description,
				currentSession.WorkingDir,
				currentSession.CreatedAt.Format("2006-01-02 15:04"),
				currentSession.LastUsedAt.Format("2006-01-02 15:04"),
				currentSession.ID,
			)
		}
		reply := tgbotapi.NewMessage(msg.Chat.ID, status)
		b.api.Send(reply)

	case "sessions":
		sessions := b.sessionManager.List()
		if len(sessions) == 0 {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "No sessions found\n\nUse /newsession to create one"))
			return
		}

		var text strings.Builder
		text.WriteString(fmt.Sprintf("Sessions (%d)\n\n", len(sessions)))

		currentSession := b.sessionManager.Current()
		for _, s := range sessions {
			marker := "  "
			if currentSession != nil && s.Name == currentSession.Name {
				marker = "→ "
			}
			text.WriteString(fmt.Sprintf("%s%s\n", marker, s.Name))
			if s.Description != "" {
				text.WriteString(fmt.Sprintf("   %s\n", s.Description))
			}
			text.WriteString(fmt.Sprintf("   Dir: %s\n", s.WorkingDir))
			text.WriteString(fmt.Sprintf("   Last used: %s\n\n", s.LastUsedAt.Format("2006-01-02 15:04")))
		}

		b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, text.String()))

	case "newsession":
		args := strings.TrimSpace(msg.CommandArguments())
		if args == "" {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /newsession <name> [description]"))
			return
		}

		// Parse name and description
		parts := strings.SplitN(args, " ", 2)
		name := parts[0]
		description := ""
		if len(parts) > 1 {
			description = parts[1]
		}

		// Create new session
		newSession, err := b.sessionManager.Create(name, description, "/workspace")
		if err != nil {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Error: %v", err)))
			return
		}

		// Update bot's working directory
		b.workingDir = newSession.WorkingDir

		b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Created and switched to session: %s", name)))

	case "switch":
		args := strings.TrimSpace(msg.CommandArguments())
		if args == "" {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /switch <name>"))
			return
		}

		// Switch session
		switchedSession, err := b.sessionManager.Switch(args)
		if err != nil {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Error: %v", err)))
			return
		}

		// Update bot's working directory
		b.workingDir = switchedSession.WorkingDir

		b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf(
			"Switched to session: %s\nWorking directory: %s",
			switchedSession.Name,
			switchedSession.WorkingDir,
		)))

	case "delsession":
		args := strings.TrimSpace(msg.CommandArguments())
		if args == "" {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /delsession <name>"))
			return
		}

		// Delete session
		if err := b.sessionManager.Delete(args); err != nil {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Error: %v", err)))
			return
		}

		b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Deleted session: %s", args)))

	case "pwd":
		b.execDirectCommand(msg, "pwd")

	case "ls":
		b.execDirectCommand(msg, "ls", "-lah", b.workingDir)

	case "cd":
		args := strings.TrimSpace(msg.CommandArguments())
		if args == "" {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /cd <path>"))
			return
		}

		// Resolve to absolute path
		var newDir string
		if strings.HasPrefix(args, "/") {
			// Already absolute
			newDir = args
		} else {
			// Relative to current working directory
			newDir = b.workingDir + "/" + args
		}

		// Clean the path (resolve .., ., etc.)
		newDir = cleanPath(newDir)

		// Verify directory exists
		if _, err := os.Stat(newDir); os.IsNotExist(err) {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Directory does not exist: %s", newDir)))
			return
		}

		b.workingDir = newDir

		// Save working directory to session
		if err := b.sessionManager.UpdateWorkingDir(newDir); err != nil {
			log.Printf("Warning: failed to save working directory: %v", err)
		}

		b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Working directory changed to: %s", b.workingDir)))

	case "cat":
		args := strings.TrimSpace(msg.CommandArguments())
		if args == "" {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /cat <filename>"))
			return
		}

		// Resolve to absolute path if relative
		filePath := args
		if !strings.HasPrefix(args, "/") {
			filePath = b.workingDir + "/" + args
		}
		b.execDirectCommand(msg, "cat", filePath)

	case "exec":
		args := strings.TrimSpace(msg.CommandArguments())
		if args == "" {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /exec <command>"))
			return
		}
		b.execDirectCommand(msg, "bash", "-c", fmt.Sprintf("cd %s && %s", b.workingDir, args))

	default:
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Unknown command. Use /start for help.")
		b.api.Send(reply)
	}
}

// execDirectCommand executes a command directly using os/exec
func (b *Bot) execDirectCommand(msg *tgbotapi.Message, command string, args ...string) {
	log.Printf("Executing command directly: %s %v", command, args)

	// Send thinking message
	thinkingMsg := tgbotapi.NewMessage(msg.Chat.ID, "Executing...")
	sentMsg, err := b.api.Send(thinkingMsg)
	if err != nil {
		log.Printf("Failed to send thinking message: %v", err)
		return
	}

	// Execute command
	cmd := exec.Command(command, args...)
	cmd.Dir = b.workingDir
	output, err := cmd.CombinedOutput()

	// Prepare response text
	var text string
	if err != nil {
		text = fmt.Sprintf("Error: %v\n\nOutput:\n%s", err, string(output))
	} else {
		text = string(output)
		if text == "" {
			text = "✓ Command executed successfully (no output)"
		}
	}

	// Truncate if too long
	if len(text) > 4000 {
		text = text[:4000] + "\n\n... (truncated)"
	}

	// Send result
	editMsg := tgbotapi.NewEditMessageText(msg.Chat.ID, sentMsg.MessageID, text)
	b.api.Send(editMsg)
}

// forwardToClaude forwards a message to Claude and streams the response
func (b *Bot) forwardToClaude(ctx context.Context, msg *tgbotapi.Message) {
	log.Printf("→ Forwarding to Claude: %s", msg.Text)

	// Get current session
	currentSession := b.sessionManager.Current()
	if currentSession == nil {
		b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "No active session. Use /newsession to create one."))
		return
	}

	// Send "thinking" message
	thinkingMsg := tgbotapi.NewMessage(msg.Chat.ID, "🤔 Processing...")
	sentMsg, err := b.api.Send(thinkingMsg)
	if err != nil {
		log.Printf("Failed to send thinking message: %v", err)
		return
	}

	// Query Claude with bypassed permissions for autonomous operation
	req := claude.QueryRequest{
		Prompt:         msg.Text,
		SessionID:      currentSession.ID,
		Workspace:      b.workingDir,
		PermissionMode: "bypassPermissions", // Skip all permission prompts
	}

	responseChan, errorChan := b.claudeClient.Query(ctx, req)

	var fullResponse strings.Builder
	var lastEdit int
	messageCount := 0

	for {
		select {
		case err := <-errorChan:
			if err != nil {
				log.Printf("Claude query error: %v", err)
				editMsg := tgbotapi.NewEditMessageText(
					msg.Chat.ID,
					sentMsg.MessageID,
					fmt.Sprintf("❌ Error: %v", err),
				)
				b.api.Send(editMsg)
				return
			}

		case response, ok := <-responseChan:
			if !ok {
				// Channel closed
				return
			}

			messageCount++

			switch response.Type {
			case "claude_message":
				// Parse SDK message
				var sdkMsg map[string]interface{}
				if err := json.Unmarshal(response.Data, &sdkMsg); err != nil {
					log.Printf("Failed to parse SDK message: %v", err)
					continue
				}

				// Extract session ID if this is a system message
				if msgType, ok := sdkMsg["type"].(string); ok && msgType == "system" {
					if sessionID, ok := sdkMsg["session_id"].(string); ok && sessionID != "" {
						// Update session with ID from Claude
						if currentSession.ID == "" {
							currentSession.ID = sessionID
							if err := b.sessionManager.UpdateSessionID(currentSession.Name, sessionID); err != nil {
								log.Printf("Warning: failed to update session ID: %v", err)
							} else {
								log.Printf("Session ID set: %s", sessionID)
							}
						}
					}
				}

				// Extract text content from assistant messages
				if msgType, ok := sdkMsg["type"].(string); ok && msgType == "assistant" {
					if message, ok := sdkMsg["message"].(map[string]interface{}); ok {
						if content, ok := message["content"].([]interface{}); ok {
							for _, item := range content {
								if contentItem, ok := item.(map[string]interface{}); ok {
									if contentType, ok := contentItem["type"].(string); ok && contentType == "text" {
										if text, ok := contentItem["text"].(string); ok {
											fullResponse.WriteString(text)
										}
									}
								}
							}
						}
					}
				}

				// Update message every 2 seconds or every 10 messages
				currentTime := msg.Date
				if messageCount%10 == 0 || currentTime-lastEdit >= 2 {
					if fullResponse.Len() > 0 {
						text := fullResponse.String()
						if len(text) > 4000 {
							text = text[:4000] + "\n\n... (truncated)"
						}

						editMsg := tgbotapi.NewEditMessageText(msg.Chat.ID, sentMsg.MessageID, text)
						b.api.Send(editMsg)
						lastEdit = currentTime
					}
				}

			case "done":
				log.Printf("← Received %d messages from Claude", messageCount)

				// Final update
				text := fullResponse.String()
				if text == "" {
					text = "✅ Done (no output)"
				}
				if len(text) > 4000 {
					text = text[:4000] + "\n\n... (truncated)"
				}

				editMsg := tgbotapi.NewEditMessageText(msg.Chat.ID, sentMsg.MessageID, text)
				b.api.Send(editMsg)
				return

			case "error":
				log.Printf("Claude error: %s", response.Error)
				editMsg := tgbotapi.NewEditMessageText(
					msg.Chat.ID,
					sentMsg.MessageID,
					fmt.Sprintf("❌ Error: %s", response.Error),
				)
				b.api.Send(editMsg)
				return
			}
		}
	}
}

// cleanPath resolves relative path components (.. and .)
func cleanPath(path string) string {
	// Split path into components
	parts := strings.Split(path, "/")
	cleaned := []string{}

	for _, part := range parts {
		if part == "" || part == "." {
			// Skip empty and current directory
			continue
		} else if part == ".." {
			// Go up one directory
			if len(cleaned) > 0 {
				cleaned = cleaned[:len(cleaned)-1]
			}
		} else {
			cleaned = append(cleaned, part)
		}
	}

	// Rebuild absolute path
	if len(cleaned) == 0 {
		return "/"
	}
	return "/" + strings.Join(cleaned, "/")
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() (Config, error) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return Config{}, fmt.Errorf("TELEGRAM_BOT_TOKEN not set")
	}

	uidStr := os.Getenv("AUTHORIZED_USER_ID")
	if uidStr == "" {
		return Config{}, fmt.Errorf("AUTHORIZED_USER_ID not set")
	}

	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil {
		return Config{}, fmt.Errorf("invalid AUTHORIZED_USER_ID: %w", err)
	}

	// Check if using SDK mode
	useSDK := os.Getenv("USE_CLAUDE_SDK") == "true"

	// Model configuration
	model := os.Getenv("CLAUDE_MODEL")
	if model == "" {
		model = "sonnet" // Default to sonnet
	}

	// Bridge URL for HTTP mode
	bridgeURL := os.Getenv("CLAUDE_BRIDGE_URL")
	if bridgeURL == "" {
		bridgeURL = "http://claude-bridge:9000"
	}

	return Config{
		TelegramToken:   token,
		AuthorizedUID:   uid,
		ClaudeBridgeURL: bridgeURL,
		UseSDK:          useSDK,
		ClaudeModel:     model,
	}, nil
}

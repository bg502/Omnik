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
	"sync"
	"time"

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
	stopChannels   map[int64]chan struct{} // Track stop signals for active queries
	stopMutex      sync.Mutex              // Protect stopChannels map
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
		stopChannels:   make(map[int64]chan struct{}),
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
			// Handle callback queries (inline keyboard button clicks)
			if update.CallbackQuery != nil {
				b.handleCallbackQuery(ctx, update.CallbackQuery)
				continue
			}

			// Handle messages
			if update.Message != nil {
				b.handleMessage(ctx, update.Message)
			}
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

	// Handle keyboard button presses (execute commands directly)
	if msg.Text != "" {
		switch msg.Text {
		case "📂 Sessions":
			b.executeCommand(ctx, msg, "sessions", "")
			return
		case "📊 Status":
			b.executeCommand(ctx, msg, "status", "")
			return
		case "📁 pwd":
			b.executeCommand(ctx, msg, "pwd", "")
			return
		case "📋 ls":
			b.executeCommand(ctx, msg, "ls", "")
			return
		case "ℹ️ Help":
			b.executeCommand(ctx, msg, "start", "")
			return
		}

		// Forward text message to Claude
		b.forwardToClaude(ctx, msg)
		return
	}
}

// handleCallbackQuery handles inline keyboard button callbacks
func (b *Bot) handleCallbackQuery(ctx context.Context, query *tgbotapi.CallbackQuery) {
	// Check authorization
	if query.From.ID != b.authorizedUID {
		log.Printf("Unauthorized callback query from user %d", query.From.ID)
		b.api.Request(tgbotapi.NewCallback(query.ID, "❌ Unauthorized"))
		return
	}

	// Parse callback data
	data := query.Data
	log.Printf("Received callback query: %s", data)

	// Handle different callback types
	if strings.HasPrefix(data, "switch:") {
		// Extract session name
		sessionName := strings.TrimPrefix(data, "switch:")

		// Switch to the session
		switchedSession, err := b.sessionManager.Switch(sessionName)
		if err != nil {
			b.api.Request(tgbotapi.NewCallback(query.ID, "❌ Failed to switch session"))
			b.api.Send(tgbotapi.NewMessage(query.Message.Chat.ID,
				fmt.Sprintf("Error: %v", err)))
			return
		}

		// Update working directory
		if switchedSession != nil {
			b.workingDir = switchedSession.WorkingDir
		}

		// Acknowledge callback
		b.api.Request(tgbotapi.NewCallback(query.ID, "✓ Switched to "+sessionName))

		// Send confirmation message
		b.api.Send(tgbotapi.NewMessage(query.Message.Chat.ID,
			fmt.Sprintf("Switched to session: %s\nWorking directory: %s",
				sessionName, b.workingDir)))

	} else if data == "newsession" {
		// Acknowledge callback
		b.api.Request(tgbotapi.NewCallback(query.ID, ""))

		// Send instruction message
		b.api.Send(tgbotapi.NewMessage(query.Message.Chat.ID,
			"To create a new session, use:\n/newsession <name> [description]"))

	} else if data == "stop" {
		// Acknowledge callback
		b.api.Request(tgbotapi.NewCallback(query.ID, "⏹️ Stopping..."))

		// Send stop signal
		b.stopMutex.Lock()
		stopChan, exists := b.stopChannels[query.Message.Chat.ID]
		b.stopMutex.Unlock()

		if exists {
			close(stopChan) // Signal to stop
			log.Printf("Sent stop signal for chat %d", query.Message.Chat.ID)
		} else {
			log.Printf("No active query found for chat %d", query.Message.Chat.ID)
		}

	} else {
		// Unknown callback
		b.api.Request(tgbotapi.NewCallback(query.ID, "❌ Unknown action"))
	}
}

// executeCommand executes a command by name (for keyboard buttons)
func (b *Bot) executeCommand(ctx context.Context, msg *tgbotapi.Message, command string, args string) {
	switch command {
	case "start":
		reply := tgbotapi.NewMessage(msg.Chat.ID,
			"Welcome to Omnik - Claude Code on Telegram 🤖\n\n"+
				"Send me any message and I'll forward it to Claude!\n\n"+
				"📱 Use the keyboard buttons below for quick access to commands.\n\n"+
				"**File Navigation:**\n"+
				"/pwd - Show current working directory\n"+
				"/ls - List files (ls -lah)\n"+
				"/cd <path> - Change directory\n"+
				"/cat <file> - Show file contents\n"+
				"/exec <cmd> - Execute bash command\n\n"+
				"**Session Management:**\n"+
				"/sessions - List all sessions\n"+
				"/newsession <name> [description] - Create new session\n"+
				"/switch <name> - Switch to session\n"+
				"/delsession <name> - Delete session\n"+
				"/status - Show current session status")
		reply.ReplyMarkup = createMainKeyboard()
		reply.ParseMode = "Markdown"
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

		reply := tgbotapi.NewMessage(msg.Chat.ID, text.String())
		reply.ReplyMarkup = b.createSessionsInlineKeyboard(sessions)
		b.api.Send(reply)

	case "newsession":
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

// handleCommand handles bot commands (wrapper for executeCommand)
func (b *Bot) handleCommand(ctx context.Context, msg *tgbotapi.Message) {
	command := msg.Command()
	args := strings.TrimSpace(msg.CommandArguments())
	b.executeCommand(ctx, msg, command, args)
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

// formatToolUsage formats a tool use for display in Telegram
func formatToolUsage(toolName string, toolInput map[string]interface{}) string {
	// Map tools to emojis
	emoji := "🔧"
	switch toolName {
	case "Read":
		emoji = "📖"
	case "Edit":
		emoji = "✏️"
	case "Write":
		emoji = "📝"
	case "Grep", "Glob":
		emoji = "🔍"
	case "Bash":
		emoji = "🔨"
	case "WebFetch", "WebSearch":
		emoji = "🌐"
	case "Task":
		emoji = "🤖"
	}

	// Extract key parameter based on tool type
	var detail string
	switch toolName {
	case "Read", "Edit", "Write":
		if filePath, ok := toolInput["file_path"].(string); ok {
			// Show just the filename, not the full path
			parts := strings.Split(filePath, "/")
			detail = parts[len(parts)-1]
		}
	case "Grep":
		if pattern, ok := toolInput["pattern"].(string); ok {
			detail = pattern
			if len(detail) > 30 {
				detail = detail[:30] + "..."
			}
		}
	case "Glob":
		if pattern, ok := toolInput["pattern"].(string); ok {
			detail = pattern
		}
	case "Bash":
		if command, ok := toolInput["command"].(string); ok {
			detail = command
			if len(detail) > 40 {
				detail = detail[:40] + "..."
			}
		}
	case "WebFetch":
		if url, ok := toolInput["url"].(string); ok {
			detail = url
			if len(detail) > 40 {
				detail = detail[:40] + "..."
			}
		}
	case "Task":
		if description, ok := toolInput["description"].(string); ok {
			detail = description
		}
	}

	if detail != "" {
		return fmt.Sprintf("%s %s: %s", emoji, toolName, detail)
	}
	return fmt.Sprintf("%s %s", emoji, toolName)
}

// createStopButtonMarkup creates the inline keyboard with stop button
func createStopButtonMarkup() *tgbotapi.InlineKeyboardMarkup {
	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏹️ Stop", "stop"),
		),
	)
	return &markup
}

// updateOrSplitMessage updates the current message, splitting into new message if needed
// sentCharCount tracks how many characters have been finalized in previous messages
// Returns the new message ID to edit (same if no split, new if split occurred)
func (b *Bot) updateOrSplitMessage(chatID int64, currentMsgID int, fullText string, sentCharCount *int, partNum *int) int {
	const maxLen = 4000

	// Calculate unsent portion (what hasn't been finalized in previous messages yet)
	if *sentCharCount >= len(fullText) {
		// Everything already sent, nothing to update
		return currentMsgID
	}

	unsentText := fullText[*sentCharCount:]

	if len(unsentText) <= maxLen {
		// Current message can hold all unsent content - just update it
		editMsg := tgbotapi.NewEditMessageText(chatID, currentMsgID, unsentText)
		editMsg.ReplyMarkup = createStopButtonMarkup() // Keep stop button visible
		b.api.Send(editMsg)
		return currentMsgID
	}

	// Unsent content > 4000, need to split
	// Finalize current message with first 4000 chars of unsent portion
	currentPortionText := unsentText[:maxLen]

	// Finalize current message with continuation indicator (remove stop button as this message is done)
	editMsg := tgbotapi.NewEditMessageText(chatID, currentMsgID, currentPortionText+"\n\n... (continued)")
	editMsg.ReplyMarkup = nil // Remove stop button from finalized message
	b.api.Send(editMsg)

	// Update sent count - we've now committed these chars to a finalized message
	*sentCharCount += maxLen

	// Calculate remaining unsent text after this split
	remainingText := fullText[*sentCharCount:]

	// Send new message for remaining content
	*partNum++
	continueMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("(part %d)\n\n%s", *partNum, remainingText))
	sentMsg, err := b.api.Send(continueMsg)
	if err != nil {
		log.Printf("Failed to send continuation message: %v", err)
		return currentMsgID // Fall back to current message
	}

	return sentMsg.MessageID
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

	// Send "thinking" message with stop button
	stopButton := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏹️ Stop", "stop"),
		),
	)
	thinkingMsg := tgbotapi.NewMessage(msg.Chat.ID, "🤔 Processing...")
	thinkingMsg.ReplyMarkup = stopButton
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

	// Create cancellable context for this query
	queryCtx, cancelQuery := context.WithCancel(ctx)
	defer cancelQuery()

	// Create and register stop channel
	stopChan := make(chan struct{})
	b.stopMutex.Lock()
	b.stopChannels[msg.Chat.ID] = stopChan
	b.stopMutex.Unlock()

	defer func() {
		b.stopMutex.Lock()
		delete(b.stopChannels, msg.Chat.ID)
		b.stopMutex.Unlock()
	}()

	responseChan, errorChan := b.claudeClient.Query(queryCtx, req)

	// Track content as chronological events
	type contentEvent struct {
		eventType string // "text" or "tool"
		content   string
	}
	var contentHistory []contentEvent
	var lastEdit time.Time
	messageCount := 0
	currentMessageID := sentMsg.MessageID // Track which message we're editing
	messagePartNum := 1                    // Which part/continuation we're on
	sentCharCount := 0                     // How many chars finalized in previous messages

	for {
		select {
		case <-stopChan:
			log.Printf("Stop requested by user")
			// Cancel the query
			cancelQuery()
			// Update message to show stopped status and remove stop button
			editMsg := tgbotapi.NewEditMessageText(msg.Chat.ID, currentMessageID, "⏹️ Stopped by user")
			editMsg.ReplyMarkup = nil // Remove stop button
			b.api.Send(editMsg)
			return

		case err := <-errorChan:
			if err != nil {
				log.Printf("Claude query error: %v", err)
				editMsg := tgbotapi.NewEditMessageText(
					msg.Chat.ID,
					currentMessageID,
					fmt.Sprintf("❌ Error: %v", err),
				)
				editMsg.ReplyMarkup = nil // Remove stop button
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

				// Extract text and tool_use content from assistant messages
				if msgType, ok := sdkMsg["type"].(string); ok && msgType == "assistant" {
					if message, ok := sdkMsg["message"].(map[string]interface{}); ok {
						if content, ok := message["content"].([]interface{}); ok {
							for _, item := range content {
								if contentItem, ok := item.(map[string]interface{}); ok {
									contentType, _ := contentItem["type"].(string)

									// Extract text content
									if contentType == "text" {
										if text, ok := contentItem["text"].(string); ok {
											// Append to last text event or create new one
											if len(contentHistory) > 0 && contentHistory[len(contentHistory)-1].eventType == "text" {
												// Append to existing text event
												contentHistory[len(contentHistory)-1].content += text
											} else {
												// Create new text event
												contentHistory = append(contentHistory, contentEvent{
													eventType: "text",
													content:   text,
												})
											}
										}
									}

									// Extract tool usage
									if contentType == "tool_use" {
										toolName, _ := contentItem["name"].(string)
										toolInput, _ := contentItem["input"].(map[string]interface{})
										if toolName != "" {
											toolStr := formatToolUsage(toolName, toolInput)
											log.Printf("Tool usage: %s", toolStr)
											// Always create new tool event
											contentHistory = append(contentHistory, contentEvent{
												eventType: "tool",
												content:   toolStr,
											})
										}
									}
								}
							}
						}
					}
				}

				// Update message with rate limiting (updates more frequently for real-time feel)
				now := time.Now()
				shouldUpdate := messageCount%3 == 0 || time.Since(lastEdit) >= 1000*time.Millisecond

				if shouldUpdate && len(contentHistory) > 0 {
					// Build chronological log from all events
					var displayParts []string
					for _, event := range contentHistory {
						displayParts = append(displayParts, event.content)
					}
					displayText := strings.Join(displayParts, "\n\n")

					if displayText != "" {
						// Update message, splitting if necessary
						currentMessageID = b.updateOrSplitMessage(msg.Chat.ID, currentMessageID, displayText, &sentCharCount, &messagePartNum)
						lastEdit = now
					}
				}

			case "done":
				log.Printf("← Received %d messages from Claude", messageCount)

				// Final update - show complete chronological log with all tools and text
				if len(contentHistory) == 0 {
					editMsg := tgbotapi.NewEditMessageText(msg.Chat.ID, currentMessageID, "✅ Done (no output)")
					editMsg.ReplyMarkup = nil // Remove stop button
					b.api.Send(editMsg)
					return
				}

				// Build final display from all events
				var displayParts []string
				for _, event := range contentHistory {
					displayParts = append(displayParts, event.content)
				}
				displayText := strings.Join(displayParts, "\n\n")

				// Update message, splitting if necessary
				currentMessageID = b.updateOrSplitMessage(msg.Chat.ID, currentMessageID, displayText, &sentCharCount, &messagePartNum)

				// Remove stop button from final message
				editMarkup := tgbotapi.EditMessageReplyMarkupConfig{
					BaseEdit: tgbotapi.BaseEdit{
						ChatID:    msg.Chat.ID,
						MessageID: currentMessageID,
					},
				}
				editMarkup.ReplyMarkup = nil
				b.api.Send(editMarkup)
				return

			case "error":
				log.Printf("Claude error: %s", response.Error)
				editMsg := tgbotapi.NewEditMessageText(
					msg.Chat.ID,
					currentMessageID,
					fmt.Sprintf("❌ Error: %s", response.Error),
				)
				editMsg.ReplyMarkup = nil // Remove stop button
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

// createMainKeyboard creates the main reply keyboard with common commands
func createMainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📂 Sessions"),
			tgbotapi.NewKeyboardButton("📊 Status"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📁 pwd"),
			tgbotapi.NewKeyboardButton("📋 ls"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("ℹ️ Help"),
		),
	)
}

// createSessionsInlineKeyboard creates inline keyboard for session list
func (b *Bot) createSessionsInlineKeyboard(sessions []*session.Session) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	currentSession := b.sessionManager.Current()

	// Add a button for each session
	for _, s := range sessions {
		// Skip current session (already active)
		if currentSession != nil && s.Name == currentSession.Name {
			continue
		}

		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"➡️ "+s.Name,
				"switch:"+s.Name,
			),
		)
		rows = append(rows, row)
	}

	// Add "New Session" button
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			"➕ Create New Session",
			"newsession",
		),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() (Config, error) {
	token := os.Getenv("OMNI_TELEGRAM_BOT_TOKEN")
	if token == "" {
		return Config{}, fmt.Errorf("OMNI_TELEGRAM_BOT_TOKEN not set")
	}

	uidStr := os.Getenv("OMNI_AUTHORIZED_USER_ID")
	if uidStr == "" {
		return Config{}, fmt.Errorf("OMNI_AUTHORIZED_USER_ID not set")
	}

	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil {
		return Config{}, fmt.Errorf("invalid OMNI_AUTHORIZED_USER_ID: %w", err)
	}

	// Check if using SDK mode
	useSDK := os.Getenv("OMNI_USE_CLAUDE_SDK") == "true"

	// Model configuration
	model := os.Getenv("OMNI_CLAUDE_MODEL")
	if model == "" {
		model = "sonnet" // Default to sonnet
	}

	// Bridge URL for HTTP mode (legacy, not used in CLI mode)
	bridgeURL := os.Getenv("OMNI_CLAUDE_BRIDGE_URL")
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

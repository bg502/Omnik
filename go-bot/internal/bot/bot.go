package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/drew/omnik-bot/internal/claude"
	"github.com/drew/omnik-bot/internal/session"
)

// ChatContext holds session context for a specific chat
type ChatContext struct {
	ChatID         int64
	CurrentSession string // Current session name for this chat
	WorkingDir     string // Current working directory for this chat
}

// Bot represents the Telegram bot
type Bot struct {
	api            *tgbotapi.BotAPI
	claudeClient   claude.QueryClient
	sessionManager *session.Manager
	authorizedUID  int64
	authChatID     int64                 // Optional: Authorized chat ID (for programmatic access)
	chatContexts   map[int64]*ChatContext // Per-chat session contexts
	contextMutex   sync.RWMutex          // Protect chatContexts map
	stopChannels   map[int64]chan struct{} // Track stop signals for active queries
	stopMutex      sync.Mutex              // Protect stopChannels map
}

// Config holds bot configuration
type Config struct {
	TelegramToken string
	AuthorizedUID int64
	AuthChatID    int64  // Optional: Allow messages from specific chat (for programmatic access)
	UseSDK        bool   // Use SDK client instead of HTTP
	ClaudeModel   string // Model to use (sonnet, opus, etc)
}

// New creates a new bot instance
func New(cfg Config) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	log.Printf("Authorized on account %s", api.Self.UserName)

	// Create Claude CLI client
	log.Printf("Using Claude CLI client (model: %s)", cfg.ClaudeModel)
	claudeClient := claude.NewCLIClient(cfg.ClaudeModel, "bypassPermissions")

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

	return &Bot{
		api:            api,
		claudeClient:   claudeClient,
		sessionManager: sessionManager,
		authorizedUID:  cfg.AuthorizedUID,
		authChatID:     cfg.AuthChatID,
		chatContexts:   make(map[int64]*ChatContext),
		stopChannels:   make(map[int64]chan struct{}),
	}, nil
}

// getChatContext gets or creates a chat context for the given chat ID
func (b *Bot) getChatContext(chatID int64) *ChatContext {
	b.contextMutex.RLock()
	ctx, exists := b.chatContexts[chatID]
	b.contextMutex.RUnlock()

	if exists {
		return ctx
	}

	// Create new context
	b.contextMutex.Lock()
	defer b.contextMutex.Unlock()

	// Double-check after acquiring write lock
	if ctx, exists := b.chatContexts[chatID]; exists {
		return ctx
	}

	// Initialize new chat context
	currentSession := b.sessionManager.Current()
	workingDir := "/workspace"
	currentSessionName := ""

	if currentSession != nil {
		workingDir = currentSession.WorkingDir
		currentSessionName = currentSession.Name
	}

	ctx = &ChatContext{
		ChatID:         chatID,
		CurrentSession: currentSessionName,
		WorkingDir:     workingDir,
	}

	b.chatContexts[chatID] = ctx
	log.Printf("[ChatContext] Created context for chat %d: session=%q workingDir=%q",
		chatID, currentSessionName, workingDir)

	return ctx
}

// updateChatContext updates the chat context with new session/working directory
func (b *Bot) updateChatContext(chatID int64, sessionName string, workingDir string) {
	b.contextMutex.Lock()
	defer b.contextMutex.Unlock()

	ctx, exists := b.chatContexts[chatID]
	if !exists {
		ctx = &ChatContext{ChatID: chatID}
		b.chatContexts[chatID] = ctx
	}

	ctx.CurrentSession = sessionName
	ctx.WorkingDir = workingDir

	log.Printf("[ChatContext] Updated context for chat %d: session=%q workingDir=%q",
		chatID, sessionName, workingDir)
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
	// Check authorization: either authorized user OR authorized chat
	isAuthorizedUser := msg.From.ID == b.authorizedUID
	isAuthorizedChat := b.authChatID != 0 && msg.Chat.ID == b.authChatID

	if !isAuthorizedUser && !isAuthorizedChat {
		log.Printf("Unauthorized access attempt from user %d in chat %d", msg.From.ID, msg.Chat.ID)
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Unauthorized")
		b.api.Send(reply)
		return
	}

	// Log incoming message details
	log.Printf("✅ Message from user %d in chat %d (type: %s, title: %q)",
		msg.From.ID, msg.Chat.ID, msg.Chat.Type, msg.Chat.Title)

	// Handle file uploads (documents, photos, etc.)
	if msg.Document != nil || msg.Photo != nil {
		go b.handleFileUpload(ctx, msg)
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
		case "🔧 MCP":
			b.executeCommand(ctx, msg, "mcp", "")
			return
		case "🔄 Reload":
			b.executeCommand(ctx, msg, "reload", "")
			return
		case "ℹ️ Help":
			b.executeCommand(ctx, msg, "start", "")
			return
		}

		// Check if there's already an active query for this chat
		b.stopMutex.Lock()
		_, queryRunning := b.stopChannels[msg.Chat.ID]
		b.stopMutex.Unlock()

		if queryRunning {
			// Send message indicating query is already in progress
			reply := tgbotapi.NewMessage(msg.Chat.ID, "⏳ Already processing a query. Please wait or use the ⏹️ Stop button to cancel it.")
			b.api.Send(reply)
			return
		}

		// Forward text message to Claude (run in goroutine to not block update loop)
		go b.forwardToClaude(ctx, msg)
		return
	}
}

// handleCallbackQuery handles inline keyboard button callbacks
func (b *Bot) handleCallbackQuery(ctx context.Context, query *tgbotapi.CallbackQuery) {
	// Check authorization: either authorized user OR authorized chat
	isAuthorizedUser := query.From.ID == b.authorizedUID
	isAuthorizedChat := b.authChatID != 0 && query.Message.Chat.ID == b.authChatID

	if !isAuthorizedUser && !isAuthorizedChat {
		log.Printf("Unauthorized callback query from user %d in chat %d", query.From.ID, query.Message.Chat.ID)
		b.api.Request(tgbotapi.NewCallback(query.ID, "❌ Unauthorized"))
		return
	}

	// Parse callback data
	data := query.Data
	log.Printf("Received callback query: %s", data)

	// Log callback query details
	log.Printf("✅ Callback from user %d in chat %d (type: %s, title: %q, data: %s)",
		query.From.ID, query.Message.Chat.ID, query.Message.Chat.Type,
		query.Message.Chat.Title, query.Data)

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

		// Update chat-specific context
		if switchedSession != nil {
			b.updateChatContext(query.Message.Chat.ID, switchedSession.Name, switchedSession.WorkingDir)
		}

		// Acknowledge callback
		b.api.Request(tgbotapi.NewCallback(query.ID, "✓ Switched to "+sessionName))

		// Send confirmation message
		chatCtx := b.getChatContext(query.Message.Chat.ID)
		b.api.Send(tgbotapi.NewMessage(query.Message.Chat.ID,
			fmt.Sprintf("Switched to session: %s\nWorking directory: %s",
				sessionName, chatCtx.WorkingDir)))

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

	} else if data == "reload_confirm" {
		// Acknowledge callback
		b.api.Request(tgbotapi.NewCallback(query.ID, "🔄 Reloading session..."))

		// Delete confirmation message
		deleteMsg := tgbotapi.NewDeleteMessage(query.Message.Chat.ID, query.Message.MessageID)
		b.api.Request(deleteMsg)

		// Perform reload
		b.reloadSession(ctx, query.Message.Chat.ID)

	} else if data == "reload_cancel" {
		// Acknowledge callback
		b.api.Request(tgbotapi.NewCallback(query.ID, "❌ Cancelled"))

		// Delete confirmation message
		deleteMsg := tgbotapi.NewDeleteMessage(query.Message.Chat.ID, query.Message.MessageID)
		b.api.Request(deleteMsg)

		b.api.Send(tgbotapi.NewMessage(query.Message.Chat.ID, "Reload cancelled."))

	} else if strings.HasPrefix(data, "mcp:") {
		// Extract session name
		sessionName := strings.TrimPrefix(data, "mcp:")

		// Get session details to show working directory
		sessions := b.sessionManager.List()
		var targetSession *session.Session
		for _, s := range sessions {
			if s.Name == sessionName {
				targetSession = s
				break
			}
		}

		if targetSession == nil {
			b.api.Request(tgbotapi.NewCallback(query.ID, "❌ Session not found"))
			return
		}

		// Acknowledge callback
		b.api.Request(tgbotapi.NewCallback(query.ID, ""))

		// Show MCP management menu
		menuMsg := tgbotapi.NewMessage(query.Message.Chat.ID,
			fmt.Sprintf("MCP Management for: %s\n"+
				"Working directory: %s\n\n"+
				"Available commands:\n"+
				"• /mcp - List MCP servers\n"+
				"• /mcpadd <transport> <name> <url> - Add MCP server\n\n"+
				"Examples:\n"+
				"• /mcpadd http archon http://archon-mcp:8051/mcp\n"+
				"• /mcpadd stdio myserver /path/to/server",
				sessionName, targetSession.WorkingDir))

		// Add inline keyboard with quick actions
		menuMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					"📋 List MCP Servers",
					"mcp_list:"+sessionName,
				),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					"◀️ Back to Sessions",
					"back_to_sessions",
				),
			),
		)

		b.api.Send(menuMsg)

	} else if strings.HasPrefix(data, "mcp_list:") {
		sessionName := strings.TrimPrefix(data, "mcp_list:")

		// Get session
		sessions := b.sessionManager.List()
		var targetSession *session.Session
		for _, s := range sessions {
			if s.Name == sessionName {
				targetSession = s
				break
			}
		}

		if targetSession == nil {
			b.api.Request(tgbotapi.NewCallback(query.ID, "❌ Session not found"))
			return
		}

		// Acknowledge callback
		b.api.Request(tgbotapi.NewCallback(query.ID, "🔍 Checking MCP servers..."))

		// Execute claude mcp list from session's working directory
		cmd := exec.Command("claude", "mcp", "list")
		cmd.Dir = targetSession.WorkingDir
		output, err := cmd.CombinedOutput()

		var text string
		if err != nil {
			text = fmt.Sprintf("Error listing MCP servers:\n%v\n\nOutput:\n%s", err, string(output))
		} else {
			text = fmt.Sprintf("MCP Servers for: %s\n\n%s", sessionName, string(output))
		}

		b.api.Send(tgbotapi.NewMessage(query.Message.Chat.ID, text))

	} else if data == "back_to_sessions" {
		// Acknowledge callback
		b.api.Request(tgbotapi.NewCallback(query.ID, ""))

		// Re-show sessions list
		sessions := b.sessionManager.List()
		if len(sessions) == 0 {
			b.api.Send(tgbotapi.NewMessage(query.Message.Chat.ID, "No sessions found"))
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

		reply := tgbotapi.NewMessage(query.Message.Chat.ID, text.String())
		reply.ReplyMarkup = b.createSessionsInlineKeyboard(sessions)
		b.api.Send(reply)

	} else {
		// Unknown callback
		b.api.Request(tgbotapi.NewCallback(query.ID, "❌ Unknown action"))
	}
}

// executeCommand executes a command by name (for keyboard buttons)
func (b *Bot) executeCommand(ctx context.Context, msg *tgbotapi.Message, command string, args string) {
	// Get chat-specific context
	chatCtx := b.getChatContext(msg.Chat.ID)

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
				"/sendfile <file> - Send file from workspace to chat\n"+
				"/exec <cmd> - Execute bash command\n\n"+
				"**File Upload:**\n"+
				"Send any file or photo to upload it to your current working directory\n\n"+
				"**Session Management:**\n"+
				"/sessions - List all sessions\n"+
				"/newsession <name> [description] - Create new session\n"+
				"/switch <name> - Switch to session\n"+
				"/delsession <name> - Delete session\n"+
				"/status - Show current session status\n\n"+
				"**Archive Management:**\n"+
				"/archives - List archived sessions\n"+
				"/archive-view <name> - View archive details\n"+
				"/archive-delete <name> - Delete archive\n\n"+
				"**MCP Management:**\n"+
				"/mcp - List MCP servers for current project\n"+
				"/mcpadd <transport> <name> <url> - Add MCP server\n"+
				"/reload - Reload session to apply MCP changes")
		reply.ReplyMarkup = createMainKeyboard()
		reply.ParseMode = "Markdown"
		b.api.Send(reply)

	case "status":
		currentSession := b.sessionManager.Current()
		var status string
		if currentSession == nil {
			status = "No active session\n\nUse /newsession to create one"
		} else {
			// Get session size
			sizeBytes, messageCount, _ := b.sessionManager.GetSessionSize(currentSession.Name)
			sizeMB := float64(sizeBytes) / (1024 * 1024)

			status = fmt.Sprintf(
				"Current Session\n\n"+
					"Name: %s\n"+
					"Description: %s\n"+
					"Working Dir: %s\n"+
					"Created: %s\n"+
					"Last Used: %s\n"+
					"Session ID: %s\n"+
					"Size: %.2f MB (%d messages)",
				currentSession.Name,
				currentSession.Description,
				currentSession.WorkingDir,
				currentSession.CreatedAt.Format("2006-01-02 15:04"),
				currentSession.LastUsedAt.Format("2006-01-02 15:04"),
				currentSession.ID,
				sizeMB,
				messageCount,
			)

			// Add warning if session is getting large
			if sizeMB > 2.0 || messageCount > 1000 {
				status += "\n\n⚠️ Session is getting large. Consider using /reload to start fresh."
			}
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

		// Check for orphaned directories
		orphaned := b.findOrphanedDirectories()
		if len(orphaned) > 0 {
			text.WriteString("⚠️ Orphaned directories (no active session):\n")
			for _, dir := range orphaned {
				text.WriteString(fmt.Sprintf("  • %s\n", dir))
			}
			text.WriteString("\n")
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

		// Sanitize session name for directory creation
		dirName := sanitizeSessionName(name)
		sessionDir := fmt.Sprintf("/workspace/%s", dirName)

		// Create session directory if it doesn't exist
		if err := os.MkdirAll(sessionDir, 0755); err != nil {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID,
				fmt.Sprintf("Error creating directory: %v", err)))
			return
		}

		log.Printf("Created session directory: %s", sessionDir)

		// Create new session with dedicated directory
		newSession, err := b.sessionManager.Create(name, description, sessionDir)
		if err != nil {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Error: %v", err)))
			return
		}

		// Update chat context
		b.updateChatContext(msg.Chat.ID, newSession.Name, newSession.WorkingDir)

		b.api.Send(tgbotapi.NewMessage(msg.Chat.ID,
			fmt.Sprintf("✅ Created session: %s\n📁 Working directory: %s", name, sessionDir)))

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

		// Update chat context
		b.updateChatContext(msg.Chat.ID, switchedSession.Name, switchedSession.WorkingDir)

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

		// Get session info before deleting (to show directory path)
		sessionToDelete, err := b.sessionManager.Get(args)
		if err != nil {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Error: %v", err)))
			return
		}
		deletedDir := sessionToDelete.WorkingDir

		// Delete session
		if err := b.sessionManager.Delete(args); err != nil {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Error: %v", err)))
			return
		}

		// Find orphaned directories
		orphaned := b.findOrphanedDirectories()

		// Build response message
		var response strings.Builder
		response.WriteString(fmt.Sprintf("✅ Deleted session: %s\n\n", args))
		response.WriteString("⚠️ Note!\n")
		response.WriteString(fmt.Sprintf("The directory %s still exists with your files.\n\n", deletedDir))

		if len(orphaned) > 0 {
			response.WriteString("📁 Orphaned directories (no active session):\n")
			for _, dir := range orphaned {
				response.WriteString(fmt.Sprintf("  • %s\n", dir))
			}
			response.WriteString("\nUse /cd to navigate and manually clean up if needed.")
		} else {
			response.WriteString("All directories in /workspace have active sessions.")
		}

		b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, response.String()))

	case "pwd":
		b.execDirectCommand(msg, chatCtx.WorkingDir, "pwd")

	case "ls":
		b.execDirectCommand(msg, chatCtx.WorkingDir, "ls", "-lah", chatCtx.WorkingDir)

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
			newDir = chatCtx.WorkingDir + "/" + args
		}

		// Clean the path (resolve .., ., etc.)
		newDir = cleanPath(newDir)

		// Verify directory exists
		if _, err := os.Stat(newDir); os.IsNotExist(err) {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Directory does not exist: %s", newDir)))
			return
		}

		// Update chat context
		b.updateChatContext(msg.Chat.ID, chatCtx.CurrentSession, newDir)

		// Save working directory to session
		if err := b.sessionManager.UpdateWorkingDir(newDir); err != nil {
			log.Printf("Warning: failed to save working directory: %v", err)
		}

		b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Working directory changed to: %s", newDir)))

	case "cat":
		if args == "" {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /cat <filename>"))
			return
		}

		// Resolve to absolute path if relative
		filePath := args
		if !strings.HasPrefix(args, "/") {
			filePath = chatCtx.WorkingDir + "/" + args
		}
		b.execDirectCommand(msg, chatCtx.WorkingDir, "cat", filePath)

	case "sendfile":
		if args == "" {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /sendfile <filename>"))
			return
		}

		// Resolve file path (support both absolute and relative paths)
		var filePath string
		if filepath.IsAbs(args) {
			filePath = args
		} else {
			filePath = filepath.Join(chatCtx.WorkingDir, args)
		}

		// Check if file exists
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ File not found: %s", args)))
			} else {
				b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Error checking file: %v", err)))
			}
			return
		}

		// Check if it's a directory
		if fileInfo.IsDir() {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Cannot send directory: %s\n\nPlease specify a file.", args)))
			return
		}

		// Check file size (Telegram has limits - 50MB for bots)
		fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)
		if fileSizeMB > 50 {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ File too large: %.2f MB\n\nTelegram limit for bots is 50 MB.", fileSizeMB)))
			return
		}

		// Send "preparing file" message
		sentMsg, _ := b.api.Send(tgbotapi.NewMessage(msg.Chat.ID,
			fmt.Sprintf("📤 Preparing to send: %s (%.2f MB)...", args, fileSizeMB)))

		// Send the file as document
		doc := tgbotapi.NewDocument(msg.Chat.ID, tgbotapi.FilePath(filePath))
		doc.Caption = fmt.Sprintf("📄 %s\n💾 Size: %.2f MB", filepath.Base(filePath), fileSizeMB)

		_, err = b.api.Send(doc)
		if err != nil {
			log.Printf("❌ Failed to send file %s to chat %d: %v", filePath, msg.Chat.ID, err)
			editMsg := tgbotapi.NewEditMessageText(msg.Chat.ID, sentMsg.MessageID,
				fmt.Sprintf("❌ Failed to send file: %v", err))
			b.api.Send(editMsg)
			return
		}

		// Update status message to success
		editMsg := tgbotapi.NewEditMessageText(msg.Chat.ID, sentMsg.MessageID,
			fmt.Sprintf("✅ File sent successfully!\n\n📄 %s\n💾 %.2f MB", args, fileSizeMB))
		b.api.Send(editMsg)

		log.Printf("✓ Sent file %s to chat %d (%.2f MB)", filePath, msg.Chat.ID, fileSizeMB)

	case "exec":
		if args == "" {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /exec <command>"))
			return
		}
		b.execDirectCommand(msg, chatCtx.WorkingDir, "bash", "-c", fmt.Sprintf("cd %s && %s", chatCtx.WorkingDir, args))

	case "mcp":
		// MCP server management: /mcp list
		// Use current working directory for project-specific MCP configuration
		if args == "" {
			b.execDirectCommand(msg, chatCtx.WorkingDir, "claude", "mcp", "list")
			return
		}
		// Parse subcommand
		parts := strings.Fields(args)
		subCmd := parts[0]

		switch subCmd {
		case "list":
			b.execDirectCommand(msg, chatCtx.WorkingDir, "claude", "mcp", "list")
		default:
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "MCP commands:\n/mcp - List MCP servers\n/mcp list - List MCP servers"))
		}

	case "mcpadd":
		// Usage: /mcpadd <transport> <name> <url>
		if args == "" {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID,
				"Usage: /mcpadd <transport> <name> <url>\n\n"+
					"Transport types: http, stdio, sse\n\n"+
					"Examples:\n"+
					"• /mcpadd http archon http://archon-mcp:8051/mcp\n"+
					"• /mcpadd stdio myserver /path/to/server\n\n"+
					fmt.Sprintf("Current directory: %s", chatCtx.WorkingDir)))
			return
		}

		parts := strings.Fields(args)
		if len(parts) < 3 {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID,
				"❌ Error: Need 3 arguments: <transport> <name> <url>"))
			return
		}

		transport := parts[0]
		name := parts[1]
		url := parts[2]

		// Validate transport type
		if transport != "http" && transport != "stdio" && transport != "sse" {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID,
				"❌ Error: transport must be http, stdio, or sse"))
			return
		}

		// Execute claude mcp add from session's working directory
		b.execDirectCommand(msg, chatCtx.WorkingDir, "claude", "mcp", "add",
			"--transport", transport, name, url)

	case "reload":
		// Show confirmation dialog
		currentSession := b.sessionManager.Current()
		if currentSession == nil {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "No active session to reload."))
			return
		}

		confirmMsg := tgbotapi.NewMessage(msg.Chat.ID,
			fmt.Sprintf("⚠️ This will create a new session to reload MCP servers.\n\nCurrent session: %s\n\nAre you sure?", currentSession.Name))
		confirmMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Yes", "reload_confirm"),
				tgbotapi.NewInlineKeyboardButtonData("❌ No", "reload_cancel"),
			),
		)
		b.api.Send(confirmMsg)

	case "archives":
		// List all archived sessions
		archives, err := b.sessionManager.ListArchives()
		if err != nil {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Error: %v", err)))
			return
		}

		if len(archives) == 0 {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "No archived sessions found"))
			return
		}

		var text strings.Builder
		text.WriteString(fmt.Sprintf("📦 Archived Sessions (%d)\n\n", len(archives)))

		for _, a := range archives {
			text.WriteString(fmt.Sprintf("🗂 %s\n", a.OriginalName))
			if a.Description != "" {
				text.WriteString(fmt.Sprintf("   %s\n", a.Description))
			}
			text.WriteString(fmt.Sprintf("   Archived: %s\n", a.ArchivedAt.Format("2006-01-02 15:04")))
			sizeMB := float64(a.FileSizeBytes) / (1024 * 1024)
			text.WriteString(fmt.Sprintf("   Size: %.2f MB (%d messages)\n", sizeMB, a.MessageCount))
			text.WriteString(fmt.Sprintf("   Working Dir: %s\n\n", a.WorkingDir))
		}

		text.WriteString("\nUse /archive-view <name> to see details\n")
		text.WriteString("Use /archive-delete <name> to delete an archive")

		reply := tgbotapi.NewMessage(msg.Chat.ID, text.String())
		b.api.Send(reply)

	case "archive-view":
		if args == "" {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /archive-view <name>"))
			return
		}

		archive, err := b.sessionManager.GetArchive(args)
		if err != nil {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Error: %v", err)))
			return
		}

		sizeMB := float64(archive.FileSizeBytes) / (1024 * 1024)
		status := fmt.Sprintf(
			"📦 Archived Session\n\n"+
				"Name: %s\n"+
				"Description: %s\n"+
				"Original ID: %s\n"+
				"Working Dir: %s\n"+
				"Archived: %s\n"+
				"Size: %.2f MB\n"+
				"Messages: %d\n"+
				"Archive Path: %s",
			archive.OriginalName,
			archive.Description,
			archive.OriginalID,
			archive.WorkingDir,
			archive.ArchivedAt.Format("2006-01-02 15:04:05"),
			sizeMB,
			archive.MessageCount,
			archive.ArchivePath,
		)

		reply := tgbotapi.NewMessage(msg.Chat.ID, status)
		b.api.Send(reply)

	case "archive-delete":
		if args == "" {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /archive-delete <name>"))
			return
		}

		// Get archive details first
		archive, err := b.sessionManager.GetArchive(args)
		if err != nil {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Error: %v", err)))
			return
		}

		// Delete the archive
		if err := b.sessionManager.DeleteArchive(args); err != nil {
			b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Error: %v", err)))
			return
		}

		b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Deleted archive: %s", archive.OriginalName)))

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
// workDir specifies the working directory for command execution
func (b *Bot) execDirectCommand(msg *tgbotapi.Message, workDir string, command string, args ...string) {
	log.Printf("Executing command directly: %s %v (workDir: %s)", command, args, workDir)

	// Send thinking message
	thinkingMsg := tgbotapi.NewMessage(msg.Chat.ID, "Executing...")
	sentMsg, err := b.api.Send(thinkingMsg)
	if err != nil {
		log.Printf("Failed to send thinking message: %v", err)
		return
	}

	// Execute command
	cmd := exec.Command(command, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
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

// reloadSession creates a new session to reload MCP servers
func (b *Bot) reloadSession(ctx context.Context, chatID int64) {
	currentSession := b.sessionManager.Current()
	if currentSession == nil {
		b.api.Send(tgbotapi.NewMessage(chatID, "No active session to reload."))
		return
	}

	// Save session info before deleting
	sessionName := currentSession.Name
	sessionDesc := currentSession.Description
	workingDir := currentSession.WorkingDir

	// Delete current session to clear conversation history
	if err := b.sessionManager.Delete(sessionName); err != nil {
		b.api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Failed to delete session: %v", err)))
		return
	}

	// Create new session with SAME name (reloads MCP servers)
	newSession, err := b.sessionManager.Create(sessionName, sessionDesc, workingDir)
	if err != nil {
		b.api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Failed to create new session: %v", err)))
		return
	}

	// Update chat context
	b.updateChatContext(chatID, newSession.Name, newSession.WorkingDir)

	// Send success message
	msg := tgbotapi.NewMessage(chatID,
		fmt.Sprintf("✅ Session reloaded: %s\n\nConversation cleared. MCP servers should now be available.",
			newSession.Name))
	b.api.Send(msg)

	log.Printf("Session reloaded: %s (cleared and recreated)", sessionName)
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
			if len(detail) > 150 {
				detail = detail[:150] + "..."
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

	// Get chat context
	chatCtx := b.getChatContext(msg.Chat.ID)

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
		Workspace:      chatCtx.WorkingDir,
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

			// Remove stop button from current message (preserve content!)
			editMarkup := tgbotapi.EditMessageReplyMarkupConfig{
				BaseEdit: tgbotapi.BaseEdit{
					ChatID:    msg.Chat.ID,
					MessageID: currentMessageID,
				},
			}
			editMarkup.ReplyMarkup = nil
			b.api.Send(editMarkup)

			// Send separate stop notification
			stopMsg := tgbotapi.NewMessage(msg.Chat.ID, "⏹️ Stopped by user")
			b.api.Send(stopMsg)
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

				// Debug: Log message type
				if msgType, ok := sdkMsg["type"].(string); ok {
					log.Printf("[DEBUG] Received message type: %s", msgType)
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
											log.Printf("[DEBUG] Extracted text content (length: %d): %s...", len(text), text[:min(50, len(text))])
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

					log.Printf("[DEBUG] Updating message. History items: %d, Display length: %d", len(contentHistory), len(displayText))

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

// sanitizeSessionName converts a session name to a safe directory name
func sanitizeSessionName(name string) string {
	// Replace spaces with hyphens
	sanitized := strings.ReplaceAll(name, " ", "-")

	// Remove or replace unsafe characters
	// Keep only: alphanumeric, hyphens, underscores, dots
	var result strings.Builder
	for _, r := range sanitized {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		   (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			result.WriteRune(r)
		}
	}

	sanitized = result.String()

	// Ensure it's not empty and doesn't start with a dot
	if sanitized == "" {
		sanitized = "session"
	}
	if sanitized[0] == '.' {
		sanitized = "session-" + sanitized
	}

	return sanitized
}

// findOrphanedDirectories finds directories in /workspace that don't have corresponding sessions
func (b *Bot) findOrphanedDirectories() []string {
	orphaned := []string{}

	// Get all session working directories
	sessions := b.sessionManager.List()
	sessionDirs := []string{}
	for _, s := range sessions {
		sessionDirs = append(sessionDirs, s.WorkingDir)
	}

	// Read /workspace directory
	entries, err := os.ReadDir("/workspace")
	if err != nil {
		log.Printf("Error reading /workspace: %v", err)
		return orphaned
	}

	// Find directories that don't match any session
	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Skip files
		}

		name := entry.Name()

		// Skip hidden files/directories
		if strings.HasPrefix(name, ".") {
			continue
		}

		dirPath := fmt.Sprintf("/workspace/%s", name)

		// Check if this directory (or any subdirectory) is used by any session
		isUsed := false
		for _, sessionDir := range sessionDirs {
			// Check if session's working dir is equal to or inside this directory
			if sessionDir == dirPath || strings.HasPrefix(sessionDir, dirPath+"/") {
				isUsed = true
				break
			}
		}

		if !isUsed {
			orphaned = append(orphaned, dirPath)
		}
	}

	return orphaned
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
			tgbotapi.NewKeyboardButton("🔧 MCP"),
			tgbotapi.NewKeyboardButton("🔄 Reload"),
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

	// Add a row with buttons for each session
	for _, s := range sessions {
		// Skip current session (already active)
		if currentSession != nil && s.Name == currentSession.Name {
			continue
		}

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"➡️ Switch: "+s.Name,
				"switch:"+s.Name,
			),
		))
	}

	// Add "New Session" button
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			"➕ Create New Session",
			"newsession",
		),
	))

	// Add MCP button for current session
	if currentSession != nil {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"🔧 MCP for current session",
				"mcp:"+currentSession.Name,
			),
		))
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// handleFileUpload handles file uploads from Telegram
func (b *Bot) handleFileUpload(ctx context.Context, msg *tgbotapi.Message) {
	// Get chat context to find current working directory
	chatCtx := b.getChatContext(msg.Chat.ID)

	// Determine which file type was sent
	var fileID string
	var fileName string

	if msg.Document != nil {
		// Handle documents (PDFs, text files, archives, etc.)
		fileID = msg.Document.FileID
		fileName = msg.Document.FileName
	} else if msg.Photo != nil && len(msg.Photo) > 0 {
		// Handle photos - get the largest resolution
		largestPhoto := msg.Photo[len(msg.Photo)-1]
		fileID = largestPhoto.FileID
		fileName = fmt.Sprintf("photo_%d.jpg", msg.MessageID)
	} else {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Unsupported file type")
		b.api.Send(reply)
		return
	}

	log.Printf("📥 File upload from chat %d: %s (FileID: %s)", msg.Chat.ID, fileName, fileID)

	// Send processing message
	processingMsg := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("⏳ Uploading %s...", fileName))
	sentMsg, err := b.api.Send(processingMsg)
	if err != nil {
		log.Printf("Failed to send processing message: %v", err)
	}

	// Get file info from Telegram
	fileConfig := tgbotapi.FileConfig{FileID: fileID}
	file, err := b.api.GetFile(fileConfig)
	if err != nil {
		log.Printf("❌ Failed to get file info: %v", err)
		editMsg := tgbotapi.NewEditMessageText(msg.Chat.ID, sentMsg.MessageID,
			fmt.Sprintf("❌ Failed to get file: %v", err))
		b.api.Send(editMsg)
		return
	}

	// Download file from Telegram servers
	fileURL := file.Link(b.api.Token)
	resp, err := http.Get(fileURL)
	if err != nil {
		log.Printf("❌ Failed to download file: %v", err)
		editMsg := tgbotapi.NewEditMessageText(msg.Chat.ID, sentMsg.MessageID,
			fmt.Sprintf("❌ Failed to download file: %v", err))
		b.api.Send(editMsg)
		return
	}
	defer resp.Body.Close()

	// Read file content
	fileContent, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Failed to read file content: %v", err)
		editMsg := tgbotapi.NewEditMessageText(msg.Chat.ID, sentMsg.MessageID,
			fmt.Sprintf("❌ Failed to read file: %v", err))
		b.api.Send(editMsg)
		return
	}

	// Save to working directory
	filePath := filepath.Join(chatCtx.WorkingDir, fileName)
	err = os.WriteFile(filePath, fileContent, 0644)
	if err != nil {
		log.Printf("❌ Failed to save file: %v", err)
		editMsg := tgbotapi.NewEditMessageText(msg.Chat.ID, sentMsg.MessageID,
			fmt.Sprintf("❌ Failed to save file: %v", err))
		b.api.Send(editMsg)
		return
	}

	fileSizeMB := float64(len(fileContent)) / (1024 * 1024)
	log.Printf("✅ File saved: %s (%.2f MB) to %s", fileName, fileSizeMB, filePath)

	// Send success message
	editMsg := tgbotapi.NewEditMessageText(msg.Chat.ID, sentMsg.MessageID,
		fmt.Sprintf("✅ File uploaded successfully!\n\n"+
			"📄 Name: %s\n"+
			"📊 Size: %.2f MB\n"+
			"📁 Location: %s\n\n"+
			"The file is now in your current working directory.",
			fileName, fileSizeMB, chatCtx.WorkingDir))
	b.api.Send(editMsg)
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

	// Optional: Chat ID for programmatic access
	var authChatID int64
	chatIDStr := os.Getenv("OMNI_TG_AUTH_CHAT_ID")
	if chatIDStr != "" {
		authChatID, err = strconv.ParseInt(chatIDStr, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("invalid OMNI_TG_AUTH_CHAT_ID: %w", err)
		}
	}

	// Check if using SDK mode (always true now, legacy HTTP mode removed)
	useSDK := os.Getenv("OMNI_USE_CLAUDE_SDK") == "true"

	// Model configuration
	model := os.Getenv("OMNI_CLAUDE_MODEL")
	if model == "" {
		model = "sonnet" // Default to sonnet
	}

	return Config{
		TelegramToken: token,
		AuthorizedUID: uid,
		AuthChatID:    authChatID,
		UseSDK:        useSDK,
		ClaudeModel:   model,
	}, nil
}

// GetSessionManager returns the session manager (for API access)
func (b *Bot) GetSessionManager() *session.Manager {
	return b.sessionManager
}

// ProcessAPIMessage processes a message received via HTTP API
// This simulates receiving a message as if it came from the authorized user in the API chat
func (b *Bot) ProcessAPIMessage(ctx context.Context, message string, sessionID string) error {
	log.Printf("[API] Processing message: %s (session: %s)", message, sessionID)

	// Get or create chat context for API chat
	chatCtx := b.getChatContext(b.authChatID)

	// If session ID provided, try to switch to that session
	if sessionID != "" {
		sess, err := b.sessionManager.Switch(sessionID)
		if err != nil {
			log.Printf("[API] Warning: Failed to switch to session %s: %v", sessionID, err)
		} else if sess != nil {
			// Update chat context with new session
			b.updateChatContext(b.authChatID, sess.Name, sess.WorkingDir)
			chatCtx = b.getChatContext(b.authChatID)
		}
	}

	// Get current session
	currentSession := b.sessionManager.Current()
	if currentSession == nil {
		return fmt.Errorf("no active session")
	}

	// Send "processing" message with stop button
	stopButton := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏹️ Stop", "stop"),
		),
	)
	processingMsg := tgbotapi.NewMessage(b.authChatID, fmt.Sprintf("🔄 API Query: %s", message))
	processingMsg.ReplyMarkup = stopButton
	sentMsg, err := b.api.Send(processingMsg)
	if err != nil {
		log.Printf("[API] Failed to send processing message: %v", err)
		return fmt.Errorf("failed to send processing message: %w", err)
	}

	// Query Claude with the message
	req := claude.QueryRequest{
		Prompt:         message,
		SessionID:      currentSession.ID,
		Workspace:      chatCtx.WorkingDir,
		PermissionMode: "bypassPermissions",
	}

	queryCtx, cancelQuery := context.WithCancel(ctx)
	defer cancelQuery()

	// Create and register stop channel
	stopChan := make(chan struct{})
	b.stopMutex.Lock()
	b.stopChannels[b.authChatID] = stopChan
	b.stopMutex.Unlock()

	defer func() {
		b.stopMutex.Lock()
		delete(b.stopChannels, b.authChatID)
		b.stopMutex.Unlock()
	}()

	responseChan, errorChan := b.claudeClient.Query(queryCtx, req)

	// Track content as chronological events (same as forwardToClaude)
	type contentEvent struct {
		eventType string // "text" or "tool"
		content   string
	}
	var contentHistory []contentEvent
	var lastEdit time.Time
	messageCount := 0
	currentMessageID := sentMsg.MessageID
	messagePartNum := 1
	sentCharCount := 0

	for {
		select {
		case <-stopChan:
			log.Printf("[API] Stop requested by user")
			// Cancel the query
			cancelQuery()

			// Remove stop button from current message
			editMarkup := tgbotapi.EditMessageReplyMarkupConfig{
				BaseEdit: tgbotapi.BaseEdit{
					ChatID:    b.authChatID,
					MessageID: currentMessageID,
				},
			}
			editMarkup.ReplyMarkup = nil
			b.api.Send(editMarkup)

			// Send separate stop notification
			stopMsg := tgbotapi.NewMessage(b.authChatID, "⏹️ Stopped by user")
			b.api.Send(stopMsg)
			return nil

		case err := <-errorChan:
			if err != nil {
				log.Printf("[API] Claude query error: %v", err)
				editMsg := tgbotapi.NewEditMessageText(
					b.authChatID,
					currentMessageID,
					fmt.Sprintf("❌ Error: %v", err),
				)
				editMsg.ReplyMarkup = nil
				b.api.Send(editMsg)
				return fmt.Errorf("claude query error: %w", err)
			}

		case response, ok := <-responseChan:
			if !ok {
				// Channel closed
				return nil
			}

			messageCount++

			switch response.Type {
			case "claude_message":
				// Parse SDK message
				var sdkMsg map[string]interface{}
				if err := json.Unmarshal(response.Data, &sdkMsg); err != nil {
					log.Printf("[API] Failed to parse SDK message: %v", err)
					continue
				}

				// Log message type
				if msgType, ok := sdkMsg["type"].(string); ok {
					log.Printf("[API] Received message type: %s", msgType)
				}

				// Extract session ID if this is a system message
				if msgType, ok := sdkMsg["type"].(string); ok && msgType == "system" {
					if sessionIDVal, ok := sdkMsg["session_id"].(string); ok && sessionIDVal != "" {
						if currentSession.ID == "" {
							currentSession.ID = sessionIDVal
							if err := b.sessionManager.UpdateSessionID(currentSession.Name, sessionIDVal); err != nil {
								log.Printf("[API] Warning: failed to update session ID: %v", err)
							} else {
								log.Printf("[API] Session ID set: %s", sessionIDVal)
							}
						}
					}
				}

				// Extract text and tool_use content from assistant messages
				if msgType, ok := sdkMsg["type"].(string); ok && msgType == "assistant" {
					if msgData, ok := sdkMsg["message"].(map[string]interface{}); ok {
						if content, ok := msgData["content"].([]interface{}); ok {
							for _, item := range content {
								if contentItem, ok := item.(map[string]interface{}); ok {
									contentType, _ := contentItem["type"].(string)

									// Extract text content
									if contentType == "text" {
										if text, ok := contentItem["text"].(string); ok {
											log.Printf("[API] Extracted text content (length: %d): %s...", len(text), text[:min(50, len(text))])
											// Append to last text event or create new one
											if len(contentHistory) > 0 && contentHistory[len(contentHistory)-1].eventType == "text" {
												contentHistory[len(contentHistory)-1].content += text
											} else {
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
											log.Printf("[API] Tool usage: %s", toolStr)
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

				// Update message with rate limiting
				now := time.Now()
				shouldUpdate := messageCount%3 == 0 || time.Since(lastEdit) >= 1000*time.Millisecond

				if shouldUpdate && len(contentHistory) > 0 {
					// Build chronological log from all events
					var displayParts []string
					for _, event := range contentHistory {
						displayParts = append(displayParts, event.content)
					}
					displayText := strings.Join(displayParts, "\n\n")

					log.Printf("[API] Updating message. History items: %d, Display length: %d", len(contentHistory), len(displayText))

					if displayText != "" {
						currentMessageID = b.updateOrSplitMessage(b.authChatID, currentMessageID, displayText, &sentCharCount, &messagePartNum)
						lastEdit = now
					}
				}

			case "done":
				log.Printf("[API] ← Received %d messages from Claude", messageCount)

				// Final update
				if len(contentHistory) == 0 {
					editMsg := tgbotapi.NewEditMessageText(b.authChatID, currentMessageID, "✅ Done (no output)")
					editMsg.ReplyMarkup = nil
					b.api.Send(editMsg)
					return nil
				}

				// Build final display from all events
				var displayParts []string
				for _, event := range contentHistory {
					displayParts = append(displayParts, event.content)
				}
				displayText := strings.Join(displayParts, "\n\n")

				log.Printf("[API] Sending final response (length: %d)", len(displayText))

				// Update message, splitting if necessary
				currentMessageID = b.updateOrSplitMessage(b.authChatID, currentMessageID, displayText, &sentCharCount, &messagePartNum)

				// Remove stop button from final message
				editMarkup := tgbotapi.EditMessageReplyMarkupConfig{
					BaseEdit: tgbotapi.BaseEdit{
						ChatID:    b.authChatID,
						MessageID: currentMessageID,
					},
				}
				editMarkup.ReplyMarkup = nil
				b.api.Send(editMarkup)
				return nil

			case "error":
				log.Printf("[API] Claude error: %s", response.Error)
				editMsg := tgbotapi.NewEditMessageText(
					b.authChatID,
					currentMessageID,
					fmt.Sprintf("❌ Error: %s", response.Error),
				)
				editMsg.ReplyMarkup = nil
				b.api.Send(editMsg)
				return fmt.Errorf("claude error: %s", response.Error)
			}
		}
	}
}

// ExecuteCommand executes a bot command programmatically (for API access)
// NOTE: This function is deprecated and does not support per-chat context isolation.
// It was intended for REST API access which is no longer implemented.
// Commands should be sent via Telegram API to specific chats instead.
// Returns the command result as a map
func (b *Bot) ExecuteCommand(ctx context.Context, command string, args string) (map[string]interface{}, error) {
	log.Printf("[Bot.ExecuteCommand] DEPRECATED: command=%q args=%q", command, args)

	result := make(map[string]interface{})

	switch command {
	case "status":
		currentSession := b.sessionManager.Current()
		if currentSession == nil {
			result["status"] = "no_session"
			result["message"] = "No active session"
		} else {
			result["status"] = "active"
			result["name"] = currentSession.Name
			result["description"] = currentSession.Description
			result["working_dir"] = currentSession.WorkingDir
			result["created_at"] = currentSession.CreatedAt
			result["last_used_at"] = currentSession.LastUsedAt
			result["id"] = currentSession.ID
		}
		return result, nil

	case "sessions":
		sessions := b.sessionManager.List()
		sessionList := make([]map[string]interface{}, 0, len(sessions))
		for _, s := range sessions {
			sessionList = append(sessionList, map[string]interface{}{
				"name":        s.Name,
				"description": s.Description,
				"working_dir": s.WorkingDir,
				"created_at":  s.CreatedAt,
				"last_used_at": s.LastUsedAt,
				"id":          s.ID,
			})
		}
		result["sessions"] = sessionList
		result["count"] = len(sessions)
		return result, nil

	case "pwd", "ls":
		return nil, fmt.Errorf("command %s requires chat context - use Telegram API instead", command)

	default:
		return nil, fmt.Errorf("unsupported command: %s", command)
	}
}

# File Transfer Implementation Guide

This document describes how to implement bidirectional file transfer between users and a Telegram bot in Go using the `telegram-bot-api` library.

## Overview

Two features are implemented:
1. **User to Bot**: Users can send files (documents/photos) to the bot, which are saved to the current working directory
2. **Bot to User**: Users can request files using `/sendfile <filename>` command, and the bot sends them back

## Prerequisites

- `telegram-bot-api` v5 (`github.com/go-telegram-bot-api/telegram-bot-api/v5`)
- Go standard libraries: `io`, `net/http`, `path/filepath`, `os`, `fmt`, `log`

## Feature 1: File Upload (User → Bot)

### Implementation Overview

Files are automatically uploaded when users send documents or photos to the bot. The bot detects the file type, downloads it from Telegram servers, and saves it to the user's current working directory.

### Step 1: Add Message Handler in Main Message Loop

Location: `bot.go` (main message handling function)

```go
// Handle file uploads (documents, photos, etc.)
if msg.Document != nil || msg.Photo != nil {
    go b.handleFileUpload(ctx, msg)
    return
}
```

**Placement**: Add this check AFTER command handling but BEFORE regular text message processing.

### Step 2: Implement handleFileUpload Function

Location: `bot.go` (add at the end of the file)

```go
// handleFileUpload handles file uploads from Telegram
func (b *Bot) handleFileUpload(ctx context.Context, msg *tgbotapi.Message) {
    // Get user's session manager and workspace
    sessionMgr, err := b.getUserSessionManager(msg.From.ID)
    if err != nil {
        log.Printf("❌ Failed to get session manager for user %d: %v", msg.From.ID, err)
        reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Error: Unable to access your workspace")
        b.api.Send(reply)
        return
    }

    // Get current session working directory
    currentSession := sessionMgr.Current()
    if currentSession == nil {
        reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ No active session. Use /newsession to create one first.")
        b.api.Send(reply)
        return
    }

    workingDir := currentSession.WorkingDir

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

    log.Printf("📥 File upload from user %d: %s (FileID: %s)", msg.From.ID, fileName, fileID)

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
    filePath := filepath.Join(workingDir, fileName)
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
            fileName, fileSizeMB, workingDir))
    b.api.Send(editMsg)
}
```

### Required Imports for File Upload

```go
import (
    "io"
    "net/http"
    "path/filepath"
    "os"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)
```

### Key Points for File Upload

1. **File Types Supported**: Documents (`msg.Document`) and Photos (`msg.Photo`)
2. **Photo Handling**: Telegram sends photos in multiple resolutions - always use the last item (largest)
3. **Progress Feedback**: Send a status message that gets updated (processing → success)
4. **Working Directory**: Files save to user's current session working directory
5. **Error Handling**: Graceful error messages for each failure point
6. **Asynchronous**: Run in goroutine (`go b.handleFileUpload()`) to not block message processing

## Feature 2: File Download (Bot → User) - `/sendfile` Command

### Implementation Overview

Users request files using `/sendfile <filename>` command. The bot validates the file exists, checks size limits, and sends it back to the user.

### Step 1: Register Command in Command Switch

Location: `commands.go` (in `executeCommandNew` function)

```go
case "sendfile":
    b.handleSendFileCommand(mctx, args)
```

**Placement**: Add this case in the command switch statement, typically near other file-related commands like `cat` or `ls`.

### Step 2: Implement handleSendFileCommand Function

Location: `commands.go` (add after other file command handlers)

```go
func (b *Bot) handleSendFileCommand(mctx *MessageContext, args string) {
    if args == "" {
        mctx.Send("Usage: /sendfile <filename>")
        return
    }

    // Resolve file path (support both absolute and relative paths)
    var filePath string
    if filepath.IsAbs(args) {
        filePath = args
    } else {
        filePath = filepath.Join(mctx.ChatCtx.WorkingDir, args)
    }

    // Check if file exists
    fileInfo, err := os.Stat(filePath)
    if err != nil {
        if os.IsNotExist(err) {
            mctx.Send(fmt.Sprintf("❌ File not found: %s", args))
        } else {
            mctx.SendError(fmt.Errorf("checking file: %w", err))
        }
        return
    }

    // Check if it's a directory
    if fileInfo.IsDir() {
        mctx.Send(fmt.Sprintf("❌ Cannot send directory: %s\n\nPlease specify a file.", args))
        return
    }

    // Check file size (Telegram has limits - 50MB for bots)
    fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)
    if fileSizeMB > 50 {
        mctx.Send(fmt.Sprintf("❌ File too large: %.2f MB\n\nTelegram limit for bots is 50 MB.", fileSizeMB))
        return
    }

    // Send "preparing file" message
    sentMsg, _ := mctx.Bot.api.Send(tgbotapi.NewMessage(mctx.ChatID,
        fmt.Sprintf("📤 Preparing to send: %s (%.2f MB)...", args, fileSizeMB)))

    // Send the file as document
    doc := tgbotapi.NewDocument(mctx.ChatID, tgbotapi.FilePath(filePath))
    doc.Caption = fmt.Sprintf("📄 %s\n💾 Size: %.2f MB", filepath.Base(filePath), fileSizeMB)

    _, err = mctx.Bot.api.Send(doc)
    if err != nil {
        log.Printf("❌ Failed to send file %s to user %d: %v", filePath, mctx.TelegramID, err)
        editMsg := tgbotapi.NewEditMessageText(mctx.ChatID, sentMsg.MessageID,
            fmt.Sprintf("❌ Failed to send file: %v", err))
        mctx.Bot.api.Send(editMsg)
        return
    }

    // Update status message to success
    editMsg := tgbotapi.NewEditMessageText(mctx.ChatID, sentMsg.MessageID,
        fmt.Sprintf("✅ File sent successfully!\n\n📄 %s\n💾 %.2f MB", args, fileSizeMB))
    mctx.Bot.api.Send(editMsg)

    log.Printf("✓ Sent file %s to user %d (%.2f MB)", filePath, mctx.TelegramID, fileSizeMB)
}
```

### Step 3: Update Help Text

Location: `commands.go` (in `handleStartCommand` function)

```go
"**File Navigation:**\n"+
"/pwd - Show current working directory\n"+
"/ls - List files (ls -lah)\n"+
"/cd <path> - Change directory\n"+
"/cat <file> - Show file contents\n"+
"/sendfile <file> - Send file from workspace to chat\n"+  // Add this line
"/exec <cmd> - Execute bash command\n\n"+
```

### Required Imports for File Download

```go
import (
    "path/filepath"
    "os"
    "fmt"
    "log"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)
```

### Key Points for File Download

1. **Size Limit**: Telegram bots can only send files up to 50MB
2. **Path Resolution**: Supports both absolute paths and relative paths (relative to working directory)
3. **File Type**: Always send as document (`tgbotapi.NewDocument`) for reliability
4. **FilePath Method**: Use `tgbotapi.FilePath(path)` to send from filesystem
5. **Caption**: Include filename and size in the document caption
6. **Progress Feedback**: Show "preparing" message, then update to "success"
7. **Validation**: Check exists, not a directory, within size limits

## Telegram File Size Limits

| Entity Type | Limit |
|-------------|-------|
| Bot Sending Files | 50 MB |
| Users Sending Files | 2 GB |
| Bot Receiving Files | 20 MB (API limit) |

**Note**: Even though users can send up to 2GB files, your bot might only receive files up to 20MB via API. Consider implementing chunked downloads for larger files if needed.

## Error Handling Best Practices

1. **User Feedback**: Always inform users about what's happening (uploading, downloading, errors)
2. **Logging**: Log all file operations with user ID, filename, and size for debugging
3. **Graceful Failures**: Catch all error points (file access, network, Telegram API)
4. **Message Updates**: Use `NewEditMessageText` to update status messages instead of sending new ones
5. **Validation**: Check file existence, type, and size before operations

## Testing Checklist

### File Upload (User → Bot)
- [ ] Send a text document (.txt, .md)
- [ ] Send a PDF file
- [ ] Send a photo/image
- [ ] Send a file with spaces in filename
- [ ] Send a file with special characters
- [ ] Verify file appears in correct directory
- [ ] Check file permissions (0644)
- [ ] Test without active session (should show error)

### File Download (Bot → User)
- [ ] Request existing file with `/sendfile filename.txt`
- [ ] Request file with relative path
- [ ] Request file with absolute path
- [ ] Request non-existent file (should show error)
- [ ] Request directory (should show error)
- [ ] Request 50MB+ file (should show size error)
- [ ] Request file with spaces: `/sendfile "my file.txt"`
- [ ] Verify caption shows correct filename and size

## Adaptation Notes for Other Bots

When implementing in another bot (like Omnik):

1. **Session Management**: Adapt `getUserSessionManager()` and `currentSession.WorkingDir` to your bot's workspace structure
2. **Message Context**: Replace `MessageContext` (`mctx`) with your bot's context structure
3. **Error Handling**: Adapt `mctx.Send()` and `mctx.SendError()` to your bot's reply methods
4. **User Identification**: Adjust `msg.From.ID` to match your user identification system
5. **Working Directory**: Change `workingDir` logic to match your bot's directory structure

## Security Considerations

1. **Path Traversal**: Validate user-provided paths to prevent `../` attacks
2. **File Size Limits**: Always check file sizes to prevent storage exhaustion
3. **File Type Validation**: Consider adding file type restrictions if needed
4. **User Isolation**: Ensure users can only access their own workspace
5. **Disk Space**: Monitor available disk space and implement quotas if necessary
6. **Malware Scanning**: Consider implementing virus scanning for uploaded files

## Performance Tips

1. **Goroutines**: File uploads run asynchronously with `go b.handleFileUpload()`
2. **Streaming**: For very large files, consider streaming instead of loading entirely in memory
3. **Caching**: Telegram files have unique FileIDs - consider caching metadata
4. **Cleanup**: Implement periodic cleanup of old uploaded files
5. **Compression**: Consider compressing large files before sending

## Example User Workflows

### Workflow 1: Upload Configuration File
```
User: [Sends config.json file via Telegram]
Bot: ⏳ Uploading config.json...
Bot: ✅ File uploaded successfully!
     📄 Name: config.json
     📊 Size: 0.05 MB
     📁 Location: /workspace/user_123456/myproject/
```

### Workflow 2: Download Generated File
```
User: /sendfile output.pdf
Bot: 📤 Preparing to send: output.pdf (1.23 MB)...
Bot: [Sends output.pdf]
Bot: ✅ File sent successfully!
     📄 output.pdf
     💾 1.23 MB
```

## Additional File Types Support

To add support for more file types (videos, audio, etc.):

```go
// In handleFileUpload function, add cases:
if msg.Video != nil {
    fileID = msg.Video.FileID
    fileName = fmt.Sprintf("video_%d.mp4", msg.MessageID)
} else if msg.Audio != nil {
    fileID = msg.Audio.FileID
    fileName = msg.Audio.FileName
    if fileName == "" {
        fileName = fmt.Sprintf("audio_%d.mp3", msg.MessageID)
    }
} else if msg.Voice != nil {
    fileID = msg.Voice.FileID
    fileName = fmt.Sprintf("voice_%d.ogg", msg.MessageID)
}
```

## Troubleshooting

### Issue: "Failed to download file"
- **Cause**: Network issues or invalid Telegram token
- **Solution**: Check bot token is valid and bot has internet access

### Issue: "File too large"
- **Cause**: File exceeds Telegram's 50MB bot limit
- **Solution**: Implement file compression or use file hosting service for large files

### Issue: "Permission denied" when saving
- **Cause**: Bot doesn't have write permissions to working directory
- **Solution**: Check directory permissions, ensure bot user has write access

### Issue: Files upload but show wrong directory
- **Cause**: Working directory not properly tracked
- **Solution**: Verify session manager correctly stores/returns working directory

## Complete File Reference

For complete, working implementations, see:
- **File Upload**: `/workspace/holos/holos/go-bot/internal/bot/bot.go` (lines 1248-1355)
- **File Download**: `/workspace/holos/holos/go-bot/internal/bot/commands.go` (lines 354-413)
- **Message Handler**: `/workspace/holos/holos/go-bot/internal/bot/bot.go` (lines 281-285)
- **Command Registration**: `/workspace/holos/holos/go-bot/internal/bot/commands.go` (lines 48-49)

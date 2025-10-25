# Output Buffering Configuration

## Overview

The omnik bot uses a debounce buffer mechanism to batch Claude Code output and send complete prompts/responses to Telegram as single messages instead of fragmented chunks.

## Current Configuration

**Location:** `src/bot/session_manager.py` - `ClaudeCodeProcess._read_output()` method (lines 146-159)

### Buffer Flush Triggers

The buffer flushes (sends output to Telegram) when **either** condition is met:

1. **Timeout Trigger:** No new data received for **500ms** (0.5 seconds)
   - This ensures complete prompts are sent together
   - Waits for Claude Code to finish sending a logical unit of output

2. **Size Trigger:** Buffer exceeds **4KB** (4096 bytes)
   - Prevents memory issues with very large outputs
   - Ensures responsive feedback for long-running operations

### Configuration Values

```python
DEBOUNCE_TIMEOUT = 0.5  # seconds (line 152)
MAX_BUFFER_SIZE = 4096  # bytes (line 152)
```

## Why These Values?

### 500ms Debounce
- Claude Code typically sends prompts in quick bursts
- 500ms is long enough to capture complete prompts with box drawing and options
- Short enough to feel responsive to users (<1 second perceived delay)
- Tested with trust prompts, file confirmations, and multi-line outputs

### 4KB Buffer Size
- Most prompts are 1-2KB (trust prompt is ~1.4KB)
- Leaves headroom for prompts with long file paths or detailed messages
- Prevents buffering entire large file contents (logs, code dumps)
- Keeps memory usage predictable (~4KB per active session)

## When to Adjust

### Increase Timeout (>500ms)
**If:** Prompts still arrive fragmented
- Large prompts with slow network/PTY
- Complex box-drawing taking longer to render
- **Risk:** Increased perceived latency

### Increase Buffer Size (>4KB)
**If:** Very large prompts are being split
- Long file paths (>200 chars) in trust prompts
- Multi-screen prompts with many options
- Large error messages or stack traces
- **Risk:** Memory usage scales with concurrent sessions

### Decrease Timeout (<500ms)
**If:** Users perceive delay as "lagging"
- Trade-off: More fragmentation risk
- **Not recommended** below 300ms

### Decrease Buffer Size (<4KB)
**If:** Memory constraints are critical
- Many concurrent sessions (>50)
- Limited container resources
- **Risk:** Large prompts will fragment

## Monitoring

### Logs to Watch

The buffer flush is logged for debugging:

```json
{"event": "Buffered output flushed", "session_id": "...", "buffer_size": 1382}
```

**Healthy patterns:**
- Buffer sizes: 100-2000 bytes (typical messages)
- Flush intervals: 500-1000ms apart (indicates timeout triggered)

**Warning signs:**
- Buffer size consistently 4096 bytes → Increase MAX_BUFFER_SIZE
- Multiple small flushes (<100 bytes) in rapid succession → Increase timeout
- Very long flush intervals (>5s) → Check for Claude Code hangs

## Implementation Details

### How It Works

1. **PTY Read Loop** runs every 10ms (line 189)
2. Reads up to 1KB chunks from PTY master (line 130)
3. Strips ANSI escape codes (line 136)
4. Appends to buffer and updates `last_output_time` (lines 138-140)
5. Every iteration checks flush conditions (lines 149-159):
   - If `time_since_output > 0.5s` OR `buffer_size > 4096` → flush
6. Flushes any remaining buffer on process exit (lines 184-186)

### Thread Safety

- Buffer is local to `_read_output()` async task
- Only one task per session reads PTY
- Queue writes are thread-safe (asyncio.Queue)
- No race conditions possible

## Testing Recommendations

After changing buffer parameters:

1. **Test trust prompts**: Should arrive as single message with buttons
2. **Test long outputs**: Large code blocks shouldn't cause delays
3. **Test rapid messages**: Quick back-and-forth shouldn't buffer excessively
4. **Monitor logs**: Check buffer sizes and flush frequency
5. **Load test**: Multiple concurrent sessions to verify memory usage

## Future Improvements

### Adaptive Buffering
- Start with small timeout (200ms)
- Increase to 500ms if partial prompt detected (e.g., incomplete option list)
- Reset to small timeout after flush

### Content-Aware Flushing
- Parse for prompt markers (box drawing, "Enter to confirm")
- Flush immediately when complete prompt detected
- Don't wait for timeout if we know prompt is complete

### Size-Based Streaming
- Stream large outputs (>10KB) in chunks instead of buffering
- Keep buffering only for interactive prompts
- Detect output type (prompt vs. log dump vs. file content)

## Related Files

- `src/bot/session_manager.py` - Buffer implementation
- `src/bot/prompt_parser.py` - Prompt detection (works on buffered output)
- `src/bot/handlers.py` - Telegram message handler (receives buffered chunks)

## Change History

- **2025-10-24:** Initial implementation with 200ms / 2KB
- **2025-10-24:** Increased to 500ms / 4KB based on testing with trust prompts

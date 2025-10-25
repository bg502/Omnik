/**
 * Claude Code SDK wrapper for streaming queries
 */

import { query, type PermissionMode } from '@anthropic-ai/claude-code';
import type { QueryRequest, StreamResponse } from './types.js';

/**
 * Remove emojis from text
 * Matches all emoji characters and removes them
 */
function stripEmojis(text: string): string {
  // Comprehensive emoji regex covering all Unicode emoji ranges
  return text.replace(
    /[\u{1F600}-\u{1F64F}\u{1F300}-\u{1F5FF}\u{1F680}-\u{1F6FF}\u{1F700}-\u{1F77F}\u{1F780}-\u{1F7FF}\u{1F800}-\u{1F8FF}\u{1F900}-\u{1F9FF}\u{1FA00}-\u{1FA6F}\u{1FA70}-\u{1FAFF}\u{2600}-\u{26FF}\u{2700}-\u{27BF}\u{2300}-\u{23FF}\u{2B50}\u{2B55}\u{231A}\u{231B}\u{2328}\u{23CF}\u{23E9}-\u{23F3}\u{23F8}-\u{23FA}\u{24C2}\u{25AA}\u{25AB}\u{25B6}\u{25C0}\u{25FB}-\u{25FE}\u{2934}\u{2935}\u{2B05}-\u{2B07}\u{2B1B}\u{2B1C}\u{3030}\u{303D}\u{3297}\u{3299}\u{FE0F}]/gu,
    ''
  );
}

/**
 * Recursively strip emojis from SDK message objects
 * Handles nested structures like message.content[].text
 */
function stripEmojisFromMessage(message: any): any {
  if (typeof message === 'string') {
    return stripEmojis(message);
  }

  if (Array.isArray(message)) {
    return message.map(item => stripEmojisFromMessage(item));
  }

  if (message && typeof message === 'object') {
    const cleaned: any = {};
    for (const [key, value] of Object.entries(message)) {
      // Strip emojis from text fields
      if (key === 'text' && typeof value === 'string') {
        cleaned[key] = stripEmojis(value);
      } else {
        cleaned[key] = stripEmojisFromMessage(value);
      }
    }
    return cleaned;
  }

  return message;
}

/**
 * Configuration for Claude Code execution
 */
export interface ClaudeConfig {
  /** Path to claude executable (auto-detected if not provided) */
  claudePath?: string;

  /** Default permission mode if not specified in request */
  defaultPermissionMode?: PermissionMode;

  /** Default model to use if not specified in request */
  defaultModel?: string;
}

/**
 * Execute a Claude Code query and stream responses
 *
 * @param request - Query request with prompt and options
 * @param config - Claude configuration
 * @yields StreamResponse objects (claude_message, error, or done)
 */
export async function* executeClaudeQuery(
  request: QueryRequest,
  config: ClaudeConfig = {}
): AsyncGenerator<StreamResponse> {
  // Validate request
  if (!request.prompt || request.prompt.trim().length === 0) {
    yield {
      type: 'error',
      error: 'Prompt cannot be empty',
      code: 'INVALID_REQUEST',
    };
    return;
  }

  // Determine which model to use and save original for restoration
  const modelToUse = request.model || config.defaultModel;
  const originalModel = process.env.ANTHROPIC_MODEL;

  // Set model via environment variable if specified
  // Note: We need to set this before calling query()
  if (modelToUse) {
    process.env.ANTHROPIC_MODEL = modelToUse;
  }

  try {

    // Build SDK query options
    const queryOptions: Parameters<typeof query>[0] = {
      prompt: request.prompt,
      options: {
        // Use node as executable (SDK will handle finding it)
        executable: 'node' as const,
        executableArgs: [],

        // Use provided claude path or let SDK auto-detect
        ...(config.claudePath
          ? { pathToClaudeCodeExecutable: config.claudePath }
          : {}),

        // Session continuity
        ...(request.sessionId ? { resume: request.sessionId } : {}),

        // Working directory
        ...(request.workspace ? { cwd: request.workspace } : {}),

        // Permission mode
        permissionMode:
          request.permissionMode || config.defaultPermissionMode || 'default',

        // Tool restrictions
        ...(request.allowedTools ? { allowedTools: request.allowedTools } : {}),
      },
    };

    // Execute query and stream SDK messages
    const promptPreview = typeof queryOptions.prompt === 'string'
      ? queryOptions.prompt.substring(0, 50) + '...'
      : '[AsyncIterable prompt]';
    console.log('[CLAUDE] Starting query with options:', JSON.stringify({
      ...queryOptions,
      prompt: promptPreview
    }));

    for await (const sdkMessage of query(queryOptions)) {
      // Strip emojis from SDK message if it contains text content
      const cleanedMessage = stripEmojisFromMessage(sdkMessage);

      yield {
        type: 'claude_message',
        data: cleanedMessage,
      };
    }

    console.log('[CLAUDE] Query completed successfully');

    // Restore original environment variable
    if (originalModel !== undefined) {
      process.env.ANTHROPIC_MODEL = originalModel;
    } else {
      delete process.env.ANTHROPIC_MODEL;
    }

    // Signal completion
    yield { type: 'done' };
  } catch (error) {
    // Restore original environment variable in error case too
    if (originalModel !== undefined) {
      process.env.ANTHROPIC_MODEL = originalModel;
    } else {
      delete process.env.ANTHROPIC_MODEL;
    }

    // Handle errors gracefully
    const errorMessage = error instanceof Error ? error.message : String(error);
    const errorStack = error instanceof Error ? error.stack : undefined;

    console.error('[CLAUDE] Error during query:', errorMessage);
    console.error('[CLAUDE] Error stack:', errorStack);
    console.error('[CLAUDE] Full error object:', error);

    // Determine error code based on message
    let code: string = 'INTERNAL_ERROR';
    if (errorMessage.includes('authentication') || errorMessage.includes('auth')) {
      code = 'AUTHENTICATION_ERROR';
    } else if (errorMessage.includes('workspace') || errorMessage.includes('directory')) {
      code = 'WORKSPACE_ERROR';
    } else if (errorMessage.includes('claude') && errorMessage.includes('not found')) {
      code = 'CLAUDE_NOT_FOUND';
    } else if (errorMessage.includes('execution') || errorMessage.includes('exit')) {
      code = 'CLAUDE_EXEC_ERROR';
    }

    yield {
      type: 'error',
      error: errorMessage,
      code,
    };
  }
}

/**
 * Check if Claude CLI is available and authenticated
 *
 * @returns Object with status and version info
 */
export async function checkClaudeHealth(): Promise<{
  available: boolean;
  version?: string;
  error?: string;
}> {
  try {
    // Try a simple query to verify Claude is working
    const testQuery = query({
      prompt: '/version',
      options: {
        executable: 'node' as const,
        executableArgs: [],
        permissionMode: 'default',
      },
    });

    // Collect first few messages to check if working
    let version: string | undefined;
    let messageCount = 0;

    for await (const message of testQuery) {
      messageCount++;

      // Extract version from system message if available
      if (message.type === 'system' && 'version' in message) {
        version = (message as any).version;
      }

      // Break after a few messages to avoid long wait
      if (messageCount >= 3) {
        break;
      }
    }

    return {
      available: true,
      version,
    };
  } catch (error) {
    return {
      available: false,
      error: error instanceof Error ? error.message : String(error),
    };
  }
}

/**
 * Type definitions for omnik Claude Bridge API
 * Shared between Go bot client and Node.js bridge server
 */

import type { PermissionMode } from '@anthropic-ai/claude-code';

/**
 * Request to query Claude Code
 */
export interface QueryRequest {
  /** User's prompt/message to Claude */
  prompt: string;

  /** Optional session ID for conversation continuity */
  sessionId?: string;

  /** Working directory for Claude execution */
  workspace?: string;

  /** Permission mode: "default" | "plan" | etc */
  permissionMode?: PermissionMode;

  /** Array of allowed tool names (restricts Claude's capabilities) */
  allowedTools?: string[];

  /** Claude model to use (e.g., 'sonnet', 'opus', 'haiku', or full model name) */
  model?: string;
}

/**
 * Streaming response message types
 */
export type StreamResponse =
  | ClaudeMessageResponse
  | ErrorResponse
  | DoneResponse;

/**
 * Claude SDK message wrapper
 */
export interface ClaudeMessageResponse {
  type: 'claude_message';

  /** Raw SDK message from @anthropic-ai/claude-code */
  data: any; // SDK message structure varies by type
}

/**
 * Error response
 */
export interface ErrorResponse {
  type: 'error';

  /** Error message */
  error: string;

  /** Optional error code */
  code?: string;
}

/**
 * Stream completion marker
 */
export interface DoneResponse {
  type: 'done';
}

/**
 * Health check response
 */
export interface HealthResponse {
  status: 'ok' | 'error';
  version: string;
  claudeVersion?: string;
}

/**
 * Error codes for structured error handling
 */
export enum ErrorCode {
  INVALID_REQUEST = 'INVALID_REQUEST',
  CLAUDE_NOT_FOUND = 'CLAUDE_NOT_FOUND',
  CLAUDE_EXEC_ERROR = 'CLAUDE_EXEC_ERROR',
  AUTHENTICATION_ERROR = 'AUTHENTICATION_ERROR',
  WORKSPACE_ERROR = 'WORKSPACE_ERROR',
  INTERNAL_ERROR = 'INTERNAL_ERROR',
}

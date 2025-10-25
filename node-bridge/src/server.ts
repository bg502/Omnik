/**
 * Express server with Server-Sent Events for Claude Code streaming
 */

import express, { Request, Response } from 'express';
import cors from 'cors';
import { executeClaudeQuery, checkClaudeHealth, type ClaudeConfig } from './claude.js';
import type { QueryRequest } from './types.js';

// Configuration
const PORT = parseInt(process.env.PORT || '9000', 10);
const HOST = process.env.HOST || '0.0.0.0';
const CLAUDE_PATH = process.env.CLAUDE_PATH; // Optional custom claude path

const config: ClaudeConfig = {
  claudePath: CLAUDE_PATH,
  defaultPermissionMode: 'bypassPermissions', // Skip all permission prompts for autonomous operation
  defaultModel: 'sonnet', // Use Sonnet by default (Opus limit exhausted)
};

// Initialize Express app
const app = express();

// Middleware
app.use(cors());
app.use(express.json());

// Request logging middleware
app.use((req, _res, next) => {
  const timestamp = new Date().toISOString();
  console.log(`[${timestamp}] ${req.method} ${req.path}`);
  next();
});

/**
 * Health check endpoint
 * GET /health
 */
app.get('/health', async (_req: Request, res: Response) => {
  try {
    const health = await checkClaudeHealth();

    if (health.available) {
      res.json({
        status: 'ok',
        version: '1.0.0',
        claudeVersion: health.version,
      });
    } else {
      res.status(503).json({
        status: 'error',
        version: '1.0.0',
      });
    }
  } catch (error) {
    res.status(500).json({
      status: 'error',
      version: '1.0.0',
    });
  }
});

/**
 * Claude query endpoint with Server-Sent Events streaming
 * POST /api/query
 *
 * Request body: QueryRequest
 * Response: text/event-stream with StreamResponse objects
 */
app.post('/api/query', async (req: Request, res: Response) => {
  try {
    const queryRequest: QueryRequest = req.body;

    // Validate request
    if (!queryRequest.prompt) {
      res.status(400).json({
        error: 'Missing required field: prompt',
      });
      return;
    }

    // Set headers for Server-Sent Events
    res.setHeader('Content-Type', 'text/event-stream');
    res.setHeader('Cache-Control', 'no-cache');
    res.setHeader('Connection', 'keep-alive');
    res.setHeader('X-Accel-Buffering', 'no'); // Disable nginx buffering

    // Log query start
    console.log(`[QUERY] Session: ${queryRequest.sessionId || 'new'}, Workspace: ${queryRequest.workspace || 'default'}`);
    console.log(`[QUERY] Prompt: ${queryRequest.prompt.substring(0, 100)}${queryRequest.prompt.length > 100 ? '...' : ''}`);

    // Execute Claude query and stream responses
    let messageCount = 0;
    for await (const response of executeClaudeQuery(queryRequest, config)) {
      // Send response as SSE data
      const data = JSON.stringify(response);
      res.write(`data: ${data}\n\n`);

      messageCount++;

      // Log completion
      if (response.type === 'done') {
        console.log(`[QUERY] Completed: ${messageCount} messages streamed`);
      } else if (response.type === 'error') {
        console.error(`[QUERY] Error: ${response.error}`);
      }
    }

    // Close the stream
    res.end();
  } catch (error) {
    console.error('[QUERY] Unexpected error:', error);

    // Try to send error response if headers not sent
    if (!res.headersSent) {
      res.status(500).json({
        error: error instanceof Error ? error.message : 'Internal server error',
      });
    } else {
      // Send error via SSE if already streaming
      const errorData = JSON.stringify({
        type: 'error',
        error: error instanceof Error ? error.message : 'Internal server error',
      });
      res.write(`data: ${errorData}\n\n`);
      res.end();
    }
  }
});

/**
 * 404 handler
 */
app.use((_req, res) => {
  res.status(404).json({
    error: 'Not found',
    endpoints: {
      health: 'GET /health',
      query: 'POST /api/query',
    },
  });
});

/**
 * Error handler
 */
app.use((err: Error, _req: Request, res: Response) => {
  console.error('[ERROR]', err);
  res.status(500).json({
    error: 'Internal server error',
    message: err.message,
  });
});

/**
 * Start server
 */
async function start() {
  // Check Claude availability on startup
  console.log('[STARTUP] Checking Claude Code availability...');
  const health = await checkClaudeHealth();

  if (health.available) {
    console.log(`[STARTUP] ✓ Claude Code available: ${health.version || 'version unknown'}`);
  } else {
    console.warn(`[STARTUP] ⚠ Claude Code health check failed: ${health.error}`);
    console.warn('[STARTUP] Server will start anyway, but queries may fail');
  }

  // Start listening
  app.listen(PORT, HOST, () => {
    console.log('[STARTUP] ================================');
    console.log(`[STARTUP] omnik Claude Bridge v1.0.0`);
    console.log(`[STARTUP] Listening on http://${HOST}:${PORT}`);
    console.log('[STARTUP] ================================');
    console.log(`[STARTUP] Health check: GET  /health`);
    console.log(`[STARTUP] Query endpoint: POST /api/query`);
    console.log('[STARTUP] ================================');
  });
}

// Handle graceful shutdown
process.on('SIGTERM', () => {
  console.log('[SHUTDOWN] Received SIGTERM, shutting down gracefully...');
  process.exit(0);
});

process.on('SIGINT', () => {
  console.log('[SHUTDOWN] Received SIGINT, shutting down gracefully...');
  process.exit(0);
});

// Start the server
start().catch((error) => {
  console.error('[FATAL] Failed to start server:', error);
  process.exit(1);
});

"""Session manager for Claude Code subprocess control."""
import asyncio
import uuid
import signal
import os
import pty
import fcntl
import re
from pathlib import Path
from typing import Dict, Optional, AsyncIterator, List
from datetime import datetime
import structlog

from ..models import Session, SessionStatus
from ..database import DatabaseManager

logger = structlog.get_logger()

# ANSI escape code pattern for stripping terminal codes
# More comprehensive pattern to catch all ANSI/VT100 escape sequences
ANSI_ESCAPE = re.compile(r'''
    \x1B  # ESC
    (?:   # 7-bit C1 Fe (except CSI)
        [@-Z\\-_]
    |     # or [ for CSI, followed by control string
        \[
        [0-?]*  # Parameter bytes
        [ -/]*  # Intermediate bytes
        [@-~]   # Final byte
    )
    |     # Also catch bare CSI sequences (broken output without ESC prefix)
    \[(?:
        [0-9;]+[a-zA-Z]  # Bare CSI with numbers and letter (e.g., [3;1R)
        |[0-9;]+m        # Bare SGR (color codes)
    )
''', re.VERBOSE)


class ClaudeCodeProcess:
    """Manages a single Claude Code subprocess."""

    def __init__(
        self,
        session_id: str,
        workspace: Path,
        anthropic_key: Optional[str] = None,
    ):
        self.session_id = session_id
        self.workspace = workspace
        self.anthropic_key = anthropic_key
        self.process: Optional[asyncio.subprocess.Process] = None
        self._output_queue: asyncio.Queue = asyncio.Queue()
        self._reading_task: Optional[asyncio.Task] = None
        self._master_fd: Optional[int] = None
        self._slave_fd: Optional[int] = None

    async def start(self) -> int:
        """Start Claude Code subprocess with PTY, return PID."""
        logger.info("Starting Claude Code process with PTY", session_id=self.session_id)

        # Create workspace if it doesn't exist
        self.workspace.mkdir(parents=True, exist_ok=True)

        # Prepare environment variables
        env = os.environ.copy()
        env.update({
            "CLAUDE_TELEMETRY_OPTOUT": "1",
            "TERM": "xterm-256color",  # Set terminal type for PTY
        })

        # Only set API key if provided (otherwise use existing Pro account auth)
        if self.anthropic_key:
            env["ANTHROPIC_API_KEY"] = self.anthropic_key
            logger.info("Using provided API key", session_id=self.session_id)
        else:
            logger.info("Using existing Claude Pro account authentication", session_id=self.session_id)

        # Create PTY (pseudo-terminal)
        self._master_fd, self._slave_fd = pty.openpty()

        # Make master non-blocking for async reads
        flags = fcntl.fcntl(self._master_fd, fcntl.F_GETFL)
        fcntl.fcntl(self._master_fd, fcntl.F_SETFL, flags | os.O_NONBLOCK)

        # Start Claude Code as subprocess with PTY
        # Using permission-mode=default for now (will ask for permissions)
        self.process = await asyncio.create_subprocess_exec(
            "claude",
            "--permission-mode", "default",
            stdin=self._slave_fd,
            stdout=self._slave_fd,
            stderr=self._slave_fd,
            cwd=str(self.workspace),
            env=env,
            preexec_fn=os.setsid,  # Create new session
        )

        # Close slave fd in parent process (child has its own copy)
        os.close(self._slave_fd)
        self._slave_fd = None

        # Start background task to read output
        self._reading_task = asyncio.create_task(self._read_output())

        logger.info(
            "Claude Code process started with PTY",
            session_id=self.session_id,
            pid=self.process.pid,
        )

        return self.process.pid

    async def _read_output(self):
        """Background task to read from PTY master and queue output."""
        if not self.process or self._master_fd is None:
            return

        try:
            loop = asyncio.get_event_loop()
            buffer = ""
            last_output_time = asyncio.get_event_loop().time()

            while True:
                # Read from PTY master in non-blocking mode
                try:
                    # Use run_in_executor for non-blocking read
                    data = await loop.run_in_executor(
                        None,
                        self._try_read_pty,
                        1024  # Read up to 1KB at a time
                    )

                    if data:
                        # Decode and strip ANSI escape codes
                        text = data.decode("utf-8", errors="ignore")
                        clean_text = ANSI_ESCAPE.sub('', text)

                        if clean_text.strip():  # Only buffer non-empty output
                            buffer += clean_text
                            last_output_time = asyncio.get_event_loop().time()

                except Exception as e:
                    # If read fails, likely EOF or process ended
                    pass

                # Flush buffer if:
                # 1. No new data for 500ms (debounce complete prompts)
                # 2. Buffer is getting large (> 4KB)
                current_time = asyncio.get_event_loop().time()
                time_since_output = current_time - last_output_time

                if buffer and (time_since_output > 0.5 or len(buffer) > 4096):
                    await self._output_queue.put(buffer)
                    logger.info(
                        "Buffered output flushed",
                        session_id=self.session_id,
                        buffer_size=len(buffer)
                    )
                    buffer = ""

                # Check if process has exited
                if self.process.returncode is not None:
                    logger.warning(
                        "Claude Code process exited",
                        session_id=self.session_id,
                        returncode=self.process.returncode,
                    )
                    # Try to read any remaining output and flush buffer
                    try:
                        remaining = self._try_read_pty(4096)
                        if remaining:
                            text = remaining.decode("utf-8", errors="ignore")
                            clean_text = ANSI_ESCAPE.sub('', text)
                            if clean_text.strip():
                                buffer += clean_text
                    except:
                        pass

                    # Flush any remaining buffer
                    if buffer:
                        await self._output_queue.put(buffer)
                        buffer = ""
                    break

                await asyncio.sleep(0.01)

        except Exception as e:
            logger.error(
                "Error reading output", session_id=self.session_id, error=str(e)
            )
        finally:
            # Close master FD when done
            if self._master_fd is not None:
                try:
                    os.close(self._master_fd)
                except:
                    pass
                self._master_fd = None

    def _try_read_pty(self, size: int) -> bytes:
        """Try to read from PTY master, return empty bytes if nothing available."""
        try:
            if self._master_fd is None:
                return b""
            return os.read(self._master_fd, size)
        except (OSError, BlockingIOError):
            # No data available or FD closed
            return b""

    async def send_input(self, text: str):
        """Send input to Claude Code via PTY."""
        if not self.process or self._master_fd is None:
            raise RuntimeError("Process not started or PTY not available")

        # Write to PTY master
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(
            None,
            os.write,
            self._master_fd,
            text.encode("utf-8") + b"\n"
        )

    async def read_output(self, timeout: float = 30.0) -> AsyncIterator[str]:
        """Stream stdout/stderr from Claude Code."""
        start_time = asyncio.get_event_loop().time()

        while True:
            try:
                # Get output with timeout
                remaining = timeout - (asyncio.get_event_loop().time() - start_time)
                if remaining <= 0:
                    break

                try:
                    output = await asyncio.wait_for(
                        self._output_queue.get(), timeout=min(remaining, 1.0)
                    )
                    yield output
                except asyncio.TimeoutError:
                    # Check if there's more output coming
                    if self._output_queue.empty() and self.process and self.process.returncode is not None:
                        break
                    continue

            except Exception as e:
                logger.error(
                    "Error reading output", session_id=self.session_id, error=str(e)
                )
                break

    async def terminate(self, timeout: int = 10):
        """Send SIGTERM, wait for graceful shutdown, fallback to SIGKILL."""
        if not self.process:
            return

        logger.info("Terminating Claude Code process", session_id=self.session_id)

        try:
            # Send SIGTERM
            self.process.send_signal(signal.SIGTERM)

            # Wait for graceful shutdown
            try:
                await asyncio.wait_for(self.process.wait(), timeout=timeout)
                logger.info(
                    "Claude Code process terminated gracefully",
                    session_id=self.session_id,
                )
            except asyncio.TimeoutError:
                # Force kill
                logger.warning(
                    "Claude Code process did not terminate, forcing kill",
                    session_id=self.session_id,
                )
                self.process.kill()
                await self.process.wait()

        except Exception as e:
            logger.error(
                "Error terminating process", session_id=self.session_id, error=str(e)
            )

        finally:
            # Cancel reading task
            if self._reading_task and not self._reading_task.done():
                self._reading_task.cancel()
                try:
                    await self._reading_task
                except asyncio.CancelledError:
                    pass

            # Clean up PTY file descriptors
            if self._master_fd is not None:
                try:
                    os.close(self._master_fd)
                except:
                    pass
                self._master_fd = None

            if self._slave_fd is not None:
                try:
                    os.close(self._slave_fd)
                except:
                    pass
                self._slave_fd = None

    def is_alive(self) -> bool:
        """Check if subprocess is running."""
        return self.process is not None and self.process.returncode is None


class SessionManager:
    """Manages multiple Claude Code sessions."""

    def __init__(self, db: DatabaseManager, workspace_base: Path, anthropic_key: Optional[str] = None):
        self.db = db
        self.workspace_base = workspace_base
        self.anthropic_key = anthropic_key
        self.sessions: Dict[str, ClaudeCodeProcess] = {}
        self.active_sessions: Dict[int, str] = {}  # user_id -> session_id

    async def create_session(
        self, user_id: int, name: Optional[str] = None
    ) -> Session:
        """Create new Claude Code session with isolated workspace."""
        session_id = str(uuid.uuid4())
        workspace_path = self.workspace_base / session_id

        logger.info(
            "Creating new session",
            session_id=session_id,
            user_id=user_id,
            name=name,
        )

        # Create session model
        session = Session(
            id=session_id,
            user_id=user_id,
            workspace_path=str(workspace_path),
            name=name,
            status=SessionStatus.ACTIVE,
        )

        # Save to database
        await self.db.create_session(session)

        # Start Claude Code process
        process = ClaudeCodeProcess(
            session_id=session_id,
            workspace=workspace_path,
            anthropic_key=self.anthropic_key,
        )

        pid = await process.start()
        session.pid = pid

        # Update database with PID
        await self.db.update_session(session_id, pid=pid)

        # Store in memory
        self.sessions[session_id] = process

        # Set as active session for user
        self.active_sessions[user_id] = session_id

        logger.info(
            "Session created successfully",
            session_id=session_id,
            pid=pid,
        )

        return session

    async def get_session(self, session_id: str) -> Optional[Session]:
        """Retrieve session by ID."""
        return await self.db.get_session(session_id)

    async def list_sessions(self, user_id: int) -> List[Session]:
        """List all sessions for user."""
        return await self.db.list_sessions(user_id)

    async def get_active_session(self, user_id: int) -> Optional[Session]:
        """Get the active session for a user."""
        session_id = self.active_sessions.get(user_id)
        if session_id:
            return await self.get_session(session_id)
        return None

    async def set_active_session(self, user_id: int, session_id: str) -> bool:
        """Set the active session for a user."""
        session = await self.get_session(session_id)
        if session and session.user_id == user_id:
            self.active_sessions[user_id] = session_id
            return True
        return False

    async def terminate_session(self, session_id: str) -> bool:
        """Gracefully stop Claude Code subprocess and cleanup."""
        logger.info("Terminating session", session_id=session_id)

        # Get process
        process = self.sessions.get(session_id)
        if process:
            await process.terminate()
            del self.sessions[session_id]

        # Update database
        await self.db.update_session(
            session_id,
            status=SessionStatus.TERMINATED,
            last_activity=datetime.utcnow(),
        )

        # Remove from active sessions
        for user_id, active_id in list(self.active_sessions.items()):
            if active_id == session_id:
                del self.active_sessions[user_id]

        logger.info("Session terminated", session_id=session_id)
        return True

    async def restart_session(self, session_id: str) -> bool:
        """Restart crashed or hung session."""
        logger.info("Restarting session", session_id=session_id)

        session = await self.get_session(session_id)
        if not session:
            return False

        # Terminate existing process
        process = self.sessions.get(session_id)
        if process:
            await process.terminate()

        # Start new process
        workspace_path = Path(session.workspace_path)
        new_process = ClaudeCodeProcess(
            session_id=session_id,
            workspace=workspace_path,
            anthropic_key=self.anthropic_key,
        )

        pid = await new_process.start()

        # Update database
        await self.db.update_session(
            session_id,
            pid=pid,
            status=SessionStatus.ACTIVE,
            last_activity=datetime.utcnow(),
        )

        # Store in memory
        self.sessions[session_id] = new_process

        logger.info("Session restarted", session_id=session_id, pid=pid)
        return True

    async def send_message(
        self, session_id: str, message: str
    ) -> AsyncIterator[str]:
        """Send message to Claude Code and stream response."""
        process = self.sessions.get(session_id)
        if not process:
            raise RuntimeError(f"Session {session_id} not found or not running")

        # Log the request
        logger.info(
            "→ Sending message to Claude Code",
            session_id=session_id,
            message_preview=message[:100] + "..." if len(message) > 100 else message
        )

        # Check if process is still alive
        if not process.is_alive():
            logger.error(
                "Claude Code process is not running",
                session_id=session_id,
                returncode=process.process.returncode if process.process else None,
            )
            raise RuntimeError(
                f"Claude Code process has crashed (exit code: {process.process.returncode if process.process else 'unknown'}). "
                f"Please check logs or restart the session with /restart"
            )

        # Send input
        try:
            await process.send_input(message)
        except Exception as e:
            logger.error("Failed to send input to Claude Code", session_id=session_id, error=str(e))
            raise RuntimeError(f"Failed to send message: {str(e)}")

        # Update last activity
        await self.db.update_session(session_id, last_activity=datetime.utcnow())

        # Stream output
        has_output = False
        full_response = ""
        async for output in process.read_output(timeout=60.0):
            has_output = True
            full_response += output
            yield output

        # Log the complete response
        if has_output:
            logger.info(
                "← Received response from Claude Code",
                session_id=session_id,
                response_length=len(full_response),
                response_preview=full_response[:200] + "..." if len(full_response) > 200 else full_response
            )
        else:
            yield "✅ Command sent (no output received)"
            logger.info("← No output received from Claude Code", session_id=session_id)

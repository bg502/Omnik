"""Session manager using Claude Agent SDK."""
import uuid
from pathlib import Path
from typing import Dict, Optional, AsyncIterator, List
from datetime import datetime
import structlog

from ..models import Session, SessionStatus
from ..database import DatabaseManager
from ..agent import ClaudeAgent

logger = structlog.get_logger()


class SessionManager:
    """Manages multiple Claude Agent sessions."""

    def __init__(self, db: DatabaseManager, workspace_base: Path, anthropic_key: Optional[str] = None):
        self.db = db
        self.workspace_base = workspace_base
        self.anthropic_key = anthropic_key
        self.agents: Dict[str, ClaudeAgent] = {}  # session_id -> agent
        self.active_sessions: Dict[int, str] = {}  # user_id -> session_id

    async def create_session(
        self, user_id: int, name: Optional[str] = None
    ) -> Session:
        """Create new Claude Agent session with isolated workspace."""
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

        # Create Claude agent
        agent = ClaudeAgent(
            workspace=workspace_path,
            api_key=self.anthropic_key,
        )

        # Store agent
        self.agents[session_id] = agent

        # Set as active session for user
        self.active_sessions[user_id] = session_id

        logger.info(
            "Session created successfully",
            session_id=session_id,
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

            # Ensure agent is loaded
            if session_id not in self.agents:
                workspace_path = Path(session.workspace_path)
                agent = ClaudeAgent(
                    workspace=workspace_path,
                    api_key=self.anthropic_key,
                )
                self.agents[session_id] = agent

            return True
        return False

    async def terminate_session(self, session_id: str) -> bool:
        """Terminate a session and cleanup."""
        logger.info("Terminating session", session_id=session_id)

        # Remove agent
        if session_id in self.agents:
            del self.agents[session_id]

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
        """Restart a session (clear conversation)."""
        logger.info("Restarting session", session_id=session_id)

        session = await self.get_session(session_id)
        if not session:
            return False

        # Get or create agent
        if session_id in self.agents:
            agent = self.agents[session_id]
            agent.clear_conversation()
        else:
            workspace_path = Path(session.workspace_path)
            agent = ClaudeAgent(
                workspace=workspace_path,
                api_key=self.anthropic_key,
            )
            self.agents[session_id] = agent

        # Update database
        await self.db.update_session(
            session_id,
            status=SessionStatus.ACTIVE,
            last_activity=datetime.utcnow(),
        )

        logger.info("Session restarted", session_id=session_id)
        return True

    async def send_message(
        self, session_id: str, message: str
    ) -> AsyncIterator[str]:
        """Send message to Claude Agent and stream response."""
        # Get or create agent
        if session_id not in self.agents:
            session = await self.get_session(session_id)
            if not session:
                raise RuntimeError(f"Session {session_id} not found")

            workspace_path = Path(session.workspace_path)
            agent = ClaudeAgent(
                workspace=workspace_path,
                api_key=self.anthropic_key,
            )
            self.agents[session_id] = agent

        agent = self.agents[session_id]

        logger.info("Sending message to Claude Agent", session_id=session_id)

        # Update last activity
        await self.db.update_session(session_id, last_activity=datetime.utcnow())

        # Stream response from agent
        async for chunk in agent.send_message(message):
            yield chunk

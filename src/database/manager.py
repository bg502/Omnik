"""Database manager for CRUD operations."""
from typing import List, Optional
from datetime import datetime
from sqlalchemy import select, update, delete
from sqlalchemy.ext.asyncio import AsyncSession

from .schema import SessionDB, MessageDB, AuditLogDB, get_db
from ..models import Session, Message, AuditLog, SessionStatus, MessageRole


class DatabaseManager:
    """Database manager for CRUD operations."""

    async def create_session(self, session: Session) -> Session:
        """Create a new session."""
        async with get_db() as db:
            db_session = SessionDB(
                id=session.id,
                user_id=session.user_id,
                workspace_path=session.workspace_path,
                name=session.name,
                status=session.status.value,
                pid=session.pid,
                created_at=session.created_at,
                last_activity=session.last_activity,
                token_usage=session.token_usage,
                cost_usd=session.cost_usd,
            )
            db.add(db_session)
            await db.commit()
            return session

    async def get_session(self, session_id: str) -> Optional[Session]:
        """Get a session by ID."""
        async with get_db() as db:
            result = await db.execute(
                select(SessionDB).where(SessionDB.id == session_id)
            )
            db_session = result.scalar_one_or_none()

            if db_session is None:
                return None

            return Session(
                id=db_session.id,
                user_id=db_session.user_id,
                workspace_path=db_session.workspace_path,
                name=db_session.name,
                status=SessionStatus(db_session.status),
                pid=db_session.pid,
                created_at=db_session.created_at,
                last_activity=db_session.last_activity,
                token_usage=db_session.token_usage,
                cost_usd=db_session.cost_usd,
            )

    async def list_sessions(self, user_id: int, status: Optional[SessionStatus] = None) -> List[Session]:
        """List all sessions for a user."""
        async with get_db() as db:
            query = select(SessionDB).where(SessionDB.user_id == user_id)

            if status is not None:
                query = query.where(SessionDB.status == status.value)

            result = await db.execute(query.order_by(SessionDB.created_at.desc()))
            db_sessions = result.scalars().all()

            return [
                Session(
                    id=s.id,
                    user_id=s.user_id,
                    workspace_path=s.workspace_path,
                    name=s.name,
                    status=SessionStatus(s.status),
                    pid=s.pid,
                    created_at=s.created_at,
                    last_activity=s.last_activity,
                    token_usage=s.token_usage,
                    cost_usd=s.cost_usd,
                )
                for s in db_sessions
            ]

    async def update_session(
        self,
        session_id: str,
        status: Optional[SessionStatus] = None,
        pid: Optional[int] = None,
        last_activity: Optional[datetime] = None,
        token_usage: Optional[int] = None,
        cost_usd: Optional[float] = None,
    ) -> bool:
        """Update session fields."""
        async with get_db() as db:
            values = {}
            if status is not None:
                values["status"] = status.value
            if pid is not None:
                values["pid"] = pid
            if last_activity is not None:
                values["last_activity"] = last_activity
            if token_usage is not None:
                values["token_usage"] = token_usage
            if cost_usd is not None:
                values["cost_usd"] = cost_usd

            if not values:
                return False

            result = await db.execute(
                update(SessionDB)
                .where(SessionDB.id == session_id)
                .values(**values)
            )
            await db.commit()
            return result.rowcount > 0

    async def delete_session(self, session_id: str) -> bool:
        """Delete a session."""
        async with get_db() as db:
            result = await db.execute(
                delete(SessionDB).where(SessionDB.id == session_id)
            )
            await db.commit()
            return result.rowcount > 0

    async def add_message(self, message: Message) -> Message:
        """Add a message to the database."""
        async with get_db() as db:
            db_message = MessageDB(
                session_id=message.session_id,
                role=message.role.value,
                content=message.content,
                timestamp=message.timestamp,
                token_count=message.token_count,
            )
            db.add(db_message)
            await db.flush()
            await db.refresh(db_message)
            await db.commit()

            message.id = db_message.id
            return message

    async def get_messages(
        self, session_id: str, limit: Optional[int] = None
    ) -> List[Message]:
        """Get messages for a session."""
        async with get_db() as db:
            query = (
                select(MessageDB)
                .where(MessageDB.session_id == session_id)
                .order_by(MessageDB.timestamp.asc())
            )

            if limit is not None:
                query = query.limit(limit)

            result = await db.execute(query)
            db_messages = result.scalars().all()

            return [
                Message(
                    id=m.id,
                    session_id=m.session_id,
                    role=MessageRole(m.role),
                    content=m.content,
                    timestamp=m.timestamp,
                    token_count=m.token_count,
                )
                for m in db_messages
            ]

    async def add_audit_log(self, audit_log: AuditLog) -> AuditLog:
        """Add an audit log entry."""
        async with get_db() as db:
            db_audit = AuditLogDB(
                user_id=audit_log.user_id,
                action=audit_log.action,
                workspace=audit_log.workspace,
                timestamp=audit_log.timestamp,
                details=audit_log.details,
                success=audit_log.success,
            )
            db.add(db_audit)
            await db.flush()
            await db.refresh(db_audit)
            await db.commit()

            audit_log.id = db_audit.id
            return audit_log

    async def get_audit_logs(
        self, user_id: Optional[int] = None, limit: int = 100
    ) -> List[AuditLog]:
        """Get audit logs."""
        async with get_db() as db:
            query = select(AuditLogDB).order_by(AuditLogDB.timestamp.desc())

            if user_id is not None:
                query = query.where(AuditLogDB.user_id == user_id)

            query = query.limit(limit)

            result = await db.execute(query)
            db_logs = result.scalars().all()

            return [
                AuditLog(
                    id=log.id,
                    user_id=log.user_id,
                    action=log.action,
                    workspace=log.workspace,
                    timestamp=log.timestamp,
                    details=log.details,
                    success=log.success,
                )
                for log in db_logs
            ]

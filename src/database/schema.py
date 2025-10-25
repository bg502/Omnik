"""Database schema and initialization."""
from datetime import datetime
from sqlalchemy import (
    Column,
    Integer,
    String,
    Text,
    DateTime,
    Boolean,
    Float,
    ForeignKey,
    create_engine,
)
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession, async_sessionmaker
from sqlalchemy.orm import declarative_base, relationship
from contextlib import asynccontextmanager

Base = declarative_base()


class SessionDB(Base):
    """Sessions table."""

    __tablename__ = "sessions"

    id = Column(String, primary_key=True)
    user_id = Column(Integer, nullable=False, index=True)
    workspace_path = Column(Text, nullable=False)
    name = Column(String, nullable=True)
    status = Column(String, nullable=False, default="active", index=True)
    pid = Column(Integer, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    last_activity = Column(DateTime, default=datetime.utcnow)
    token_usage = Column(Integer, default=0)
    cost_usd = Column(Float, default=0.0)

    messages = relationship("MessageDB", back_populates="session", cascade="all, delete-orphan")


class MessageDB(Base):
    """Messages table."""

    __tablename__ = "messages"

    id = Column(Integer, primary_key=True, autoincrement=True)
    session_id = Column(String, ForeignKey("sessions.id", ondelete="CASCADE"), nullable=False, index=True)
    role = Column(String, nullable=False)
    content = Column(Text, nullable=False)
    timestamp = Column(DateTime, default=datetime.utcnow, index=True)
    token_count = Column(Integer, nullable=True)

    session = relationship("SessionDB", back_populates="messages")


class AuditLogDB(Base):
    """Audit logs table."""

    __tablename__ = "audit_logs"

    id = Column(Integer, primary_key=True, autoincrement=True)
    user_id = Column(Integer, nullable=False, index=True)
    action = Column(String, nullable=False)
    workspace = Column(String, nullable=True)
    timestamp = Column(DateTime, default=datetime.utcnow, index=True)
    details = Column(Text, nullable=True)
    success = Column(Boolean, default=True)


# Global engine and session maker
_engine = None
_async_session_maker = None


async def init_db(database_url: str = "sqlite+aiosqlite:///data/omnik.db"):
    """Initialize the database."""
    global _engine, _async_session_maker

    _engine = create_async_engine(
        database_url,
        echo=False,
        future=True,
    )

    _async_session_maker = async_sessionmaker(
        _engine,
        class_=AsyncSession,
        expire_on_commit=False,
    )

    # Create tables
    async with _engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)


@asynccontextmanager
async def get_db():
    """Get database session."""
    if _async_session_maker is None:
        raise RuntimeError("Database not initialized. Call init_db() first.")

    async with _async_session_maker() as session:
        try:
            yield session
            await session.commit()
        except Exception:
            await session.rollback()
            raise
        finally:
            await session.close()

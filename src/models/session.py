from datetime import datetime
from enum import Enum
from typing import Optional
from pydantic import BaseModel, Field


class SessionStatus(str, Enum):
    """Session status enumeration."""
    ACTIVE = "active"
    PAUSED = "paused"
    CRASHED = "crashed"
    TERMINATED = "terminated"


class Session(BaseModel):
    """Session model representing a Claude Code session."""

    id: str = Field(description="Unique session identifier (UUID)")
    user_id: int = Field(description="Telegram user ID")
    workspace_path: str = Field(description="Absolute path to workspace directory")
    name: Optional[str] = Field(default=None, description="User-provided session name")
    status: SessionStatus = Field(default=SessionStatus.ACTIVE)
    pid: Optional[int] = Field(default=None, description="Claude Code subprocess PID")
    created_at: datetime = Field(default_factory=datetime.utcnow)
    last_activity: datetime = Field(default_factory=datetime.utcnow)
    token_usage: int = Field(default=0, description="Total tokens used")
    cost_usd: float = Field(default=0.0, description="Total cost in USD")

    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat(),
        }

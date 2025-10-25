from datetime import datetime
from enum import Enum
from typing import Optional
from pydantic import BaseModel, Field


class MessageRole(str, Enum):
    """Message role enumeration."""
    USER = "user"
    ASSISTANT = "assistant"
    SYSTEM = "system"


class Message(BaseModel):
    """Message model for conversation history."""

    id: Optional[int] = None
    session_id: str = Field(description="Parent session ID")
    role: MessageRole = Field(description="Message sender")
    content: str = Field(description="Message text content")
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    token_count: Optional[int] = Field(default=None, description="Tokens in this message")

    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat(),
        }

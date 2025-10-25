from datetime import datetime
from typing import Optional
from pydantic import BaseModel, Field


class AuditLog(BaseModel):
    """Audit log model for tracking user actions."""

    id: Optional[int] = None
    user_id: int
    action: str = Field(description="Command or action performed")
    workspace: Optional[str] = Field(default=None)
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    details: Optional[str] = Field(default=None, description="Additional context as JSON")
    success: bool = Field(default=True)

    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat(),
        }

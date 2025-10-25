from datetime import datetime
from pydantic import BaseModel, Field


class WorkspaceInfo(BaseModel):
    """Workspace information model."""

    session_id: str
    path: str
    size_bytes: int = Field(description="Total workspace size")
    file_count: int = Field(description="Number of files in workspace")
    last_modified: datetime

    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat(),
        }

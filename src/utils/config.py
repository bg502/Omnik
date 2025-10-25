"""Configuration management."""
import os
from pathlib import Path
from typing import Optional
from pydantic import BaseModel, Field


class Config(BaseModel):
    """Application configuration."""

    # Telegram settings
    telegram_bot_token: str = Field(description="Telegram bot token")
    authorized_user_id: Optional[int] = Field(
        default=None, description="Authorized Telegram user ID"
    )

    # Anthropic settings (optional - can use existing Pro account auth)
    anthropic_api_key: Optional[str] = Field(
        default=None, description="Anthropic API key (optional if using Pro account)"
    )

    # Application settings
    log_level: str = Field(default="INFO", description="Logging level")
    max_sessions: int = Field(default=10, description="Maximum concurrent sessions")
    session_timeout_hours: int = Field(default=24, description="Session timeout in hours")
    workspace_base: Path = Field(
        default=Path("/workspace"), description="Base path for workspaces"
    )
    database_url: str = Field(
        default="sqlite+aiosqlite:///data/omnik.db",
        description="Database connection URL",
    )

    # Rate limiting
    rate_limit_requests: int = Field(
        default=60, description="Rate limit requests per minute"
    )

    class Config:
        arbitrary_types_allowed = True


def read_secret(secret_name: str) -> str:
    """Read Docker secret from /run/secrets/ or environment variable."""
    secret_path = Path("/run/secrets") / secret_name

    if secret_path.exists():
        content = secret_path.read_text().strip()
        # Handle case where file contains KEY=value format
        if "=" in content:
            # Extract value after the = sign
            return content.split("=", 1)[1].strip()
        return content
    else:
        # Fallback to environment variable for development
        value = os.getenv(secret_name.upper())
        if value is None:
            raise ValueError(f"Secret {secret_name} not found")
        return value


def load_config() -> Config:
    """Load configuration from environment and secrets."""
    # Read secrets
    telegram_bot_token = read_secret("telegram_bot_token")

    # Read Anthropic API key (optional - can use Pro account auth)
    anthropic_api_key_str = os.getenv("ANTHROPIC_API_KEY", "")
    anthropic_api_key = None
    if anthropic_api_key_str and anthropic_api_key_str.strip():
        anthropic_api_key = anthropic_api_key_str.strip()

    # Read other config from environment
    authorized_user_id_str = os.getenv("AUTHORIZED_USER_ID", "")
    authorized_user_id = None
    if authorized_user_id_str and authorized_user_id_str.strip():
        try:
            authorized_user_id = int(authorized_user_id_str.strip())
        except ValueError:
            pass

    return Config(
        telegram_bot_token=telegram_bot_token,
        anthropic_api_key=anthropic_api_key,
        authorized_user_id=authorized_user_id,
        log_level=os.getenv("LOG_LEVEL", "INFO"),
        max_sessions=int(os.getenv("MAX_SESSIONS", "10")),
        session_timeout_hours=int(os.getenv("SESSION_TIMEOUT_HOURS", "24")),
        workspace_base=Path(os.getenv("WORKSPACE_BASE", "/workspace")),
        database_url=os.getenv(
            "DATABASE_URL", "sqlite+aiosqlite:///data/omnik.db"
        ),
        rate_limit_requests=int(os.getenv("RATE_LIMIT_REQUESTS", "60")),
    )

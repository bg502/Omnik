"""Logging setup using structlog."""
import logging
import sys
from pathlib import Path
import structlog


def setup_logging(log_level: str = "INFO", log_dir: Path = None):
    """Configure structured logging."""
    # Convert log level string to logging constant
    level = getattr(logging, log_level.upper(), logging.INFO)

    # Configure standard logging
    logging.basicConfig(
        format="%(message)s",
        stream=sys.stdout,
        level=level,
    )

    # Configure structlog with cleaner output
    structlog.configure(
        processors=[
            structlog.contextvars.merge_contextvars,
            structlog.processors.add_log_level,
            # Only show stack info for errors and above
            structlog.processors.StackInfoRenderer() if level <= logging.ERROR else lambda _, __, event_dict: event_dict,
            structlog.dev.set_exc_info,
            structlog.processors.TimeStamper(fmt="%H:%M:%S"),  # Shorter timestamp
            structlog.dev.ConsoleRenderer(
                colors=True,
                # More compact output
                pad_event=25,
                exception_formatter=structlog.dev.plain_traceback,
            )
            if sys.stdout.isatty()
            else structlog.processors.JSONRenderer(),
        ],
        wrapper_class=structlog.make_filtering_bound_logger(level),
        context_class=dict,
        logger_factory=structlog.PrintLoggerFactory(),
        cache_logger_on_first_use=True,
    )

    # Suppress overly verbose libraries
    logging.getLogger("telegram").setLevel(logging.WARNING)
    logging.getLogger("telegram.ext").setLevel(logging.WARNING)
    logging.getLogger("httpx").setLevel(logging.WARNING)
    logging.getLogger("httpcore").setLevel(logging.WARNING)
    logging.getLogger("urllib3").setLevel(logging.WARNING)
    logging.getLogger("asyncio").setLevel(logging.WARNING)
    logging.getLogger("aiosqlite").setLevel(logging.WARNING)
    logging.getLogger("sqlite3").setLevel(logging.WARNING)
    logging.getLogger("sqlalchemy").setLevel(logging.WARNING)

    logger = structlog.get_logger()
    logger.info("Logging configured", log_level=log_level)

    return logger

"""Main entry point for omnik bot."""
import asyncio
import signal
from pathlib import Path
import structlog

from telegram.ext import (
    Application,
    CommandHandler,
    MessageHandler,
    CallbackQueryHandler,
    filters,
)

from .utils import load_config, setup_logging
from .database import init_db, DatabaseManager
from .bot import SessionManager
from .bot.handlers import BotHandlers

logger = structlog.get_logger()


class OmnikBot:
    """Main bot application."""

    def __init__(self):
        self.config = load_config()
        self.db: DatabaseManager = None
        self.session_manager: SessionManager = None
        self.handlers: BotHandlers = None
        self.app: Application = None
        self._shutdown_event = asyncio.Event()

    async def initialize(self):
        """Initialize all components."""
        logger.info("Initializing omnik bot")

        # Setup logging
        setup_logging(self.config.log_level)

        # Initialize database
        logger.info("Initializing database", url=self.config.database_url)
        await init_db(self.config.database_url)
        self.db = DatabaseManager()

        # Ensure workspace base exists
        self.config.workspace_base.mkdir(parents=True, exist_ok=True)

        # Initialize session manager
        logger.info("Initializing session manager")
        self.session_manager = SessionManager(
            db=self.db,
            workspace_base=self.config.workspace_base,
            anthropic_key=self.config.anthropic_api_key,
        )

        # Initialize bot handlers
        self.handlers = BotHandlers(
            session_manager=self.session_manager,
            db=self.db,
            authorized_user_id=self.config.authorized_user_id,
        )

        # Build Telegram application
        logger.info("Building Telegram application")
        self.app = (
            Application.builder()
            .token(self.config.telegram_bot_token)
            .build()
        )

        # Register command handlers
        self.register_handlers()

        logger.info("Initialization complete")

    def register_handlers(self):
        """Register all command and message handlers."""
        # Wrap handlers with authorization
        authorized = self.handlers.authorized_only

        # Command handlers
        self.app.add_handler(
            CommandHandler("start", authorized(self.handlers.start_command))
        )
        self.app.add_handler(
            CommandHandler("help", authorized(self.handlers.help_command))
        )
        self.app.add_handler(
            CommandHandler("new", authorized(self.handlers.new_session_command))
        )
        self.app.add_handler(
            CommandHandler("list", authorized(self.handlers.list_sessions_command))
        )
        self.app.add_handler(
            CommandHandler("switch", authorized(self.handlers.switch_session_command))
        )
        self.app.add_handler(
            CommandHandler("status", authorized(self.handlers.status_command))
        )
        self.app.add_handler(
            CommandHandler("kill", authorized(self.handlers.kill_session_command))
        )
        self.app.add_handler(
            CommandHandler("restart", authorized(self.handlers.restart_session_command))
        )
        self.app.add_handler(
            CommandHandler("pwd", authorized(self.handlers.pwd_command))
        )
        self.app.add_handler(
            CommandHandler("ls", authorized(self.handlers.ls_command))
        )

        # Message handlers
        self.app.add_handler(
            MessageHandler(
                filters.TEXT & ~filters.COMMAND,
                authorized(self.handlers.message_handler),
            )
        )

        # File upload handler
        self.app.add_handler(
            MessageHandler(
                filters.Document.ALL,
                authorized(self.handlers.file_handler),
            )
        )

        # Callback query handler for interactive buttons
        self.app.add_handler(
            CallbackQueryHandler(
                authorized(self.handlers.button_callback_handler),
                pattern="^prompt:"
            )
        )

        logger.info("Handlers registered")

    async def run(self):
        """Run the bot."""
        logger.info("Starting omnik bot")

        # Initialize components
        await self.initialize()

        # Setup signal handlers
        loop = asyncio.get_event_loop()

        def signal_handler(sig):
            logger.info("Received signal, shutting down", signal=sig)
            self._shutdown_event.set()

        for sig in (signal.SIGTERM, signal.SIGINT):
            loop.add_signal_handler(sig, lambda s=sig: signal_handler(s))

        # Start polling
        async with self.app:
            await self.app.initialize()
            await self.app.start()
            await self.app.updater.start_polling()
            logger.info("Bot started successfully - now polling for updates")

            # Run until shutdown signal
            await self._shutdown_event.wait()

            # Shutdown
            logger.info("Shutting down bot")
            await self.shutdown()

    async def shutdown(self):
        """Gracefully shutdown the bot."""
        logger.info("Starting graceful shutdown")

        # Terminate all active sessions
        if self.session_manager:
            for session_id in list(self.session_manager.sessions.keys()):
                try:
                    logger.info("Terminating session", session_id=session_id)
                    await self.session_manager.terminate_session(session_id)
                except Exception as e:
                    logger.error(
                        "Error terminating session",
                        session_id=session_id,
                        error=str(e),
                    )

        # Stop the application
        if self.app:
            await self.app.updater.stop()
            await self.app.stop()
            await self.app.shutdown()

        logger.info("Shutdown complete")


async def main():
    """Main entry point."""
    bot = OmnikBot()
    await bot.run()


if __name__ == "__main__":
    asyncio.run(main())

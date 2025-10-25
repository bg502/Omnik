"""Telegram bot command handlers."""
import asyncio
from pathlib import Path
from typing import Optional
from datetime import datetime
import structlog

from telegram import Update, InlineKeyboardButton, InlineKeyboardMarkup
from telegram.ext import ContextTypes
from telegram.constants import ChatAction

from ..models import Message, MessageRole, AuditLog
from .session_manager import SessionManager
from .prompt_parser import parse_prompt, format_prompt_message
from ..database import DatabaseManager

logger = structlog.get_logger()


class BotHandlers:
    """Telegram bot command handlers."""

    def __init__(
        self,
        session_manager: SessionManager,
        db: DatabaseManager,
        authorized_user_id: Optional[int] = None,
    ):
        self.session_manager = session_manager
        self.db = db
        self.authorized_user_id = authorized_user_id
        self.pending_prompts = {}  # message_id -> session_id mapping

    def authorized_only(self, handler):
        """Decorator to check authorization."""

        async def wrapper(update: Update, context: ContextTypes.DEFAULT_TYPE):
            user_id = update.effective_user.id

            if self.authorized_user_id and user_id != self.authorized_user_id:
                await update.message.reply_text("❌ Unauthorized")
                await self.db.add_audit_log(
                    AuditLog(
                        user_id=user_id,
                        action="unauthorized_access",
                        success=False,
                    )
                )
                return

            return await handler(update, context)

        return wrapper

    async def start_command(self, update: Update, context: ContextTypes.DEFAULT_TYPE):
        """Handle /start command."""
        user_id = update.effective_user.id

        await self.db.add_audit_log(
            AuditLog(user_id=user_id, action="start_command")
        )

        await update.message.reply_text(
            "👋 Welcome to omnik - Claude Code on Telegram\n\n"
            "Commands:\n"
            "/new [name] - Create new session\n"
            "/list - Show all sessions\n"
            "/switch <id> - Switch active session\n"
            "/status - Show current session status\n"
            "/kill - Terminate active session\n"
            "/restart - Restart active session\n"
            "/help - Full command reference\n\n"
            "Send a message to interact with Claude Code!"
        )

    async def help_command(self, update: Update, context: ContextTypes.DEFAULT_TYPE):
        """Handle /help command."""
        await update.message.reply_text(
            "📖 omnik Command Reference\n\n"
            "Session Management:\n"
            "/new [name] - Create new Claude Code session\n"
            "/list - Show all your sessions\n"
            "/switch <session-id> - Change active session\n"
            "/kill - Terminate active session\n"
            "/restart - Restart Claude Code process\n\n"
            "Session Info:\n"
            "/status - Current session details\n"
            "/pwd - Show current working directory\n"
            "/ls [path] - List files in directory\n\n"
            "File Operations:\n"
            "Send file - Upload to workspace\n"
            "/download <path> - Download file from workspace\n\n"
            "Just send a message to chat with Claude Code!"
        )

    async def new_session_command(
        self, update: Update, context: ContextTypes.DEFAULT_TYPE
    ):
        """Handle /new [name] command."""
        user_id = update.effective_user.id
        name = " ".join(context.args) if context.args else None

        await self.db.add_audit_log(
            AuditLog(user_id=user_id, action="new_session", details=name)
        )

        await update.message.reply_text("⏳ Creating new session...")

        try:
            session = await self.session_manager.create_session(user_id, name)

            await update.message.reply_text(
                f"✅ Created session `{session.id[:8]}`\n"
                f"Name: {session.name or 'Unnamed'}\n"
                f"Workspace: `{session.workspace_path}`\n\n"
                "Send a message to start coding!",
                parse_mode="Markdown",
            )

        except Exception as e:
            logger.error("Failed to create session", error=str(e))
            await update.message.reply_text(
                f"❌ Failed to create session: {str(e)}"
            )

    async def list_sessions_command(
        self, update: Update, context: ContextTypes.DEFAULT_TYPE
    ):
        """Handle /list command."""
        user_id = update.effective_user.id

        await self.db.add_audit_log(
            AuditLog(user_id=user_id, action="list_sessions")
        )

        sessions = await self.session_manager.list_sessions(user_id)

        if not sessions:
            await update.message.reply_text(
                "No sessions found. Create one with /new"
            )
            return

        active_session = await self.session_manager.get_active_session(user_id)
        active_id = active_session.id if active_session else None

        lines = ["📋 Your Sessions:\n"]
        for session in sessions:
            is_active = "🔵" if session.id == active_id else "⚪"
            uptime = datetime.utcnow() - session.created_at
            lines.append(
                f"{is_active} `{session.id[:8]}` - {session.name or 'Unnamed'}\n"
                f"   Status: {session.status.value}\n"
                f"   Uptime: {uptime.days}d {uptime.seconds // 3600}h\n"
            )

        await update.message.reply_text("".join(lines), parse_mode="Markdown")

    async def switch_session_command(
        self, update: Update, context: ContextTypes.DEFAULT_TYPE
    ):
        """Handle /switch <session-id> command."""
        user_id = update.effective_user.id

        if not context.args:
            await update.message.reply_text(
                "Usage: /switch <session-id>\n"
                "Use /list to see available sessions"
            )
            return

        session_id_prefix = context.args[0]

        # Find session by prefix
        sessions = await self.session_manager.list_sessions(user_id)
        matching = [s for s in sessions if s.id.startswith(session_id_prefix)]

        if not matching:
            await update.message.reply_text(
                f"❌ No session found matching `{session_id_prefix}`",
                parse_mode="Markdown",
            )
            return

        if len(matching) > 1:
            await update.message.reply_text(
                "❌ Multiple sessions match. Please be more specific."
            )
            return

        session = matching[0]

        await self.session_manager.set_active_session(user_id, session.id)
        await self.db.add_audit_log(
            AuditLog(
                user_id=user_id,
                action="switch_session",
                workspace=session.workspace_path,
            )
        )

        await update.message.reply_text(
            f"✅ Switched to session `{session.id[:8]}`\n"
            f"Name: {session.name or 'Unnamed'}",
            parse_mode="Markdown",
        )

    async def status_command(
        self, update: Update, context: ContextTypes.DEFAULT_TYPE
    ):
        """Handle /status command."""
        user_id = update.effective_user.id

        session = await self.session_manager.get_active_session(user_id)

        if not session:
            await update.message.reply_text(
                "No active session. Create one with /new"
            )
            return

        uptime = datetime.utcnow() - session.created_at

        await update.message.reply_text(
            f"📊 Session Status\n\n"
            f"ID: `{session.id[:8]}`\n"
            f"Name: {session.name or 'Unnamed'}\n"
            f"Status: {session.status.value}\n"
            f"PID: {session.pid}\n"
            f"Workspace: `{session.workspace_path}`\n"
            f"Uptime: {uptime.days}d {uptime.seconds // 3600}h {(uptime.seconds % 3600) // 60}m\n"
            f"Tokens Used: {session.token_usage}\n"
            f"Cost: ${session.cost_usd:.4f}",
            parse_mode="Markdown",
        )

    async def kill_session_command(
        self, update: Update, context: ContextTypes.DEFAULT_TYPE
    ):
        """Handle /kill command."""
        user_id = update.effective_user.id

        session = await self.session_manager.get_active_session(user_id)

        if not session:
            await update.message.reply_text(
                "No active session to kill."
            )
            return

        await update.message.reply_text("⏳ Terminating session...")

        try:
            await self.session_manager.terminate_session(session.id)
            await self.db.add_audit_log(
                AuditLog(
                    user_id=user_id,
                    action="kill_session",
                    workspace=session.workspace_path,
                )
            )

            await update.message.reply_text(
                f"✅ Session `{session.id[:8]}` terminated",
                parse_mode="Markdown",
            )

        except Exception as e:
            logger.error("Failed to kill session", error=str(e))
            await update.message.reply_text(
                f"❌ Failed to terminate session: {str(e)}"
            )

    async def restart_session_command(
        self, update: Update, context: ContextTypes.DEFAULT_TYPE
    ):
        """Handle /restart command."""
        user_id = update.effective_user.id

        session = await self.session_manager.get_active_session(user_id)

        if not session:
            await update.message.reply_text(
                "No active session to restart."
            )
            return

        await update.message.reply_text("⏳ Restarting session...")

        try:
            await self.session_manager.restart_session(session.id)
            await self.db.add_audit_log(
                AuditLog(
                    user_id=user_id,
                    action="restart_session",
                    workspace=session.workspace_path,
                )
            )

            await update.message.reply_text(
                f"✅ Session `{session.id[:8]}` restarted",
                parse_mode="Markdown",
            )

        except Exception as e:
            logger.error("Failed to restart session", error=str(e))
            await update.message.reply_text(
                f"❌ Failed to restart session: {str(e)}"
            )

    async def pwd_command(self, update: Update, context: ContextTypes.DEFAULT_TYPE):
        """Handle /pwd command."""
        user_id = update.effective_user.id

        session = await self.session_manager.get_active_session(user_id)

        if not session:
            await update.message.reply_text(
                "No active session. Create one with /new"
            )
            return

        await update.message.reply_text(
            f"📁 Current workspace:\n`{session.workspace_path}`",
            parse_mode="Markdown",
        )

    async def ls_command(self, update: Update, context: ContextTypes.DEFAULT_TYPE):
        """Handle /ls [path] command."""
        user_id = update.effective_user.id

        session = await self.session_manager.get_active_session(user_id)

        if not session:
            await update.message.reply_text(
                "No active session. Create one with /new"
            )
            return

        workspace = Path(session.workspace_path)
        subpath = context.args[0] if context.args else "."
        target = workspace / subpath

        try:
            if not target.exists():
                await update.message.reply_text(f"❌ Path not found: {subpath}")
                return

            if target.is_file():
                await update.message.reply_text(
                    f"📄 {subpath} (file, {target.stat().st_size} bytes)"
                )
                return

            items = sorted(target.iterdir(), key=lambda x: (not x.is_dir(), x.name))
            lines = [f"📁 {subpath}:\n"]

            for item in items[:50]:  # Limit to 50 items
                icon = "📁" if item.is_dir() else "📄"
                size = f" ({item.stat().st_size} bytes)" if item.is_file() else ""
                lines.append(f"{icon} {item.name}{size}\n")

            if len(items) > 50:
                lines.append(f"\n... and {len(items) - 50} more items")

            await update.message.reply_text("".join(lines))

        except Exception as e:
            logger.error("Failed to list directory", error=str(e))
            await update.message.reply_text(f"❌ Error: {str(e)}")

    async def message_handler(
        self, update: Update, context: ContextTypes.DEFAULT_TYPE
    ):
        """Handle non-command messages - forward to Claude Code."""
        user_id = update.effective_user.id
        message_text = update.message.text

        # Get active session
        session = await self.session_manager.get_active_session(user_id)

        if not session:
            await update.message.reply_text(
                "No active session. Create one with /new"
            )
            return

        # Save user message
        await self.db.add_message(
            Message(
                session_id=session.id,
                role=MessageRole.USER,
                content=message_text,
            )
        )

        # Show typing indicator
        await update.message.chat.send_action(ChatAction.TYPING)

        # Send to Claude Code and stream response
        sent_message = await update.message.reply_text("🤔 Processing...")

        buffer = ""
        last_edit = 0
        last_content = ""  # Track last sent content to avoid duplicate edits

        try:
            async for chunk in self.session_manager.send_message(
                session.id, message_text
            ):
                buffer += chunk
                current_time = asyncio.get_event_loop().time()

                # Update message every 0.5 seconds if content changed
                if current_time - last_edit > 0.5 and len(buffer) > 0:
                    try:
                        # Limit to Telegram's 4096 character limit
                        display_text = buffer[:4000]
                        if len(buffer) > 4000:
                            display_text += "\n\n... (truncated)"

                        # Only edit if content actually changed
                        if display_text != last_content:
                            await sent_message.edit_text(display_text)
                            last_content = display_text
                            last_edit = current_time
                    except Exception:
                        # Silently ignore edit errors (rate limiting, no change, etc.)
                        pass

            # Final edit with complete response
            display_text = buffer[:4000] if buffer else "✅ Done (no output)"
            if len(buffer) > 4000:
                display_text = buffer[:4000] + "\n\n... (truncated)"

            # Classify the response type
            from .response_classifier import classify_response, requires_user_action, ResponseType

            response_type = classify_response(buffer) if buffer else ResponseType.INFO
            logger.info(
                "Response classified",
                response_type=response_type.value,
                needs_action=requires_user_action(response_type),
                buffer_length=len(buffer) if buffer else 0
            )

            # Check if response contains an interactive prompt
            parsed_prompt = parse_prompt(buffer) if buffer else None

            if parsed_prompt:
                logger.info("Parsed prompt successfully, creating buttons", num_options=len(parsed_prompt['options']))
                # Create inline keyboard buttons
                keyboard = []
                for option in parsed_prompt['options']:
                    button = InlineKeyboardButton(
                        text=f"{option['number']}. {option['text']}",
                        callback_data=f"prompt:{session.id}:{option['number']}"
                    )
                    keyboard.append([button])

                reply_markup = InlineKeyboardMarkup(keyboard)

                # Format clean message
                clean_msg = format_prompt_message(parsed_prompt)

                try:
                    await sent_message.edit_text(
                        clean_msg,
                        reply_markup=reply_markup,
                        parse_mode="Markdown"
                    )
                    # Store prompt for callback handling
                    self.pending_prompts[sent_message.message_id] = session.id
                    logger.info("Prompt buttons sent successfully", message_id=sent_message.message_id)
                except Exception as e:
                    logger.error("Failed to send prompt buttons", error=str(e), error_type=type(e).__name__)
                    # Fall back to plain text
                    await sent_message.edit_text(display_text)
            else:
                logger.debug("No prompt detected, sending as plain text")
                # Only do final edit if content changed (non-prompt response)
                if display_text != last_content:
                    try:
                        await sent_message.edit_text(display_text)
                    except Exception:
                        pass  # Silently ignore final edit errors

            # Save assistant message
            await self.db.add_message(
                Message(
                    session_id=session.id,
                    role=MessageRole.ASSISTANT,
                    content=buffer,
                )
            )

        except Exception as e:
            logger.error("Error processing message", error=str(e))
            try:
                await sent_message.edit_text(f"❌ Error: {str(e)}")
            except:
                pass  # If edit fails, message already has error info

    async def button_callback_handler(
        self, update: Update, context: ContextTypes.DEFAULT_TYPE
    ):
        """Handle inline button callbacks for Claude Code prompts."""
        query = update.callback_query
        await query.answer()

        # Parse callback data: "prompt:session_id:option_number"
        try:
            _, session_id, option_num = query.data.split(":")
        except ValueError:
            await query.edit_message_text("❌ Invalid button data")
            return

        logger.info(
            "Button clicked",
            session_id=session_id,
            option=option_num,
            user_id=update.effective_user.id
        )

        # Send the option number to Claude Code
        try:
            # Just send the number (Claude Code will interpret it)
            await self.session_manager.sessions[session_id].send_input(option_num)

            # Update message to show selection
            await query.edit_message_text(
                f"✅ Selected option {option_num}\n\n⏳ Processing..."
            )

            # Clean up pending prompt
            if query.message.message_id in self.pending_prompts:
                del self.pending_prompts[query.message.message_id]

        except Exception as e:
            logger.error("Error sending button response", error=str(e), session_id=session_id)
            await query.edit_message_text(f"❌ Error: {str(e)}")

    async def file_handler(
        self, update: Update, context: ContextTypes.DEFAULT_TYPE
    ):
        """Handle file uploads."""
        user_id = update.effective_user.id

        session = await self.session_manager.get_active_session(user_id)

        if not session:
            await update.message.reply_text(
                "No active session. Create one with /new"
            )
            return

        document = update.message.document
        if not document:
            return

        # Download file
        file = await document.get_file()
        workspace = Path(session.workspace_path)
        file_path = workspace / document.file_name

        await update.message.reply_text(f"⏳ Uploading {document.file_name}...")

        try:
            await file.download_to_drive(str(file_path))

            await self.db.add_audit_log(
                AuditLog(
                    user_id=user_id,
                    action="file_upload",
                    workspace=session.workspace_path,
                    details=document.file_name,
                )
            )

            await update.message.reply_text(
                f"✅ Uploaded to `{file_path.relative_to(workspace)}`",
                parse_mode="Markdown",
            )

        except Exception as e:
            logger.error("Failed to upload file", error=str(e))
            await update.message.reply_text(f"❌ Upload failed: {str(e)}")

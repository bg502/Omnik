"""Permission approval handling for Claude Code via Telegram."""
import re
from typing import Optional, Dict
import structlog
from telegram import Update, InlineKeyboardButton, InlineKeyboardMarkup
from telegram.ext import ContextTypes, CallbackQueryHandler

logger = structlog.get_logger()


class PermissionHandler:
    """Handles Claude Code permission requests via Telegram."""

    def __init__(self):
        self.pending_permissions: Dict[str, Dict] = {}  # session_id -> permission_data

    def detect_permission_request(self, output: str) -> Optional[Dict]:
        """
        Detect if output contains a permission request from Claude Code.

        Claude Code typically outputs permission requests in a specific format.
        Returns permission data if found, None otherwise.
        """
        # Common permission request patterns
        patterns = [
            r"Allow Claude to (.*)\?",
            r"Permission required: (.*)",
            r"Claude would like to (.*)",
            r"Allow (.*) permission\?",
        ]

        for pattern in patterns:
            match = re.search(pattern, output, re.IGNORECASE)
            if match:
                permission_text = match.group(1)
                return {
                    "text": permission_text,
                    "full_output": output,
                }

        return None

    async def request_permission(
        self,
        update: Update,
        context: ContextTypes.DEFAULT_TYPE,
        session_id: str,
        permission_data: Dict,
    ) -> InlineKeyboardMarkup:
        """
        Send a permission request to the user via Telegram with approve/deny buttons.

        Returns the keyboard markup.
        """
        permission_text = permission_data.get("text", "Unknown permission")

        # Store pending permission
        self.pending_permissions[session_id] = permission_data

        # Create inline keyboard with approve/deny buttons
        keyboard = [
            [
                InlineKeyboardButton("✅ Approve", callback_data=f"perm_approve_{session_id}"),
                InlineKeyboardButton("❌ Deny", callback_data=f"perm_deny_{session_id}"),
            ],
            [
                InlineKeyboardButton("✅ Always Approve", callback_data=f"perm_always_{session_id}"),
            ],
        ]

        return InlineKeyboardMarkup(keyboard)

    async def handle_permission_callback(
        self,
        update: Update,
        context: ContextTypes.DEFAULT_TYPE,
    ) -> Optional[str]:
        """
        Handle permission approval/denial callback from inline buttons.

        Returns the response to send to Claude Code stdin.
        """
        query = update.callback_query
        await query.answer()

        data = query.data

        # Parse callback data
        if data.startswith("perm_approve_"):
            session_id = data.replace("perm_approve_", "")
            response = "y\n"  # Send 'y' to approve
            await query.edit_message_text(
                text=f"✅ Permission approved\n\n{query.message.text}"
            )
            logger.info("Permission approved", session_id=session_id)

        elif data.startswith("perm_deny_"):
            session_id = data.replace("perm_deny_", "")
            response = "n\n"  # Send 'n' to deny
            await query.edit_message_text(
                text=f"❌ Permission denied\n\n{query.message.text}"
            )
            logger.info("Permission denied", session_id=session_id)

        elif data.startswith("perm_always_"):
            session_id = data.replace("perm_always_", "")
            response = "a\n"  # Send 'a' to always approve
            await query.edit_message_text(
                text=f"✅ Permission always approved (for this session)\n\n{query.message.text}"
            )
            logger.info("Permission always approved", session_id=session_id)
        else:
            return None

        # Clean up pending permission
        if session_id in self.pending_permissions:
            del self.pending_permissions[session_id]

        return response

    def get_callback_handler(self):
        """Get the CallbackQueryHandler for permission callbacks."""
        return CallbackQueryHandler(
            self.handle_permission_callback,
            pattern="^perm_(approve|deny|always)_.*",
        )

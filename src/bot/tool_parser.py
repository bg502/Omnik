"""Parse and highlight Claude Code tool usage in output."""
import re
from typing import Optional, Tuple
from enum import Enum


class ClaudeTool(Enum):
    """Claude Code tools."""
    BASH = "Bash"
    READ = "Read"
    WRITE = "Write"
    EDIT = "Edit"
    GLOB = "Glob"
    GREP = "Grep"
    TASK = "Task"
    WEB_FETCH = "WebFetch"
    WEB_SEARCH = "WebSearch"
    ASK_USER = "AskUserQuestion"
    TODO_WRITE = "TodoWrite"
    NOTEBOOK_EDIT = "NotebookEdit"
    KILL_SHELL = "KillShell"
    BASH_OUTPUT = "BashOutput"


# Emoji mapping for tool types
TOOL_EMOJI = {
    ClaudeTool.BASH: "🔧",
    ClaudeTool.READ: "📖",
    ClaudeTool.WRITE: "✏️",
    ClaudeTool.EDIT: "📝",
    ClaudeTool.GLOB: "🔍",
    ClaudeTool.GREP: "🔎",
    ClaudeTool.TASK: "🤖",
    ClaudeTool.WEB_FETCH: "🌐",
    ClaudeTool.WEB_SEARCH: "🔍",
    ClaudeTool.ASK_USER: "❓",
    ClaudeTool.TODO_WRITE: "📋",
    ClaudeTool.NOTEBOOK_EDIT: "📓",
    ClaudeTool.KILL_SHELL: "⚠️",
    ClaudeTool.BASH_OUTPUT: "📊",
}


class ToolUsageParser:
    """Parse Claude Code output to detect tool usage."""

    # Pattern to detect tool invocations
    # Claude Code typically outputs in format like:
    # "I'm going to use the <tool> to <description>"
    # "Using <tool> to <description>"
    # "Let me <action> using <tool>"
    TOOL_PATTERNS = {
        ClaudeTool.BASH: [
            r"(?:I'm going to |Let me |I'll )(?:use the )?Bash(?: tool)?",
            r"(?:Running|Executing) (?:the )?(?:bash )?command",
            r"Let me run",
            r"I'll execute",
        ],
        ClaudeTool.READ: [
            r"(?:I'm going to |Let me |I'll )(?:use the )?Read(?: tool)?",
            r"(?:Reading|Let me read) (?:the )?file",
            r"I'll read",
        ],
        ClaudeTool.WRITE: [
            r"(?:I'm going to |Let me |I'll )(?:use the )?Write(?: tool)?",
            r"(?:Writing|Creating) (?:the )?file",
            r"I'll write",
            r"I'm going to create",
        ],
        ClaudeTool.EDIT: [
            r"(?:I'm going to |Let me |I'll )(?:use the )?Edit(?: tool)?",
            r"(?:Editing|Modifying|Updating) (?:the )?file",
            r"I'll edit",
            r"I'll update",
        ],
        ClaudeTool.GLOB: [
            r"(?:I'm going to |Let me |I'll )(?:use the )?Glob(?: tool)?",
            r"(?:Finding|Searching for) files",
            r"I'll search for files",
        ],
        ClaudeTool.GREP: [
            r"(?:I'm going to |Let me |I'll )(?:use the )?Grep(?: tool)?",
            r"(?:Searching|Looking) (?:for|through)",
            r"I'll search (?:for|through)",
        ],
        ClaudeTool.TASK: [
            r"(?:I'm going to |Let me |I'll )(?:use the )?Task(?: tool)?",
            r"(?:Launching|Starting) (?:an? )?agent",
            r"I'll launch",
        ],
        ClaudeTool.WEB_FETCH: [
            r"(?:I'm going to |Let me |I'll )(?:use the )?WebFetch(?: tool)?",
            r"(?:Fetching|Getting) (?:from )?(?:the )?(?:web|URL)",
        ],
        ClaudeTool.WEB_SEARCH: [
            r"(?:I'm going to |Let me |I'll )(?:use the )?WebSearch(?: tool)?",
            r"(?:Searching|Looking up) (?:the )?web",
        ],
        ClaudeTool.TODO_WRITE: [
            r"(?:I'm going to |Let me |I'll )(?:use the )?TodoWrite(?: tool)?",
            r"(?:Creating|Updating) (?:the )?todo list",
            r"(?:Adding|Writing) todos",
        ],
    }

    def __init__(self):
        # Compile all patterns for efficiency
        self.compiled_patterns = {}
        for tool, patterns in self.TOOL_PATTERNS.items():
            self.compiled_patterns[tool] = [
                re.compile(pattern, re.IGNORECASE) for pattern in patterns
            ]

    def detect_tool_usage(self, text: str) -> Optional[ClaudeTool]:
        """
        Detect if text contains a tool usage indication.

        Returns the tool type if detected, None otherwise.
        """
        for tool, patterns in self.compiled_patterns.items():
            for pattern in patterns:
                if pattern.search(text):
                    return tool
        return None

    def format_tool_usage(self, tool: ClaudeTool, description: str = "") -> str:
        """
        Format tool usage with emoji and description.

        Args:
            tool: The detected tool
            description: Optional description of what the tool is doing

        Returns:
            Formatted string like "🔧 Bash: Running command"
        """
        emoji = TOOL_EMOJI.get(tool, "🔹")
        tool_name = tool.value

        if description:
            return f"{emoji} {tool_name}: {description}"
        else:
            return f"{emoji} {tool_name}"

    def extract_tool_description(self, text: str, tool: ClaudeTool) -> str:
        """
        Extract the description of what the tool is doing from the text.

        For example, from "Let me read the file config.py", extract "config.py"
        """
        # Try to extract meaningful description based on tool type
        if tool == ClaudeTool.BASH:
            # Look for command in backticks or quotes
            match = re.search(r'`([^`]+)`', text)
            if match:
                return match.group(1)
            match = re.search(r'"([^"]+)"', text)
            if match:
                return match.group(1)

        elif tool == ClaudeTool.READ:
            # Look for file path
            match = re.search(r'file[s]?\s+([^\s,\.]+)', text, re.IGNORECASE)
            if match:
                return match.group(1)

        elif tool == ClaudeTool.WRITE:
            # Look for file path
            match = re.search(r'(?:file|to)\s+([^\s,\.]+)', text, re.IGNORECASE)
            if match:
                return match.group(1)

        elif tool == ClaudeTool.EDIT:
            # Look for file path
            match = re.search(r'(?:file|in)\s+([^\s,\.]+)', text, re.IGNORECASE)
            if match:
                return match.group(1)

        # Default: return first part of sentence after tool mention
        sentences = text.split('.')
        if sentences:
            return sentences[0].strip()

        return ""

    def highlight_tools_in_output(self, output: str) -> Tuple[str, list]:
        """
        Process output text and highlight tool usage.

        Returns:
            Tuple of (processed_output, list_of_detected_tools)
        """
        lines = output.split('\n')
        processed_lines = []
        detected_tools = []

        for line in lines:
            tool = self.detect_tool_usage(line)
            if tool:
                detected_tools.append(tool)
                description = self.extract_tool_description(line, tool)
                formatted = self.format_tool_usage(tool, description)
                # Add formatted tool usage before the original line
                processed_lines.append(f"┌─ {formatted}")
                processed_lines.append(f"└─ {line}")
            else:
                processed_lines.append(line)

        return '\n'.join(processed_lines), detected_tools

    def get_tool_summary(self, tools: list) -> str:
        """
        Create a summary of tools used.

        Args:
            tools: List of ClaudeTool enums

        Returns:
            Formatted string like "🔧 Bash (2x), 📖 Read (1x)"
        """
        if not tools:
            return ""

        # Count tool occurrences
        tool_counts = {}
        for tool in tools:
            tool_counts[tool] = tool_counts.get(tool, 0) + 1

        # Format as "emoji Tool (count)"
        parts = []
        for tool, count in tool_counts.items():
            emoji = TOOL_EMOJI.get(tool, "🔹")
            if count > 1:
                parts.append(f"{emoji} {tool.value} ({count}x)")
            else:
                parts.append(f"{emoji} {tool.value}")

        return ", ".join(parts)

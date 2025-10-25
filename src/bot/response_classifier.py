"""Classifier to determine if Claude Code response requires user action."""
import re
from typing import Literal, Optional, Dict, List
from enum import Enum
import structlog

logger = structlog.get_logger()


class ResponseType(str, Enum):
    """Types of responses from Claude Code."""
    PROMPT = "prompt"  # Requires user interaction (numbered options)
    CONFIRMATION = "confirmation"  # Yes/No question
    INFO = "info"  # Informational message, no action needed
    ERROR = "error"  # Error message
    WORKING = "working"  # Claude is working on something
    COMPLETE = "complete"  # Task completed


def classify_response(text: str) -> ResponseType:
    """
    Classify a Claude Code response to determine its type.

    Args:
        text: The response text from Claude Code

    Returns:
        ResponseType enum indicating the classification
    """
    # Strip ANSI codes first
    ansi_escape = re.compile(r'\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])')
    clean_text = ansi_escape.sub('', text).strip()

    if not clean_text:
        logger.debug("Empty response received")
        return ResponseType.INFO

    logger.debug(
        "Classifying response",
        length=len(clean_text),
        preview=clean_text[:200]
    )

    # Check for numbered options (interactive prompt)
    if _has_numbered_options(clean_text):
        logger.info("Classified as PROMPT (has numbered options)")
        return ResponseType.PROMPT

    # Check for Yes/No confirmation patterns
    if _is_confirmation(clean_text):
        logger.info("Classified as CONFIRMATION")
        return ResponseType.CONFIRMATION

    # Check for error patterns
    if _is_error(clean_text):
        logger.info("Classified as ERROR")
        return ResponseType.ERROR

    # Check for working/processing indicators
    if _is_working(clean_text):
        logger.info("Classified as WORKING")
        return ResponseType.WORKING

    # Check for completion indicators
    if _is_complete(clean_text):
        logger.info("Classified as COMPLETE")
        return ResponseType.COMPLETE

    # Default to info
    logger.debug("Classified as INFO (default)")
    return ResponseType.INFO


def _has_numbered_options(text: str) -> bool:
    """Check if text contains numbered options (1. 2. etc.)."""
    # Look for patterns like:
    # "❯ 1. Yes, proceed"
    # "  1. Option one"
    # "│ 1. Choice"
    option_pattern = re.compile(
        r'^\s*[❯│\s]*\d+\.\s+\w+',  # Number followed by period and text
        re.MULTILINE
    )

    matches = option_pattern.findall(text)

    # Need at least 2 options to be a real prompt
    if len(matches) >= 2:
        logger.debug(f"Found {len(matches)} numbered options", options=matches[:5])
        return True

    return False


def _is_confirmation(text: str) -> bool:
    """Check if text is asking for Yes/No confirmation."""
    confirmation_patterns = [
        r'\b(yes|no)\b',
        r'\b(y/n)\b',
        r'\b(confirm|proceed|continue)\b',
        r'\?$',  # Ends with question mark
    ]

    text_lower = text.lower()

    # Must have question indicator
    has_question = any(re.search(pattern, text_lower) for pattern in confirmation_patterns)

    # Should be relatively short (not a long explanation)
    is_short = len(text) < 500

    return has_question and is_short


def _is_error(text: str) -> bool:
    """Check if text contains error indicators."""
    error_indicators = [
        'error', 'failed', 'exception', 'cannot', 'unable',
        'invalid', 'not found', 'denied', 'permission',
        '❌', '⚠️', 'warning'
    ]

    text_lower = text.lower()

    return any(indicator in text_lower for indicator in error_indicators)


def _is_working(text: str) -> bool:
    """Check if Claude is indicating it's working on something."""
    working_indicators = [
        'working on', 'processing', 'analyzing', 'searching',
        'reading', 'writing', 'creating', 'updating',
        'let me', 'i will', 'i am', "i'm",
        '⏳', '🔄', '⚙️'
    ]

    text_lower = text.lower()

    # Should be at the beginning of response
    first_100 = text_lower[:100]

    return any(indicator in first_100 for indicator in working_indicators)


def _is_complete(text: str) -> bool:
    """Check if text indicates task completion."""
    completion_indicators = [
        'done', 'completed', 'finished', 'success',
        'created successfully', 'updated successfully',
        '✅', '✓', 'ready'
    ]

    text_lower = text.lower()

    return any(indicator in text_lower for indicator in completion_indicators)


def requires_user_action(response_type: ResponseType) -> bool:
    """
    Determine if a response type requires user action.

    Args:
        response_type: The classified response type

    Returns:
        True if user needs to respond, False otherwise
    """
    return response_type in (ResponseType.PROMPT, ResponseType.CONFIRMATION)


def extract_prompt_details(text: str) -> Optional[Dict]:
    """
    Extract detailed information from a prompt response.

    Args:
        text: The prompt text

    Returns:
        Dict with 'options' (list), 'question' (str), 'default' (optional str)
        or None if not a valid prompt
    """
    from .prompt_parser import parse_prompt
    return parse_prompt(text)


def get_response_summary(text: str, response_type: ResponseType) -> str:
    """
    Get a short summary of the response for logging/display.

    Args:
        text: The full response text
        response_type: The classified type

    Returns:
        Short summary string
    """
    # Strip ANSI
    ansi_escape = re.compile(r'\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])')
    clean_text = ansi_escape.sub('', text).strip()

    # Get first meaningful line
    lines = [line.strip() for line in clean_text.split('\n') if line.strip()]

    if not lines:
        return "(empty response)"

    first_line = lines[0][:100]

    if response_type == ResponseType.PROMPT:
        # Count options
        option_count = len(re.findall(r'^\s*[❯│\s]*\d+\.', clean_text, re.MULTILINE))
        return f"Interactive prompt with {option_count} options"

    elif response_type == ResponseType.ERROR:
        return f"Error: {first_line}"

    elif response_type == ResponseType.COMPLETE:
        return f"Completed: {first_line}"

    else:
        return first_line


# Example usage and testing
if __name__ == "__main__":
    # Test cases
    test_trust_prompt = """
╭──────────────────────────────────────────────────────────────────────────────╮
│ Do you trust the files in this folder?                                       │
│                                                                              │
│ /workspace/8afdab13-0fce-4365-ace8-226127b13f9a                              │
│                                                                              │
│ Claude Code may read, write, or execute files contained in this directory.   │
│ This can pose security risks, so only use files from trusted sources.        │
│                                                                              │
│ Learn more ( https://docs.claude.com/s/claude-code-security )                │
│                                                                              │
│ ❯ 1. Yes, proceed                                                            │
│   2. No, exit                                                                │
│                                                                              │
╰─────────────────────────────────────────────────────────────────────────────╯
   Enter to confirm · Esc to exit
    """

    result = classify_response(test_trust_prompt)
    print(f"Trust prompt classified as: {result}")
    print(f"Requires action: {requires_user_action(result)}")
    print(f"Summary: {get_response_summary(test_trust_prompt, result)}")

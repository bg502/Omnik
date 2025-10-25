"""Parser for Claude Code interactive prompts."""
import re
from typing import Optional, List, Dict
import structlog

logger = structlog.get_logger()


def parse_prompt(text: str) -> Optional[Dict]:
    """
    Parse Claude Code interactive prompts and extract options.

    Returns:
        Dict with 'question', 'options' (list of dicts with 'number' and 'text'),
        and 'raw_text' if a prompt is detected, None otherwise.
    """
    # First, strip any remaining ANSI escape codes that might have been missed
    ansi_escape = re.compile(r'\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])')
    clean_text = ansi_escape.sub('', text)

    # Log the text being parsed for debugging
    logger.debug(
        "Parsing potential prompt",
        text_length=len(text),
        clean_length=len(clean_text),
        text_preview=clean_text[:300] if clean_text else ""
    )

    # Pattern to detect numbered options like "❯ 1. Yes, proceed" or "  1. Yes, proceed"
    # More flexible to handle various formats
    option_pattern = re.compile(r'^\s*[❯│\s]*(\d+)\.\s+(.+?)(?:\s*│)?\s*$', re.MULTILINE)

    # Check if this looks like a prompt - more flexible matching
    prompt_indicators = [
        'Yes, proceed', 'No, exit', 'Enter to confirm', 'Esc to exit',
        'Do you trust', 'trust the files', 'security risks',
        'Select an option', 'Choose:', 'continue?'
    ]

    has_indicator = any(phrase.lower() in clean_text.lower() for phrase in prompt_indicators)

    if not has_indicator:
        logger.debug("No prompt indicators found in text")
        return None

    logger.debug("Found prompt indicator, attempting to extract options")

    # Extract options
    options = []
    for match in option_pattern.finditer(clean_text):
        option_num = match.group(1)
        option_text = match.group(2).strip()

        # Skip if option text is empty or too short
        if len(option_text) < 2:
            continue

        options.append({
            'number': option_num,
            'text': option_text,
            'callback_data': f'opt_{option_num}'
        })

    if not options:
        logger.warning(
            "Prompt indicators found but no options extracted",
            clean_text_sample=clean_text[:500]
        )
        return None

    logger.info(
        "Successfully parsed prompt",
        num_options=len(options),
        options=[f"{o['number']}. {o['text']}" for o in options]
    )

    # Extract question (usually before the options)
    lines = clean_text.split('\n')
    question_lines = []

    for line in lines:
        # Skip box drawing, option lines, and instruction lines
        clean_line = re.sub(r'[│├─╭╮╰╯└┘┌┐╯]', '', line).strip()

        # Skip empty lines, option lines, and footer instructions
        if (clean_line and
            not re.match(r'^\s*[❯\s]*\d+\.', clean_line) and
            'Enter to confirm' not in clean_line and
            'Esc to exit' not in clean_line and
            len(clean_line) > 3):
            question_lines.append(clean_line)

    # Take first few meaningful lines, but limit to avoid too long questions
    question = '\n'.join(question_lines[:6]).strip()

    # If question is too short, use a generic prompt
    if len(question) < 10:
        question = "Please select an option:"

    return {
        'question': question,
        'options': options,
        'raw_text': text
    }


def format_prompt_message(parsed_prompt: Dict) -> str:
    """Format parsed prompt into a clean message."""
    msg = f"**{parsed_prompt['question']}**\n\n"
    msg += "_Select an option below:_"
    return msg

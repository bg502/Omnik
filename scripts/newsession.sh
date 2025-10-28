#!/bin/bash
# Create a new session
# Usage: ./newsession.sh <name> [description]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/config.sh"

if [ -z "$1" ]; then
    echo "Usage: $0 <session_name> [description]"
    echo ""
    echo "Examples:"
    echo "  $0 automation"
    echo "  $0 project1 'Working on project 1'"
    exit 1
fi

SESSION_NAME="$1"
DESCRIPTION="${2:-}"

if [ -n "$DESCRIPTION" ]; then
    COMMAND="/newsession $SESSION_NAME $DESCRIPTION"
else
    COMMAND="/newsession $SESSION_NAME"
fi

echo "Creating session '$SESSION_NAME' for chat $CHAT_ID..."

curl -X POST "https://api.telegram.org/bot${OMNI_AUTH_API_TOKEN}/sendMessage" \
  -H "Content-Type: application/json" \
  -d "{\"chat_id\": \"${CHAT_ID}\", \"text\": \"$COMMAND\"}"

echo ""

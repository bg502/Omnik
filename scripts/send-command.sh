#!/bin/bash
# Send a command to the bot via Telegram API
# Usage: ./send-command.sh "/status"
#        ./send-command.sh "Hello, what's the current date?"

# Load configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/config.sh"

if [ -z "$1" ]; then
    echo "Usage: $0 <command_or_message>"
    echo ""
    echo "Examples:"
    echo "  $0 '/status'"
    echo "  $0 '/newsession test Test session'"
    echo "  $0 'Hello! What is the current date?'"
    exit 1
fi

COMMAND="$1"

echo "Sending to chat $CHAT_ID: $COMMAND"

curl -X POST "https://api.telegram.org/bot${OMNI_AUTH_API_TOKEN}/sendMessage" \
  -H "Content-Type: application/json" \
  -d "{\"chat_id\": \"${CHAT_ID}\", \"text\": \"$COMMAND\"}"

echo ""

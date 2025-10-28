#!/bin/bash
# Send a message to a specific forum topic
# Usage: ./send-to-topic.sh <topic_id> <message>

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/config.sh"

if [ -z "$1" ] || [ -z "$2" ]; then
    echo "Usage: $0 <topic_id> <message>"
    echo ""
    echo "Examples:"
    echo "  $0 2 'Test message to API topic'"
    echo "  $0 5 '/status'"
    exit 1
fi

TOPIC_ID="$1"
MESSAGE="$2"

echo "Sending to chat $CHAT_ID, topic $TOPIC_ID: $MESSAGE"

curl -X POST "https://api.telegram.org/bot${OMNI_AUTH_API_TOKEN}/sendMessage" \
  -H "Content-Type: application/json" \
  -d "{\"chat_id\": \"${CHAT_ID}\", \"message_thread_id\": ${TOPIC_ID}, \"text\": \"$MESSAGE\"}"

echo ""

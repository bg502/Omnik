#!/bin/bash
# Send a query via HTTP API to Memnikai bot
# Usage: ./api-query.sh "Your message" [session_id]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ -z "$1" ]; then
    echo "Usage: $0 <message> [session_id]"
    echo ""
    echo "Examples:"
    echo "  $0 'What is the current date?'"
    echo "  $0 'List files in workspace' 'my-session'"
    exit 1
fi

MESSAGE="$1"
SESSION_ID="${2:-}"

# API endpoint
API_URL="http://localhost:8081/api/query"

echo "Sending API query to Memnikai bot..."
echo "Message: $MESSAGE"
if [ -n "$SESSION_ID" ]; then
    echo "Session: $SESSION_ID"
fi
echo ""

# Build JSON payload
if [ -n "$SESSION_ID" ]; then
    JSON_PAYLOAD=$(jq -n \
        --arg msg "$MESSAGE" \
        --arg sid "$SESSION_ID" \
        '{message: $msg, session_id: $sid}')
else
    JSON_PAYLOAD=$(jq -n \
        --arg msg "$MESSAGE" \
        '{message: $msg}')
fi

# Send request
curl -X POST "$API_URL" \
  -H "Content-Type: application/json" \
  -d "$JSON_PAYLOAD" \
  -w "\n"

echo ""

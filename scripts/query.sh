#!/bin/bash
# Send a query to Claude
# Usage: ./query.sh "Your question here"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/config.sh"

if [ -z "$1" ]; then
    echo "Usage: $0 <query>"
    echo ""
    echo "Examples:"
    echo "  $0 'What is the current date?'"
    echo "  $0 'List all files in the current directory'"
    echo "  $0 'Create a Python script that prints hello world'"
    exit 1
fi

QUERY="$1"

echo "Sending query to Claude in chat $CHAT_ID..."
echo "Query: $QUERY"
echo ""

curl -X POST "https://api.telegram.org/bot${OMNI_AUTH_API_TOKEN}/sendMessage" \
  -H "Content-Type: application/json" \
  -d "{\"chat_id\": \"${CHAT_ID}\", \"text\": \"$QUERY\"}"

echo ""

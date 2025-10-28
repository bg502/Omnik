#!/bin/bash
# Get bot status

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/config.sh"

echo "Getting bot status for chat $CHAT_ID..."

curl -X POST "https://api.telegram.org/bot${OMNI_AUTH_API_TOKEN}/sendMessage" \
  -H "Content-Type: application/json" \
  -d "{\"chat_id\": \"${CHAT_ID}\", \"text\": \"/status\"}"

echo ""

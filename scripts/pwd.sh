#!/bin/bash
# Show current working directory

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/config.sh"

echo "Getting working directory for chat $CHAT_ID..."

curl -X POST "https://api.telegram.org/bot${OMNI_AUTH_API_TOKEN}/sendMessage" \
  -H "Content-Type: application/json" \
  -d "{\"chat_id\": \"${CHAT_ID}\", \"text\": \"/pwd\"}"

echo ""

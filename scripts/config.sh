#!/bin/bash
# Configuration for Omnik Bot API scripts
# Source this file before running other scripts: source scripts/config.sh

# Load from .env if not already set
if [ -z "$OMNI_AUTH_API_TOKEN" ]; then
    ENV_FILE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/.env"
    if [ -f "$ENV_FILE" ]; then
        export $(grep -v '^#' "$ENV_FILE" | grep -E '^(OMNI_AUTH_API_TOKEN|CHAT_ID|PERSONAL_CHAT_ID)=' | xargs)
    fi
fi

# Fallback to hardcoded values if still not set
# Note: Uses Memnikai bot token to send messages to its own chat
: ${OMNI_AUTH_API_TOKEN:="8329011908:AAE6Vv-Fx5ZheexoupguXBtDVWB1ItQffqk"}
: ${CHAT_ID:="-4958242815"}
: ${PERSONAL_CHAT_ID:=""}

echo "✓ Bot configuration loaded"
echo "  OMNI_AUTH_API_TOKEN: ${OMNI_AUTH_API_TOKEN:0:10}..."
echo "  CHAT_ID: $CHAT_ID"
echo "  PERSONAL_CHAT_ID: $PERSONAL_CHAT_ID"

#!/bin/bash
set -e

# Copy custom .bashrc if it doesn't exist or is the default one
if [ ! -f "$HOME/.bashrc" ] || ! grep -q "Omnik Container" "$HOME/.bashrc" 2>/dev/null; then
    echo "📝 Installing custom .bashrc for enhanced CLI experience..."
    cp /app/.bashrc-template "$HOME/.bashrc"
fi

# Configure git user from environment variables if provided
if [ -n "$GIT_USER_NAME" ]; then
    git config --global user.name "$GIT_USER_NAME"
fi

if [ -n "$GIT_USER_EMAIL" ]; then
    git config --global user.email "$GIT_USER_EMAIL"
fi

# Execute the main command (omnik-bot)
exec "$@"

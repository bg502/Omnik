#!/bin/bash
set -e

# Configure git user from environment variables if provided
if [ -n "$GIT_USER_NAME" ]; then
    git config --global user.name "$GIT_USER_NAME"
fi

if [ -n "$GIT_USER_EMAIL" ]; then
    git config --global user.email "$GIT_USER_EMAIL"
fi

# Execute the main command (omnik-bot)
exec "$@"

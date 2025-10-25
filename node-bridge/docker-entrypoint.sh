#!/bin/sh
set -e

# Ensure .claude directory exists and has correct permissions
# This runs after volume mount, so it fixes any permission issues
if [ ! -d /home/node/.claude ]; then
    mkdir -p /home/node/.claude
fi

# Ensure installation config exists
if [ ! -f /home/node/.claude/.installation-config.json ]; then
    echo '{"installationType":"npm-global"}' > /home/node/.claude/.installation-config.json
fi

# Fix permissions if running as root (shouldn't happen, but just in case)
if [ "$(id -u)" = "0" ]; then
    chown -R node:node /home/node/.claude
    exec su-exec node "$@"
else
    exec "$@"
fi

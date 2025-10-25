#!/bin/bash
# Git credential helper that uses GITHUB_TOKEN environment variable
# This script is used by git to authenticate with GitHub using a Personal Access Token

# Read git's credential request (we don't need to parse it for this simple case)
while read line; do
    # Read until empty line
    [ -z "$line" ] && break
done

# Output credentials if GITHUB_TOKEN is set
if [ -n "$GITHUB_TOKEN" ]; then
    echo "username=git"
    echo "password=$GITHUB_TOKEN"
fi

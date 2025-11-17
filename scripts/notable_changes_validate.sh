#!/usr/bin/env bash

# This script validates that if there are any files (other than README.md) in the notable-changes-to-release.
# If yes, we require that notable-changes-to-release/notable-change.md exists.

files=$(find notable-changes-to-release -type f -not -name 'README.md')
if [ -n "$files" ]; then
    # Check if notable-change.md exists
    if [ ! -f notable-changes-to-release/notable-change.md ]; then
        echo "Validation failed: notable-changes-to-release/notable-change.md is missing."
        exit 1
    fi
fi

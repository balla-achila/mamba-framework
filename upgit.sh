#!/bin/bash

# Default commit message if none is provided
MESSAGE=${1:-"Update"}

echo "🚀 Adding changes..."
git add .

echo "💾 Committing with message: '$MESSAGE'..."
git commit -m "$MESSAGE"

echo "📤 Pushing to remote repository..."
git push

echo "✅ Done!"

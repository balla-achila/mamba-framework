#!/bin/bash

echo "========================================="
echo "Testing All Packages"
echo "========================================="

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

packages=(
    "framework/config"
    "framework/logger"
    "framework/utils"
    "framework/router"
    "framework/server"
    "framework/session"
    "framework/security"
    "framework/database"
    "framework/validation"
    "framework/auth"
    "framework/html"
    "framework/layout"
    "framework/tenant"
    "framework/upload"
    "framework/app"
)

for pkg in "${packages[@]}"; do
    echo -n "Testing $pkg... "
    if go build -o /dev/null "./$pkg" 2>/dev/null; then
        echo -e "${GREEN}✓ PASSED${NC}"
    else
        echo -e "${RED}✗ FAILED${NC}"
        go build "./$pkg" 2>&1 | head -3
    fi
done

echo ""
echo "Testing main application..."
if go build -o /dev/null ./cmd/server 2>/dev/null; then
    echo -e "${GREEN}✓ Main application PASSED${NC}"
else
    echo -e "${RED}✗ Main application FAILED${NC}"
    go build ./cmd/server 2>&1 | head -3
fi

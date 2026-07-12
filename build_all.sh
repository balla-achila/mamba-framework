#!/bin/bash

echo "========================================="
echo "Building Mamba Framework - All Packages"
echo "========================================="

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

packages=(
    "config"
    "logger"
    "router"
    "server"
    "session"
    "security"
    "database"
)

for pkg in "${packages[@]}"; do
    echo -n "Building framework/$pkg... "
    if go build -o /dev/null "./framework/$pkg" 2>/dev/null; then
        echo -e "${GREEN}✅${NC}"
    else
        echo -e "${RED}❌ Failed${NC}"
        exit 1
    fi
done

echo ""
echo -n "Building main application... "
if go build -v -o bin/mamba cmd/server/main.go 2>/dev/null; then
    echo -e "${GREEN}✅${NC}"
    echo ""
    echo "========================================="
    echo -e "${GREEN}🎉 All packages built successfully!${NC}"
    echo "========================================="
    echo ""
    echo "Binary: bin/mamba"
    ls -lh bin/mamba
else
    echo -e "${RED}❌ Failed${NC}"
    exit 1
fi

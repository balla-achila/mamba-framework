#!/bin/bash

echo "========================================="
echo "Building Complete Mamba Framework"
echo "========================================="

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

packages=(
    "config"
    "logger"
    "utils"
    "router"
    "server"
    "session"
    "security"
    "database"
    "validation"
    "auth"
    "html"
    "layout"
    "tenant"
    "upload"
    "app"
)

failed=0
for pkg in "${packages[@]}"; do
    echo -n "Building framework/$pkg... "
    if go build -o /dev/null "./framework/$pkg" 2>/dev/null; then
        echo -e "${GREEN}✅${NC}"
    else
        echo -e "${RED}❌ Failed${NC}"
        failed=1
    fi
done

echo ""
if [ $failed -eq 0 ]; then
    echo -e "${GREEN}🎉 All packages built successfully!${NC}"
    echo ""
    echo "Building main application..."
    if go build -v -o bin/mamba cmd/server/main.go 2>&1 | tee build.log; then
        echo ""
        echo -e "${GREEN}✅ Main application built successfully!${NC}"
        echo ""
        echo "========================================="
        echo -e "${GREEN}🎉 Mamba Framework Complete!${NC}"
        echo "========================================="
        echo ""
        echo "Binary: bin/mamba"
        ls -lh bin/mamba
        echo ""
        echo "Run with: ./bin/mamba"
    else
        echo -e "${RED}❌ Main application build failed${NC}"
    fi
else
    echo -e "${RED}❌ Some packages failed to build${NC}"
fi

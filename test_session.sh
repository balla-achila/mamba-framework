#!/bin/bash

echo "========================================="
echo "Testing Session Package"
echo "========================================="

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check both files exist
echo -n "Checking session files... "
if [ -f framework/session/context.go ] && [ -f framework/session/session.go ]; then
    echo -e "${GREEN}✓ Both files exist${NC}"
else
    echo -e "${RED}✗ Missing files${NC}"
    exit 1
fi

# Check for ContextWithSession function
echo -n "Checking ContextWithSession function... "
if grep -q "func ContextWithSession" framework/session/context.go; then
    echo -e "${GREEN}✓ Found${NC}"
else
    echo -e "${RED}✗ Missing${NC}"
    exit 1
fi

# Check for FromContext function
echo -n "Checking FromContext function... "
if grep -q "func FromContext" framework/session/context.go; then
    echo -e "${GREEN}✓ Found${NC}"
else
    echo -e "${RED}✗ Missing${NC}"
    exit 1
fi

# Compile the ENTIRE package (not just one file)
echo -n "Compiling session package... "
if go build -o /dev/null ./framework/session 2>/dev/null; then
    echo -e "${GREEN}✓ Compiled successfully${NC}"
else
    echo -e "${RED}✗ Compilation failed${NC}"
    go build ./framework/session 2>&1
    exit 1
fi

echo ""
echo -e "${GREEN}✅ Session package is ready!${NC}"

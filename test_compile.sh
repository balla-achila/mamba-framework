#!/bin/bash

echo "========================================="
echo "Mamba Framework Compilation Test"
echo "========================================="

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0

# Function to test compilation
test_compile() {
    local pkg=$1
    echo -n "Testing $pkg... "
    
    if go build -o /dev/null "$pkg" 2>/dev/null; then
        echo -e "${GREEN}✓ PASSED${NC}"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAILED${NC}"
        go build "$pkg" 2>&1 | head -5
        ((FAILED++))
        return 1
    fi
}

echo ""
echo "Step 1: Checking Go environment..."
go version
echo ""

echo "Step 2: Downloading dependencies..."
go mod download
go mod tidy
echo ""

echo "Step 3: Testing individual packages..."
echo "----------------------------------------"

# Test each package
test_compile "./framework/config"
test_compile "./framework/logger"
test_compile "./framework/utils"
test_compile "./framework/html"
test_compile "./framework/validation"
test_compile "./framework/session"
test_compile "./framework/security"
test_compile "./framework/database"
test_compile "./framework/tenant"
test_compile "./framework/upload"
test_compile "./framework/auth"
test_compile "./framework/layout"
test_compile "./framework/router"
test_compile "./framework/app"
test_compile "./framework/server"

echo "----------------------------------------"
echo ""

echo "Step 4: Testing main application..."
if go build -o bin/mamba cmd/server/main.go 2>/dev/null; then
    echo -e "${GREEN}✓ Main application compiled successfully${NC}"
    ls -lh bin/mamba
    ((PASSED++))
else
    echo -e "${RED}✗ Main application failed to compile${NC}"
    go build -o bin/mamba cmd/server/main.go 2>&1 | head -10
    ((FAILED++))
fi

echo ""
echo "Step 5: Testing all packages together..."
if go build ./... 2>/dev/null; then
    echo -e "${GREEN}✓ All packages compiled successfully${NC}"
    ((PASSED++))
else
    echo -e "${RED}✗ Some packages failed to compile${NC}"
    go build ./... 2>&1 | head -10
    ((FAILED++))
fi

echo ""
echo "========================================="
echo "Compilation Test Results"
echo "========================================="
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed! Framework is ready.${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tests failed. Please fix the issues above.${NC}"
    exit 1
fi

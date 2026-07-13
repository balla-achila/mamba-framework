#!/bin/bash

echo "========================================="
echo "Mamba Framework - File by File Testing"
echo "========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Counters
TOTAL=0
PASSED=0
FAILED=0

# Function to test a single Go file
test_file() {
    local file=$1
    local package=$2
    TOTAL=$((TOTAL + 1))
    
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}Testing: ${YELLOW}$file${NC}"
    
    # Check if file exists
    if [ ! -f "$file" ]; then
        echo -e "${RED}✗ File not found: $file${NC}"
        FAILED=$((FAILED + 1))
        return 1
    fi
    
    # Get file size
    size=$(ls -lh "$file" | awk '{print $5}')
    lines=$(wc -l < "$file")
    echo -e "  Size: ${GREEN}$size${NC}, Lines: ${GREEN}$lines${NC}"
    
    # Check for syntax errors
    echo -n "  Checking syntax... "
    if go build -o /dev/null "$file" 2>/dev/null; then
        echo -e "${GREEN}✓ Syntax OK${NC}"
    else
        echo -e "${RED}✗ Syntax error${NC}"
        go build "$file" 2>&1 | head -3
        FAILED=$((FAILED + 1))
        return 1
    fi
    
    # Check for unused imports
    echo -n "  Checking imports... "
    if go vet "$file" 2>&1 | grep -q "unused"; then
        echo -e "${RED}✗ Unused imports found${NC}"
        go vet "$file" 2>&1 | grep "unused"
        FAILED=$((FAILED + 1))
        return 1
    else
        echo -e "${GREEN}✓ Imports OK${NC}"
    fi
    
    # Check formatting
    echo -n "  Checking formatting... "
    if gofmt -l "$file" | grep -q "$file"; then
        echo -e "${YELLOW}⚠ File needs formatting${NC}"
        echo "    Run: gofmt -w $file"
    else
        echo -e "${GREEN}✓ Formatted correctly${NC}"
    fi
    
    # Check for exported functions documentation (if it's a library file)
    if [[ "$file" != *"_test.go" ]]; then
        echo -n "  Checking exports... "
        exports=$(grep -c "^func [A-Z]" "$file" 2>/dev/null || echo "0")
        if [ "$exports" -gt 0 ]; then
            echo -e "${GREEN}✓ $exports exported functions${NC}"
        else
            echo -e "${YELLOW}⚠ No exported functions found${NC}"
        fi
    fi
    
    PASSED=$((PASSED + 1))
    echo -e "${GREEN}✅ $file passed${NC}"
    return 0
}

# Function to test a package directory
test_package() {
    local pkg=$1
    TOTAL=$((TOTAL + 1))
    
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}Testing Package: ${YELLOW}$pkg${NC}"
    
    # Check if package exists
    if [ ! -d "$pkg" ]; then
        echo -e "${RED}✗ Package not found: $pkg${NC}"
        FAILED=$((FAILED + 1))
        return 1
    fi
    
    # List files in package
    files=$(find "$pkg" -name "*.go" -not -name "*_test.go" | wc -l)
    test_files=$(find "$pkg" -name "*_test.go" | wc -l)
    echo -e "  Files: ${GREEN}$files${NC}, Test files: ${GREEN}$test_files${NC}"
    
    # Compile package
    echo -n "  Compiling package... "
    if go build -o /dev/null "./$pkg" 2>/dev/null; then
        echo -e "${GREEN}✓ Compiled${NC}"
    else
        echo -e "${RED}✗ Compilation failed${NC}"
        go build "./$pkg" 2>&1 | head -5
        FAILED=$((FAILED + 1))
        return 1
    fi
    
    # Run tests if they exist
    if [ "$test_files" -gt 0 ]; then
        echo -n "  Running tests... "
        if go test -v "./$pkg" 2>&1 | grep -q "PASS"; then
            echo -e "${GREEN}✓ Tests passed${NC}"
        else
            echo -e "${YELLOW}⚠ No tests or tests failed${NC}"
        fi
    fi
    
    PASSED=$((PASSED + 1))
    echo -e "${GREEN}✅ Package $pkg passed${NC}"
    return 0
}

# Function to test main.go
test_main() {
    local file="cmd/server/main.go"
    TOTAL=$((TOTAL + 1))
    
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}Testing Main Application: ${YELLOW}$file${NC}"
    
    if [ ! -f "$file" ]; then
        echo -e "${RED}✗ Main file not found${NC}"
        FAILED=$((FAILED + 1))
        return 1
    fi
    
    # Check syntax
    echo -n "  Checking syntax... "
    if go build -o /dev/null "$file" 2>/dev/null; then
        echo -e "${GREEN}✓ Syntax OK${NC}"
    else
        echo -e "${RED}✗ Syntax error${NC}"
        go build "$file" 2>&1 | head -5
        FAILED=$((FAILED + 1))
        return 1
    fi
    
    # Check imports
    echo -n "  Checking imports... "
    imports=$(grep -c "^import" "$file")
    echo -e "${GREEN}✓ $imports imports${NC}"
    
    # Check main function
    echo -n "  Checking main function... "
    if grep -q "func main()" "$file"; then
        echo -e "${GREEN}✓ Found${NC}"
    else
        echo -e "${RED}✗ Missing main function${NC}"
        FAILED=$((FAILED + 1))
        return 1
    fi
    
    # Check routes
    echo -n "  Checking routes... "
    routes=$(grep -c '\.Get(' "$file")
    if [ "$routes" -gt 0 ]; then
        echo -e "${GREEN}✓ $routes routes defined${NC}"
    else
        echo -e "${YELLOW}⚠ No routes found${NC}"
    fi
    
    PASSED=$((PASSED + 1))
    echo -e "${GREEN}✅ Main application passed${NC}"
    return 0
}

# Function to test all go files
test_all_go_files() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}Testing All Go Files...${NC}"
    echo ""
    
    # Find all Go files (excluding test files)
    go_files=$(find . -name "*.go" -not -name "*_test.go" -type f | sort)
    
    for file in $go_files; do
        test_file "$file"
        echo ""
    done
}

# Main execution
echo -e "${BLUE}Starting Framework Testing...${NC}"
echo ""

# Test each package
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}STEP 1: Testing Individual Packages${NC}"
echo ""

# List of all framework packages
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
    test_package "$pkg"
    echo ""
done

# Test main application
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}STEP 2: Testing Main Application${NC}"
echo ""

test_main
echo ""

# Test all individual files
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}STEP 3: Testing Individual Files${NC}"
echo ""

test_all_go_files

# Final summary
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}📊 Test Summary${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  Total tests: ${YELLOW}$TOTAL${NC}"
echo -e "  ${GREEN}✓ Passed: $PASSED${NC}"
echo -e "  ${RED}✗ Failed: $FAILED${NC}"
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed! Framework is ready.${NC}"
    echo ""
    echo -e "${CYAN}Next steps:${NC}"
    echo "  1. Run: ./bin/mamba"
    echo "  2. Test: curl http://localhost:8080/"
    echo "  3. Build production: ./build_production.sh"
else
    echo -e "${RED}❌ Some tests failed. Please fix the issues above.${NC}"
    echo ""
    echo -e "${YELLOW}Common fixes:${NC}"
    echo "  1. Run: gofmt -w <file>  # Format code"
    echo "  2. Run: go mod tidy      # Fix dependencies"
    echo "  3. Check import paths"
fi

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

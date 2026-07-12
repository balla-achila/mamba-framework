#!/bin/bash

echo "========================================="
echo "Testing Mamba Framework - All Endpoints"
echo "========================================="

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test home
echo -e "${YELLOW}Testing Home Page...${NC}"
curl -s http://localhost:8080/ | head -5
echo ""

# Test health
echo -e "${YELLOW}Testing Health...${NC}"
curl -s http://localhost:8080/health
echo ""

# Test hello
echo -e "${YELLOW}Testing Hello...${NC}"
curl -s http://localhost:8080/hello/Mamba
echo ""

# Test session
echo -e "${YELLOW}Testing Session (first visit)...${NC}"
curl -s -c cookies.txt http://localhost:8080/session-test
echo ""

echo -e "${YELLOW}Testing Session (second visit)...${NC}"
curl -s -b cookies.txt http://localhost:8080/session-test
echo ""

# Test API
echo -e "${YELLOW}Testing API Users...${NC}"
curl -s http://localhost:8080/api/users
echo ""

echo ""
echo "========================================="
echo -e "${GREEN}All tests completed!${NC}"
echo "========================================="

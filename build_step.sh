#!/bin/bash

echo "========================================="
echo "Building Mamba Framework - Step by Step"
echo "========================================="

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Step 1: Download dependencies
echo -e "${YELLOW}Step 1: Downloading dependencies...${NC}"
go mod download
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Failed to download dependencies${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Dependencies downloaded${NC}"

# Step 2: Tidy modules
echo -e "${YELLOW}Step 2: Tidying modules...${NC}"
go mod tidy
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Failed to tidy modules${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Modules tidied${NC}"

# Step 3: Build config
echo -e "${YELLOW}Step 3: Building config package...${NC}"
go build -o /dev/null ./framework/config 2>/dev/null
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Config package built${NC}"
else
    echo -e "${RED}❌ Config package failed${NC}"
    exit 1
fi

# Step 4: Build logger
echo -e "${YELLOW}Step 4: Building logger package...${NC}"
go build -o /dev/null ./framework/logger 2>/dev/null
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Logger package built${NC}"
else
    echo -e "${RED}❌ Logger package failed${NC}"
    exit 1
fi

# Step 5: Build router
echo -e "${YELLOW}Step 5: Building router package...${NC}"
go build -o /dev/null ./framework/router 2>/dev/null
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Router package built${NC}"
else
    echo -e "${RED}❌ Router package failed${NC}"
    exit 1
fi

# Step 6: Build main
echo -e "${YELLOW}Step 6: Building main application...${NC}"
go build -v -o bin/mamba cmd/server/main.go 2>&1 | tee build.log

if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo -e "${GREEN}✅ Main application built successfully!${NC}"
    echo ""
    echo "========================================="
    echo -e "${GREEN}🎉 Framework built successfully!${NC}"
    echo "========================================="
    echo ""
    echo "Binary: bin/mamba"
    ls -lh bin/mamba
    echo ""
    echo "Run with: ./bin/mamba"
else
    echo -e "${RED}❌ Build failed${NC}"
    echo "Check build.log for details"
    exit 1
fi

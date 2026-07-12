#!/bin/bash

echo "========================================="
echo "Building Mamba Framework - Production"
echo "========================================="

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Clean
echo -e "${YELLOW}Cleaning...${NC}"
rm -rf bin/
go clean -cache

# Download dependencies
echo -e "${YELLOW}Downloading dependencies...${NC}"
go mod download
go mod tidy

# Build with optimizations
echo -e "${YELLOW}Building optimized binary...${NC}"
CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=1.0.0" -o bin/mamba cmd/server/main.go

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Production build successful!${NC}"
    echo ""
    echo "Binary: bin/mamba"
    ls -lh bin/mamba
    echo ""
    echo "To run: ./bin/mamba"
    echo "To run with custom config: ./bin/mamba -config config/production.json"
else
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

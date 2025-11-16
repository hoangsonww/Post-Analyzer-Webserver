#!/bin/bash

# Post Analyzer Webserver - Setup Script
# This script helps set up the development environment

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   Post Analyzer Webserver - Setup Script           ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}"
echo ""

# Check Go installation
echo -e "${YELLOW}→${NC} Checking Go installation..."
if ! command -v go &> /dev/null; then
    echo -e "${RED}✗${NC} Go is not installed. Please install Go 1.21 or higher."
    echo -e "  Download from: https://golang.org/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo -e "${GREEN}✓${NC} Go ${GO_VERSION} is installed"

# Check if Docker is installed (optional)
echo -e "${YELLOW}→${NC} Checking Docker installation..."
if command -v docker &> /dev/null; then
    DOCKER_VERSION=$(docker --version | awk '{print $3}' | sed 's/,//')
    echo -e "${GREEN}✓${NC} Docker ${DOCKER_VERSION} is installed"
    HAS_DOCKER=true
else
    echo -e "${YELLOW}!${NC} Docker is not installed (optional)"
    HAS_DOCKER=false
fi

# Check if Docker Compose is installed (optional)
if [ "$HAS_DOCKER" = true ]; then
    echo -e "${YELLOW}→${NC} Checking Docker Compose installation..."
    if command -v docker-compose &> /dev/null; then
        COMPOSE_VERSION=$(docker-compose --version | awk '{print $4}' | sed 's/,//')
        echo -e "${GREEN}✓${NC} Docker Compose ${COMPOSE_VERSION} is installed"
        HAS_COMPOSE=true
    else
        echo -e "${YELLOW}!${NC} Docker Compose is not installed (optional)"
        HAS_COMPOSE=false
    fi
fi

# Install Go dependencies
echo ""
echo -e "${YELLOW}→${NC} Installing Go dependencies..."
go mod download
go mod tidy
echo -e "${GREEN}✓${NC} Dependencies installed"

# Create .env file if it doesn't exist
echo ""
echo -e "${YELLOW}→${NC} Setting up environment configuration..."
if [ ! -f .env ]; then
    if [ -f .env.example ]; then
        cp .env.example .env
        echo -e "${GREEN}✓${NC} Created .env from .env.example"
    else
        echo -e "${YELLOW}!${NC} .env.example not found, creating default .env"
        cat > .env <<EOF
PORT=8080
ENVIRONMENT=development
DB_TYPE=file
DB_FILE_PATH=posts.json
LOG_LEVEL=info
LOG_FORMAT=json
RATE_LIMIT_REQUESTS=100
EOF
        echo -e "${GREEN}✓${NC} Created default .env file"
    fi
else
    echo -e "${GREEN}✓${NC} .env file already exists"
fi

# Build the application
echo ""
echo -e "${YELLOW}→${NC} Building application..."
if go build -o post-analyzer main.go; then
    echo -e "${GREEN}✓${NC} Application built successfully"
else
    echo -e "${RED}✗${NC} Build failed"
    exit 1
fi

# Run tests
echo ""
echo -e "${YELLOW}→${NC} Running tests..."
if go test ./... -v; then
    echo -e "${GREEN}✓${NC} All tests passed"
else
    echo -e "${RED}✗${NC} Some tests failed"
fi

# Setup complete
echo ""
echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║              Setup Complete! 🎉                      ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Next steps:${NC}"
echo ""
echo -e "  1. Review your configuration:"
echo -e "     ${YELLOW}nano .env${NC}"
echo ""
echo -e "  2. Run the application:"
echo -e "     ${YELLOW}./post-analyzer${NC}"
echo -e "     or"
echo -e "     ${YELLOW}make run${NC}"
echo ""

if [ "$HAS_DOCKER" = true ] && [ "$HAS_COMPOSE" = true ]; then
    echo -e "  3. Or use Docker Compose (recommended):"
    echo -e "     ${YELLOW}docker-compose up -d${NC}"
    echo ""
fi

echo -e "  4. Access the application:"
echo -e "     ${BLUE}http://localhost:8080${NC}"
echo ""
echo -e "  5. View available commands:"
echo -e "     ${YELLOW}make help${NC}"
echo ""
echo -e "${GREEN}For more information, see README_PRODUCTION.md${NC}"
echo ""

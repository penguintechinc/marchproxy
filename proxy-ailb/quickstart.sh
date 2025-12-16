#!/bin/bash
# AILB Quick Start Script
# Sets up and runs the AILB container for development

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   AILB Quick Start${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}No .env file found. Creating from .env.example...${NC}"
    cp .env.example .env
    echo -e "${GREEN}.env file created!${NC}"
    echo -e "${YELLOW}Please edit .env and add your API keys before continuing.${NC}"
    echo ""
    read -p "Press Enter to open .env in editor (or Ctrl+C to exit)..."
    ${EDITOR:-nano} .env
fi

# Generate gRPC proto code
echo -e "${BLUE}Step 1: Generating gRPC proto code...${NC}"
if [ -f generate_proto.sh ]; then
    chmod +x generate_proto.sh
    ./generate_proto.sh
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Proto code generated${NC}"
    else
        echo -e "${YELLOW}⚠ Proto generation failed, continuing anyway...${NC}"
    fi
else
    echo -e "${YELLOW}⚠ generate_proto.sh not found, skipping...${NC}"
fi
echo ""

# Choose run mode
echo -e "${BLUE}Step 2: Choose run mode${NC}"
echo "1) Docker Compose (recommended)"
echo "2) Docker Build & Run"
echo "3) Local Python (development)"
echo ""
read -p "Enter choice [1-3]: " choice

case $choice in
    1)
        echo -e "${BLUE}Starting with Docker Compose...${NC}"
        docker-compose down 2>/dev/null || true
        docker-compose up --build -d
        echo ""
        echo -e "${GREEN}✓ AILB started with Docker Compose${NC}"
        echo -e "${YELLOW}Logs: docker-compose logs -f ailb${NC}"
        echo -e "${YELLOW}Stop: docker-compose down${NC}"
        ;;
    2)
        echo -e "${BLUE}Building Docker image...${NC}"
        docker build -t marchproxy/ailb:latest .
        echo ""
        echo -e "${BLUE}Starting container...${NC}"
        docker run -d \
            --name marchproxy-ailb \
            -p 8080:8080 \
            -p 50051:50051 \
            --env-file .env \
            -v ailb_memory:/app/ailb_memory \
            -v ailb_rag:/app/ailb_rag \
            marchproxy/ailb:latest
        echo ""
        echo -e "${GREEN}✓ AILB started in Docker${NC}"
        echo -e "${YELLOW}Logs: docker logs -f marchproxy-ailb${NC}"
        echo -e "${YELLOW}Stop: docker stop marchproxy-ailb && docker rm marchproxy-ailb${NC}"
        ;;
    3)
        echo -e "${BLUE}Installing Python dependencies...${NC}"
        pip install -r requirements.txt
        echo ""
        echo -e "${BLUE}Starting AILB locally...${NC}"
        # Load .env file
        export $(cat .env | grep -v '^#' | xargs)
        python main.py
        ;;
    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}AILB is running!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "HTTP API: ${GREEN}http://localhost:8080${NC}"
echo -e "gRPC API: ${GREEN}localhost:50051${NC}"
echo ""
echo -e "Test endpoints:"
echo -e "  Health: ${YELLOW}curl http://localhost:8080/healthz${NC}"
echo -e "  Models: ${YELLOW}curl http://localhost:8080/v1/models${NC}"
echo -e "  Stats:  ${YELLOW}curl http://localhost:8080/api/routing/stats${NC}"
echo ""
echo -e "Chat completion example:"
echo -e "${YELLOW}curl -X POST http://localhost:8080/v1/chat/completions \\${NC}"
echo -e "${YELLOW}  -H 'Content-Type: application/json' \\${NC}"
echo -e "${YELLOW}  -d '{${NC}"
echo -e "${YELLOW}    \"model\": \"gpt-3.5-turbo\",${NC}"
echo -e "${YELLOW}    \"messages\": [{\"role\": \"user\", \"content\": \"Hello!\"}]${NC}"
echo -e "${YELLOW}  }'${NC}"
echo ""

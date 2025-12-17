# AILB Configuration Guide

## Environment Variables

### HTTP Server

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | 8080 | FastAPI server port |
| `HOST` | 0.0.0.0 | Bind address |
| `PYTHONUNBUFFERED` | 1 | Unbuffered output |

### gRPC Server

| Variable | Default | Description |
|----------|---------|-------------|
| `GRPC_PORT` | 50051 | gRPC ModuleService port |
| `MODULE_ID` | ailb-1 | Unique module identifier |

### Routing

| Variable | Default | Description |
|----------|---------|-------------|
| `ROUTING_STRATEGY` | load_balanced | round_robin, cost_optimized, latency_optimized, load_balanced, failover, random |

**Example:**
```bash
export ROUTING_STRATEGY=cost_optimized
```

### Feature Flags

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_MEMORY` | true | Enable conversation memory |
| `ENABLE_RAG` | false | Enable RAG support |
| `MEMORY_BACKEND` | chromadb | Memory storage (chromadb) |
| `RAG_BACKEND` | chromadb | RAG storage (chromadb) |

**Example:**
```bash
export ENABLE_MEMORY=true
export ENABLE_RAG=true
export MEMORY_BACKEND=chromadb
export RAG_BACKEND=chromadb
```

### OpenAI Provider

| Variable | Required | Description |
|----------|----------|-------------|
| `OPENAI_API_KEY` | Yes | OpenAI API key |
| `OPENAI_BASE_URL` | No | Custom OpenAI endpoint |
| `OPENAI_MODELS` | No | Comma-separated models (default: gpt-4,gpt-3.5-turbo) |

**Example:**
```bash
export OPENAI_API_KEY=sk-proj-...
export OPENAI_MODELS=gpt-4,gpt-4-turbo,gpt-3.5-turbo
```

### Anthropic Provider

| Variable | Required | Description |
|----------|----------|-------------|
| `ANTHROPIC_API_KEY` | Yes | Anthropic API key |
| `ANTHROPIC_MODELS` | No | Comma-separated models (default: claude-3-opus, claude-3-sonnet) |

**Example:**
```bash
export ANTHROPIC_API_KEY=sk-ant-...
export ANTHROPIC_MODELS=claude-3-opus-20240229,claude-3-sonnet-20240229
```

### Ollama Provider

| Variable | Required | Description |
|----------|----------|-------------|
| `OLLAMA_BASE_URL` | No | Ollama server URL |

**Example:**
```bash
export OLLAMA_BASE_URL=http://localhost:11434
```

## Docker Configuration

### Building the Image

```bash
docker build -t marchproxy/ailb:latest .
```

### Running the Container

```bash
docker run -d \
  --name ailb \
  -p 8080:8080 \
  -p 50051:50051 \
  -e OPENAI_API_KEY=sk-... \
  -e ANTHROPIC_API_KEY=sk-ant-... \
  -e ENABLE_MEMORY=true \
  -e ENABLE_RAG=false \
  -v ailb_data:/app/ailb_memory \
  -v ailb_rag:/app/ailb_rag \
  --health-cmd="curl -f http://localhost:8080/healthz || exit 1" \
  --health-interval=30s \
  --health-timeout=10s \
  --health-retries=3 \
  marchproxy/ailb:latest
```

### Docker Compose

```yaml
version: '3.8'

services:
  ailb:
    image: marchproxy/ailb:latest
    ports:
      - "8080:8080"
      - "50051:50051"
    environment:
      HTTP_PORT: 8080
      GRPC_PORT: 50051
      MODULE_ID: ailb-1
      ROUTING_STRATEGY: load_balanced
      ENABLE_MEMORY: "true"
      ENABLE_RAG: "false"
      OPENAI_API_KEY: ${OPENAI_API_KEY}
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
      OLLAMA_BASE_URL: http://ollama:11434
    volumes:
      - ailb_data:/app/ailb_memory
      - ailb_rag:/app/ailb_rag
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
    depends_on:
      - ollama

  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama_models:/root/.ollama
    environment:
      OLLAMA_HOST: 0.0.0.0:11434

volumes:
  ailb_data:
  ailb_rag:
  ollama_models:
```

## Local Development

### Prerequisites

- Python 3.11+
- pip or uv package manager
- API keys for at least one provider (OpenAI or Anthropic)

### Installation

```bash
# Clone repository
cd /home/penguin/code/MarchProxy/proxy-ailb

# Create virtual environment
python3 -m venv venv
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Generate gRPC code
python -m grpc_tools.protoc \
  -I../proto \
  --python_out=./proto \
  --grpc_python_out=./proto \
  ../proto/marchproxy/module_service.proto
```

### Running Locally

```bash
# Set environment variables
export OPENAI_API_KEY=sk-...
export ANTHROPIC_API_KEY=sk-ant-...

# Run server
python main.py
```

Server will start on:
- HTTP: http://localhost:8080
- gRPC: localhost:50051

### Testing Locally

```bash
# Chat completions
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'

# List models
curl http://localhost:8080/v1/models

# Health check
curl http://localhost:8080/healthz

# Metrics
curl http://localhost:8080/metrics
```

## Provider Setup

### OpenAI

1. Create account at https://platform.openai.com
2. Generate API key in Settings → API Keys
3. Set `OPENAI_API_KEY` environment variable
4. Optional: Set `OPENAI_MODELS` to limit available models

### Anthropic

1. Create account at https://console.anthropic.com
2. Generate API key in Account Settings
3. Set `ANTHROPIC_API_KEY` environment variable
4. Optional: Set `ANTHROPIC_MODELS` to limit available models

### Ollama

1. Download Ollama from https://ollama.ai
2. Run: `ollama serve`
3. Pull models: `ollama pull mistral` or `ollama pull neural-chat`
4. Set `OLLAMA_BASE_URL` to Ollama server URL (default: http://localhost:11434)

## Storage Configuration

### ChromaDB (Memory & RAG)

Default persistent storage location:
- Memory: `/app/ailb_memory/`
- RAG: `/app/ailb_rag/`

Both use ChromaDB with SQLite backend. Data persists across container restarts if volume is mounted.

**Volume Mount (Docker):**
```bash
-v ailb_data:/app/ailb_memory
-v ailb_rag:/app/ailb_rag
```

## Performance Tuning

### Connection Pooling
- Persistent connections to all configured providers
- Automatic reconnection on failure
- Connection timeout: 30 seconds

### Rate Limiting
- Per-user rate limits enforced at middleware level
- Per-provider limits tracked independently
- Burst handling with token bucket algorithm

### Caching
- Frequent contexts cached in-memory (LRU)
- RAG results cached by query hash
- Cache TTL: 1 hour

### Resource Limits
- Memory: No explicit limit (container default)
- Disk: Required for ChromaDB (minimum 1GB)
- CPU: Async I/O, no CPU-intensive operations

## Security Configuration

### API Key Authentication
- Bearer token in `Authorization: Bearer <key>` header
- Keys validated against KeyManager
- Invalid keys return 401 Unauthorized

### Rate Limiting
- RPM (requests per minute) limits per key
- TPM (tokens per minute) limits per key
- Rate limit exceeded returns 429 Too Many Requests

### Budget Enforcement
- Monthly dollar limits per API key
- Request rejected if would exceed budget
- Budget exceeded returns 402 Payment Required

## Production Deployment Checklist

- [ ] Set strong `OPENAI_API_KEY` and `ANTHROPIC_API_KEY`
- [ ] Configure persistent volumes for memory/RAG
- [ ] Set appropriate `ROUTING_STRATEGY` (usually load_balanced)
- [ ] Enable `ENABLE_MEMORY=true` for stateful apps
- [ ] Configure health checks for orchestrator
- [ ] Set resource limits (CPU/memory)
- [ ] Enable metrics collection (`/metrics`)
- [ ] Configure logging aggregation
- [ ] Review rate limiting quotas
- [ ] Test failover scenarios

## Troubleshooting

### Provider Not Loading
```bash
# Verify environment variable is set
echo $OPENAI_API_KEY

# Check logs for provider error
docker logs <container-id>
```

### Memory/RAG Data Loss
```bash
# Ensure volume is mounted correctly
docker inspect <container-id> | grep Mounts

# Check disk space
df -h /app/ailb_memory
```

### Connection Timeouts
```bash
# Check provider API status
curl https://status.openai.com

# Verify network connectivity
curl -v https://api.openai.com
```

---

**Last Updated:** 2025-12-16

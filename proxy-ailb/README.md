# AILB - AI Load Balancer Container

The AILB (AI Load Balancer) container is a specialized proxy for AI/LLM requests with intelligent routing, conversation memory, and RAG (Retrieval-Augmented Generation) support.

## Features

- **Multiple LLM Provider Support**
  - OpenAI (GPT-4, GPT-3.5-turbo)
  - Anthropic (Claude 3 Opus, Sonnet, Haiku)
  - Ollama (Local LLM hosting)

- **Intelligent Routing**
  - Round-robin load balancing
  - Latency-optimized routing
  - Cost-optimized routing
  - Automatic failover

- **Conversation Memory**
  - Session-based conversation context
  - Vector similarity search for relevant history
  - ChromaDB-backed storage

- **RAG Support**
  - Knowledge base integration
  - Context-aware response generation
  - Multiple collection support

- **gRPC ModuleService Interface**
  - Health status reporting
  - Route configuration
  - Traffic metrics
  - Blue/green deployment support

## Architecture

```
┌──────────────────────────────────────┐
│         NLB (Network Load)           │
│      Routes traffic to AILB          │
└────────────────┬─────────────────────┘
                 │
                 v
┌──────────────────────────────────────┐
│  AILB Container (This Module)        │
│                                      │
│  ┌────────────────────────────┐     │
│  │  FastAPI HTTP Server       │     │
│  │  - /v1/chat/completions    │     │
│  │  - /v1/models              │     │
│  │  - /healthz                │     │
│  └────────────────────────────┘     │
│                                      │
│  ┌────────────────────────────┐     │
│  │  gRPC ModuleService        │     │
│  │  - GetStatus()             │     │
│  │  - GetMetrics()            │     │
│  │  - SetTrafficWeight()      │     │
│  └────────────────────────────┘     │
│                                      │
│  ┌────────────────────────────┐     │
│  │  Intelligent Router        │     │
│  │  - Provider selection      │     │
│  │  - Load balancing          │     │
│  │  - Automatic failover      │     │
│  └────────────────────────────┘     │
│                                      │
│  ┌────────────────────────────┐     │
│  │  Memory Manager            │     │
│  │  - ChromaDB storage        │     │
│  │  - Context retrieval       │     │
│  └────────────────────────────┘     │
│                                      │
│  ┌────────────────────────────┐     │
│  │  RAG Manager               │     │
│  │  - Knowledge base search   │     │
│  │  - Context enrichment      │     │
│  └────────────────────────────┘     │
└──────────────┬───────────────────────┘
               │
               v
┌──────────────────────────────────────┐
│      LLM Providers                   │
│  ┌──────────┐ ┌──────────┐          │
│  │ OpenAI   │ │ Anthropic│          │
│  └──────────┘ └──────────┘          │
│  ┌──────────┐                        │
│  │ Ollama   │                        │
│  └──────────┘                        │
└──────────────────────────────────────┘
```

## Configuration

### Environment Variables

```bash
# HTTP Server
HTTP_PORT=8080                    # FastAPI server port
HOST=0.0.0.0                      # Bind address

# gRPC Server
GRPC_PORT=50051                   # ModuleService gRPC port
MODULE_ID=ailb-1                  # Unique module identifier

# Routing
ROUTING_STRATEGY=load_balanced    # round_robin|cost_optimized|latency_optimized|load_balanced|failover|random

# Memory
ENABLE_MEMORY=true                # Enable conversation memory
MEMORY_BACKEND=chromadb           # Memory storage backend

# RAG
ENABLE_RAG=false                  # Enable RAG support
RAG_BACKEND=chromadb              # RAG storage backend

# OpenAI Provider
OPENAI_API_KEY=sk-...             # OpenAI API key
OPENAI_BASE_URL=                  # Optional: custom endpoint
OPENAI_MODELS=gpt-4,gpt-3.5-turbo # Comma-separated model list

# Anthropic Provider
ANTHROPIC_API_KEY=sk-ant-...      # Anthropic API key
ANTHROPIC_MODELS=claude-3-opus-20240229,claude-3-sonnet-20240229

# Ollama Provider
OLLAMA_BASE_URL=http://localhost:11434  # Ollama server URL
```

## API Endpoints

### HTTP API (Port 8080)

#### Chat Completions
```bash
POST /v1/chat/completions
Content-Type: application/json

{
  "model": "gpt-4",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "session_id": "optional-session-id",  // For memory
  "rag_collection": "optional-collection", // For RAG
  "rag_top_k": 3
}
```

#### List Models
```bash
GET /v1/models
```

#### Routing Statistics
```bash
GET /api/routing/stats
```

#### Health Check
```bash
GET /healthz
```

### gRPC API (Port 50051)

Implements the `ModuleService` interface:

- `GetStatus()` - Module health and status
- `GetRoutes()` - Route configuration
- `GetMetrics()` - Performance metrics
- `ApplyRateLimit()` - Rate limiting
- `SetTrafficWeight()` - Blue/green deployment
- `Reload()` - Configuration reload

## Building and Running

### Docker Build

```bash
docker build -t marchproxy/ailb:latest .
```

### Docker Run

```bash
docker run -d \
  -p 8080:8080 \
  -p 50051:50051 \
  -e OPENAI_API_KEY=sk-... \
  -e ANTHROPIC_API_KEY=sk-ant-... \
  -e OLLAMA_BASE_URL=http://host.docker.internal:11434 \
  -e ENABLE_MEMORY=true \
  -v ailb_data:/app/ailb_memory \
  marchproxy/ailb:latest
```

### Local Development

```bash
# Install dependencies
pip install -r requirements.txt

# Generate gRPC code
python -m grpc_tools.protoc \
  -I../proto \
  --python_out=./proto \
  --grpc_python_out=./proto \
  ../proto/marchproxy/module_service.proto

# Run the server
python main.py
```

## Usage Examples

### Basic Chat Request

```python
import requests

response = requests.post(
    "http://localhost:8080/v1/chat/completions",
    json={
        "model": "gpt-3.5-turbo",
        "messages": [
            {"role": "user", "content": "What is the capital of France?"}
        ]
    }
)

print(response.json())
```

### With Session Memory

```python
# First message
response1 = requests.post(
    "http://localhost:8080/v1/chat/completions",
    json={
        "model": "gpt-3.5-turbo",
        "messages": [
            {"role": "user", "content": "My name is Alice"}
        ],
        "session_id": "user-123"
    }
)

# Second message - will have context from first
response2 = requests.post(
    "http://localhost:8080/v1/chat/completions",
    json={
        "model": "gpt-3.5-turbo",
        "messages": [
            {"role": "user", "content": "What is my name?"}
        ],
        "session_id": "user-123"
    }
)
```

### With RAG Context

```python
response = requests.post(
    "http://localhost:8080/v1/chat/completions",
    json={
        "model": "gpt-4",
        "messages": [
            {"role": "user", "content": "Tell me about our company policies"}
        ],
        "rag_collection": "company_docs",
        "rag_top_k": 5
    }
)
```

## Integration with MarchProxy NLB

The AILB container is designed to work seamlessly with the MarchProxy NLB:

1. **Service Registration**: AILB registers with NLB via ModuleService gRPC
2. **Health Monitoring**: NLB polls AILB status via GetStatus()
3. **Metrics Collection**: NLB retrieves metrics via GetMetrics()
4. **Traffic Control**: NLB can adjust routing via SetTrafficWeight()
5. **Blue/Green Deployments**: Multiple AILB instances with traffic splitting

## Monitoring

### Prometheus Metrics

Available at `/metrics` endpoint:

- Request counts by provider
- Success/failure rates
- Average latency by provider
- Active connections
- Token usage (if providers support it)

### Health Checks

The `/healthz` endpoint returns:
- Overall health status
- Individual provider health
- Memory/RAG system status

## Troubleshooting

### Proto Files Not Generated

If you see "Proto files not generated, skipping gRPC server startup":

```bash
python -m grpc_tools.protoc \
  -I../proto \
  --python_out=./proto \
  --grpc_python_out=./proto \
  ../proto/marchproxy/module_service.proto
```

### Provider Connection Failures

Check logs for specific provider errors:
- OpenAI: Verify API key and quota
- Anthropic: Verify API key and model access
- Ollama: Ensure Ollama server is running and accessible

### Memory/RAG Issues

- Ensure sufficient disk space for ChromaDB
- Check permissions on data directories
- Verify sentence-transformers model downloads

## License

Limited AGPL3 with fair use preamble - See LICENSE file in repository root.

## Support

For issues and questions, see the main MarchProxy documentation.

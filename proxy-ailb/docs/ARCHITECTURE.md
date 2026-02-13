# AILB Architecture Documentation

## System Overview

The AILB (AI Load Balancer) is a specialized proxy container designed to intelligently route requests to multiple LLM providers while maintaining conversation context and supporting Retrieval-Augmented Generation (RAG). It integrates seamlessly with the MarchProxy platform through gRPC-based service communication.

## Overall System Design

```
┌─────────────────────────────────────────────────────────────┐
│                    NLB (Network Load Balancer)               │
│                  Routes traffic to AILB cluster              │
└────────────────────────┬────────────────────────────────────┘
                         │
         ┌───────────────┼───────────────┐
         │               │               │
         v               v               v
    ┌─────────┐    ┌─────────┐    ┌─────────┐
    │  AILB 1 │    │  AILB 2 │    │  AILB 3 │  (Multiple instances)
    └────┬────┘    └────┬────┘    └────┬────┘
         │              │              │
         └──────────────┼──────────────┘
                        │
         ┌──────────────┼──────────────┐
         │              │              │
         v              v              v
    ┌─────────┐   ┌──────────┐  ┌────────────┐
    │ OpenAI  │   │Anthropic │  │  Ollama    │
    └─────────┘   └──────────┘  └────────────┘
```

## Module Architecture

### Core Components

#### 1. **FastAPI HTTP Server** (Port 8080)
- **Responsibility**: OpenAI-compatible API endpoint for client requests
- **Key Endpoints**:
  - `POST /v1/chat/completions` - Chat completion requests
  - `GET /v1/models` - Available model listing
  - `GET /api/routing/stats` - Routing statistics
  - `GET /healthz` - Health check
- **Features**:
  - Request parsing and validation
  - Session context injection
  - RAG collection parameter handling
  - Response streaming support

#### 2. **gRPC ModuleService** (Port 50051)
- **Responsibility**: Service integration with MarchProxy orchestration
- **Key Methods**:
  - `GetStatus()` - Module health and operational status
  - `GetMetrics()` - Performance and routing metrics
  - `SetTrafficWeight()` - Blue/green deployment control
  - `ApplyRateLimit()` - Rate limiting configuration
  - `GetRoutes()` - Active route information
  - `Reload()` - Configuration reloading
- **Implementation**: Implements MarchProxy `ModuleService` protocol

#### 3. **Intelligent Router** (`app/router/`)
- **Responsibility**: Provider selection and request routing logic
- **Key Features**:
  - Multiple routing strategies (round-robin, latency-optimized, cost-optimized, load-balanced, failover, random)
  - Provider capability matching
  - Latency tracking per provider
  - Automatic failover on provider failure
  - Cost calculation integration
- **Data Flow**: Request metadata → Strategy evaluation → Provider selection

#### 4. **Memory Manager** (`app/memory/`)
- **Responsibility**: Conversation context persistence and retrieval
- **Storage Backend**: ChromaDB
- **Key Features**:
  - Session-based context storage
  - Vector similarity search for relevant history
  - Context window management
  - Automatic cleanup of stale sessions
- **Integration**: Injected into requests via session_id parameter

#### 5. **RAG Manager** (`app/rag/`)
- **Responsibility**: Knowledge base integration and context enrichment
- **Storage Backend**: ChromaDB
- **Key Features**:
  - Multi-collection support
  - Semantic search across documents
  - Configurable top-k result retrieval
  - Context injection into LLM prompts
- **Integration**: Activated via rag_collection and rag_top_k parameters

#### 6. **Provider Abstraction Layer** (`app/providers/`)
- **Responsibility**: Unified interface to heterogeneous LLM providers
- **Supported Providers**:
  - OpenAI (GPT-4, GPT-3.5-turbo)
  - Anthropic (Claude 3 series)
  - Ollama (Local LLMs)
- **Key Features**:
  - Provider-specific API adaptations
  - Error handling and fallback logic
  - Token counting (when supported)
  - Rate limit awareness

#### 7. **Security & Authentication** (`app/security/`, `app/auth/`)
- **Responsibility**: Request authentication and authorization
- **Features**:
  - API key validation
  - Token-based authentication
  - Request signing and verification
  - Rate limiting per user/key

#### 8. **Billing & Key Management** (`app/billing/`, `app/keys/`)
- **Responsibility**: Usage tracking and key lifecycle management
- **Features**:
  - Token usage tracking
  - Cost calculation by provider
  - API key generation and rotation
  - Usage quota enforcement

#### 9. **Rate Limiting** (`app/ratelimit/`)
- **Responsibility**: Request throttling and quota enforcement
- **Features**:
  - Per-user rate limits
  - Per-provider limits
  - Token-based bucket algorithm
  - Distributed rate limiting support

## Data Flow

### Request Processing Flow

```
Client Request (HTTP)
        │
        v
FastAPI HTTP Handler
        │
        ├─→ Authentication (/app/security)
        │
        ├─→ Rate Limit Check (/app/ratelimit)
        │
        ├─→ Session Context Retrieval (/app/memory)
        │
        ├─→ RAG Context Enrichment (/app/rag)
        │
        ├─→ Provider Selection (/app/router)
        │
        v
Provider API Call (/app/providers)
        │
        ├─→ Format conversion
        │
        ├─→ API request execution
        │
        └─→ Response parsing
        │
        v
Response Processing
        │
        ├─→ Session context update
        │
        ├─→ Usage tracking (/app/billing)
        │
        ├─→ Format conversion to OpenAI spec
        │
        v
Return to Client
```

### Service Registration Flow

```
AILB Instance Startup
        │
        v
gRPC Server Initialization
        │
        v
Register with MarchProxy (via GetStatus)
        │
        v
NLB Health Polling
        │
        ├─→ Periodic GetStatus() calls
        │
        ├─→ Metrics collection via GetMetrics()
        │
        └─→ Traffic adjustment via SetTrafficWeight()
```

## Component Interactions

### HTTP → Router → Provider

```
Request with model="gpt-4"
        │
        v
Router Selection Logic
        │
        ├─→ Evaluate routing strategy
        │
        ├─→ Check provider capabilities
        │
        ├─→ Consider latency metrics
        │
        └─→ Apply cost optimization (if configured)
        │
        v
Select Provider (e.g., OpenAI)
        │
        v
Format & Send Request
        │
        v
Receive & Parse Response
```

### Session Memory Integration

```
HTTP Request (with session_id)
        │
        v
Memory Manager Lookup
        │
        ├─→ Find previous conversations
        │
        ├─→ Retrieve relevant context
        │
        └─→ Inject into prompt
        │
        v
Provider API Call
        │
        v
Store Response in Memory
        │
        ├─→ Vector embedding
        │
        ├─→ Session association
        │
        └─→ ChromaDB storage
```

### RAG Context Enrichment

```
HTTP Request (with rag_collection, rag_top_k)
        │
        v
RAG Manager Query
        │
        ├─→ Semantic search in collection
        │
        ├─→ Retrieve top-k documents
        │
        └─→ Rank by relevance
        │
        v
Inject Context into Prompt
        │
        v
Provider API Call with Enhanced Context
```

## Configuration Architecture

### Environment Variables

**Server Configuration**:
- `HTTP_PORT`: FastAPI listening port (default: 8080)
- `HOST`: Bind address (default: 0.0.0.0)
- `GRPC_PORT`: gRPC service port (default: 50051)
- `MODULE_ID`: Unique identifier for this instance

**Routing Configuration**:
- `ROUTING_STRATEGY`: Selection algorithm (round_robin|cost_optimized|latency_optimized|load_balanced|failover|random)

**Feature Flags**:
- `ENABLE_MEMORY`: Enable conversation memory (default: true)
- `ENABLE_RAG`: Enable RAG support (default: false)
- `MEMORY_BACKEND`: Storage backend for memory (default: chromadb)
- `RAG_BACKEND`: Storage backend for RAG (default: chromadb)

**Provider Credentials**:
- `OPENAI_API_KEY`: OpenAI authentication
- `OPENAI_BASE_URL`: Custom OpenAI endpoint (optional)
- `OPENAI_MODELS`: Comma-separated available models
- `ANTHROPIC_API_KEY`: Anthropic authentication
- `ANTHROPIC_MODELS`: Comma-separated available models
- `OLLAMA_BASE_URL`: Ollama server endpoint

## Storage Architecture

### ChromaDB Integration

- **Location**: `/app/ailb_memory` (Docker volume mount)
- **Collections**:
  - `conversations`: Session-based conversation history
  - `rag_documents`: Knowledge base documents (per collection)
- **Embedding Model**: sentence-transformers (auto-downloaded)
- **Persistence**: Disk-backed SQLite with vector index

## API Contract

### OpenAI-Compatible Endpoint

```json
POST /v1/chat/completions
{
  "model": "gpt-4",
  "messages": [{"role": "user", "content": "..."}],
  "session_id": "optional-session-id",
  "rag_collection": "optional-collection",
  "rag_top_k": 3,
  "temperature": 0.7,
  "max_tokens": 1000
}
```

### gRPC ModuleService Contract

```proto
service ModuleService {
  rpc GetStatus(Empty) returns (ModuleStatus);
  rpc GetMetrics(Empty) returns (MetricsResponse);
  rpc SetTrafficWeight(TrafficWeightRequest) returns (Empty);
  rpc ApplyRateLimit(RateLimitRequest) returns (Empty);
  rpc GetRoutes(Empty) returns (RoutesResponse);
  rpc Reload(ReloadRequest) returns (Empty);
}
```

## Deployment Patterns

### Single Instance

```
Client → HTTP → AILB → Providers
              └─→ gRPC → MarchProxy NLB
```

### Clustered Deployment (Blue/Green)

```
NLB
 ├─→ AILB (Blue, 80% traffic)
 │   └─→ Providers
 └─→ AILB (Green, 20% traffic)
     └─→ Providers

Traffic adjustment via SetTrafficWeight()
```

### High Availability

```
NLB
 ├─→ AILB-1 (Zone A)
 ├─→ AILB-2 (Zone B)
 └─→ AILB-3 (Zone C)

Health checks via GetStatus()
Automatic failover on provider failure
```

## Performance Considerations

### Request Path Optimization

1. **FastAPI Async**: Non-blocking HTTP request handling
2. **Connection Pooling**: Persistent provider API connections
3. **Memory Caching**: Frequent context in-memory LRU cache
4. **Batching**: Support for bulk requests to providers
5. **Streaming**: Chunked response streaming for long outputs

### Latency Sources

- Provider API call: 100-500ms (dominant)
- Memory lookup: 10-50ms
- RAG search: 50-200ms
- Routing decision: <1ms
- Authentication/rate-limit: <5ms

## Extension Points

### Adding New Providers

1. Create provider class in `app/providers/`
2. Implement standard provider interface
3. Register in provider registry
4. Update environment variables
5. Test with routing strategy

### Custom Routing Strategies

1. Create strategy class in `app/router/`
2. Implement routing interface
3. Register in router factory
4. Configure via `ROUTING_STRATEGY` env var
5. Benchmark and validate

### Memory Backends

1. Implement ChromaDB-compatible interface
2. Place in `app/memory/backends/`
3. Update configuration
4. Test session persistence
5. Validate vector search accuracy

## Monitoring & Observability

### Metrics Exposed

- Request count per provider
- Success/error rates
- Latency percentiles (p50, p95, p99)
- Active connections
- Token usage (if supported)
- Cache hit rates
- Memory usage

### Health Check Signals

- HTTP endpoint availability
- gRPC server responsiveness
- Provider connectivity
- Memory/RAG system status
- Disk space for ChromaDB

### Logging Points

- Authentication events
- Provider selection rationale
- Failover triggers
- Configuration reloads
- Error conditions with context

## Security Model

### Authentication Layers

1. **API Key Validation**: Request header `Authorization: Bearer <key>`
2. **Rate Limiting**: Per-key and per-user throttling
3. **Audit Logging**: Authentication and high-risk operations
4. **Credential Isolation**: Provider credentials never exposed to clients

### Provider Isolation

- Each provider connection uses isolated credentials
- No credential mixing between providers
- Rate limits enforced independently per provider
- Fallback doesn't reveal provider details to clients

## Error Handling Strategy

### Provider Failures

```
Request to Provider
        │
        ├─→ Connection timeout → Try next provider
        │
        ├─→ Rate limit (429) → Queue and retry
        │
        ├─→ Authentication (401) → Log and alert
        │
        └─→ Server error (5xx) → Failover to next
```

### Graceful Degradation

- Memory unavailable: Continue without context
- RAG unavailable: Continue without enrichment
- All providers down: Return service unavailable
- Configuration reload failure: Keep previous config

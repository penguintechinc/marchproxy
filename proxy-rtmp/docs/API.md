# proxy-rtmp API Documentation

## Overview

proxy-rtmp exposes two primary APIs for integration with the MarchProxy architecture:

1. **RTMP Protocol API** - Native RTMP server for stream ingestion
2. **gRPC ModuleService API** - Service implementation for NLB communication

## RTMP Protocol API

### Server Details

- **Protocol**: RTMP (Real Time Messaging Protocol)
- **Default Port**: 1935
- **Configurable**: Yes (via `RTMP_HOST`, `RTMP_PORT`)

### RTMP Publishing

Clients publish streams to the RTMP server using standard streaming applications.

#### Publication Endpoint

```
rtmp://<host>:<port>/live/<stream_key>
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `host` | string | Yes | RTMP server hostname/IP |
| `port` | int | Yes | RTMP server port (default: 1935) |
| `stream_key` | string | Yes | Unique identifier for the stream |

#### Example Publishing Commands

**FFmpeg**
```bash
ffmpeg -re -i input.mp4 \
  -c:v libx264 -preset veryfast -b:v 6000k \
  -c:a aac -b:a 192k \
  -f flv rtmp://localhost:1935/live/my_stream
```

**OBS Studio**
1. Settings → Stream
2. Service: Custom
3. Server: `rtmp://your-server:1935/live`
4. Stream Key: `my_stream`

**GStreamer**
```bash
gst-launch-1.0 filesrc location=input.mp4 ! decodebin ! \
  x264enc ! queue ! flvmux ! rtmpsink location="rtmp://localhost:1935/live/my_stream"
```

### Adaptive Stream Outputs

After a stream is published, proxy-rtmp automatically generates adaptive bitrate versions:

#### HLS Output

```
http://<server>/streams/<stream_key>/master.m3u8
```

**Variants** (automatically generated):
- 1080p30: 5000 Kbps (1920x1080)
- 720p30: 3000 Kbps (1280x720)
- 480p30: 1500 Kbps (854x480)
- 360p30: 800 Kbps (640x360)

#### DASH Output

```
http://<server>/streams/<stream_key>/dash/manifest.mpd
```

**Profiles**: Same as HLS (auto-adaptive by player)

## gRPC ModuleService API

### Service Definition

proxy-rtmp implements the `ModuleService` for integration with MarchProxy NLB:

```protobuf
service ModuleService {
  rpc GetStatus(Empty) returns (StatusResponse);
  rpc GetRoutes(Empty) returns (RoutesResponse);
  rpc GetMetrics(Empty) returns (MetricsResponse);
  rpc HealthCheck(Empty) returns (HealthCheckResponse);
}
```

### Server Details

- **Protocol**: gRPC
- **Default Port**: 50053
- **Configurable**: Yes (via `RTMP_GRPC_PORT`)
- **TLS**: No (internal communication only)

### Endpoints

#### GetStatus

Returns current status and statistics of the proxy module.

**Request**
```protobuf
Empty {}
```

**Response**
```protobuf
message StatusResponse {
  string module_name = 1;      // "rtmp-proxy"
  string version = 2;           // e.g., "1.0.0"
  string status = 3;            // "healthy", "degraded", "unhealthy"
  int32 active_streams = 4;     // Number of active streams
  int64 total_bytes_in = 5;     // Total input bytes
  int64 total_bytes_out = 6;    // Total output bytes
  string uptime = 7;            // Human-readable uptime
}
```

**Example Response**
```json
{
  "module_name": "rtmp-proxy",
  "version": "1.0.0",
  "status": "healthy",
  "active_streams": 3,
  "total_bytes_in": 1073741824,
  "total_bytes_out": 536870912,
  "uptime": "2h45m30s"
}
```

#### GetRoutes

Returns information about all active streams.

**Request**
```protobuf
Empty {}
```

**Response**
```protobuf
message RoutesResponse {
  repeated StreamRoute routes = 1;
}

message StreamRoute {
  string stream_key = 1;              // Stream identifier
  string source = 2;                  // Input source (rtmp://...)
  string status = 3;                  // "active", "idle", "error"
  string encoder = 4;                 // Encoder type (x264, nvenc_h264, etc.)
  int32 input_bitrate = 5;            // Input bitrate in Kbps
  int32 input_resolution = 6;         // Input resolution (height)
  int64 bytes_processed = 7;          // Bytes processed
  repeated TranscodeProfile profiles = 8;  // Output profiles
}

message TranscodeProfile {
  string name = 1;                    // e.g., "1080p", "720p"
  string codec = 2;                   // Codec (h264, h265)
  int32 bitrate = 3;                  // Bitrate in Kbps
  int32 resolution = 4;               // Output resolution (height)
  string status = 5;                  // "encoding", "complete", "error"
}
```

**Example Response**
```json
{
  "routes": [
    {
      "stream_key": "stream_001",
      "source": "rtmp://localhost:1935/live/stream_001",
      "status": "active",
      "encoder": "nvenc_h264",
      "input_bitrate": 6000,
      "input_resolution": 1080,
      "bytes_processed": 1073741824,
      "profiles": [
        {
          "name": "1080p",
          "codec": "h264",
          "bitrate": 5000,
          "resolution": 1080,
          "status": "encoding"
        },
        {
          "name": "720p",
          "codec": "h264",
          "bitrate": 3000,
          "resolution": 720,
          "status": "encoding"
        }
      ]
    }
  ]
}
```

#### GetMetrics

Returns performance metrics for all active streams.

**Request**
```protobuf
Empty {}
```

**Response**
```protobuf
message MetricsResponse {
  int32 cpu_percent = 1;              // CPU utilization percentage
  int32 gpu_percent = 2;              // GPU utilization percentage (if applicable)
  int32 memory_mb = 3;                // Memory usage in MB
  repeated StreamMetrics streams = 4; // Per-stream metrics
}

message StreamMetrics {
  string stream_key = 1;
  float fps = 2;                      // Frames per second
  float duration = 3;                 // Stream duration in seconds
  int32 dropped_frames = 4;           // Count of dropped frames
  int32 encoder_speed = 5;            // Encoding speed factor
}
```

**Example Response**
```json
{
  "cpu_percent": 45,
  "gpu_percent": 35,
  "memory_mb": 512,
  "streams": [
    {
      "stream_key": "stream_001",
      "fps": 29.97,
      "duration": 3600.5,
      "dropped_frames": 2,
      "encoder_speed": 1.5
    }
  ]
}
```

#### HealthCheck

Returns health status for monitoring and load balancer integration.

**Request**
```protobuf
Empty {}
```

**Response**
```protobuf
message HealthCheckResponse {
  string status = 1;                  // "healthy", "degraded", "unhealthy"
  string message = 2;                 // Status message
  int32 response_time_ms = 3;         // Response time in milliseconds
  repeated string errors = 4;         // Active errors (if any)
}
```

**Example Response**
```json
{
  "status": "healthy",
  "message": "All systems operational",
  "response_time_ms": 5,
  "errors": []
}
```

**Degraded Example**
```json
{
  "status": "degraded",
  "message": "High CPU utilization",
  "response_time_ms": 8,
  "errors": [
    "CPU usage at 95%",
    "GPU memory pressure high"
  ]
}
```

## Client Examples

### Go gRPC Client

```go
package main

import (
	"context"
	"log"

	pb "github.com/penguintech/marchproxy/proto/module"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50053", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewModuleServiceClient(conn)
	ctx := context.Background()

	// Get status
	status, err := client.GetStatus(ctx, &pb.Empty{})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	log.Printf("Status: %+v\n", status)

	// Get routes
	routes, err := client.GetRoutes(ctx, &pb.Empty{})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	log.Printf("Routes: %+v\n", routes)

	// Get metrics
	metrics, err := client.GetMetrics(ctx, &pb.Empty{})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	log.Printf("Metrics: %+v\n", metrics)

	// Health check
	health, err := client.HealthCheck(ctx, &pb.Empty{})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	log.Printf("Health: %+v\n", health)
}
```

### Python gRPC Client

```python
import grpc
from marchproxy.proto import module_pb2, module_pb2_grpc

def get_status():
    with grpc.insecure_channel('localhost:50053') as channel:
        stub = module_pb2_grpc.ModuleServiceStub(channel)
        response = stub.GetStatus(module_pb2.Empty())
        return response

if __name__ == '__main__':
    status = get_status()
    print(f"Module: {status.module_name}")
    print(f"Status: {status.status}")
    print(f"Active streams: {status.active_streams}")
```

### JavaScript/TypeScript Client

```typescript
import * as grpc from '@grpc/grpc-js';
import { ModuleServiceClient } from './proto/module_grpc_pb';
import { Empty } from './proto/module_pb';

const client = new ModuleServiceClient(
  'localhost:50053',
  grpc.credentials.createInsecure()
);

client.getStatus(new Empty(), (err, response) => {
  if (err) {
    console.error('Error:', err);
    return;
  }
  console.log('Status:', response.toObject());
});
```

## Error Handling

### gRPC Status Codes

| Code | Meaning | Common Cause |
|------|---------|--------------|
| `OK` (0) | Success | No error |
| `INVALID_ARGUMENT` (3) | Invalid argument | Bad parameters |
| `DEADLINE_EXCEEDED` (4) | Timeout | Request took too long |
| `NOT_FOUND` (5) | Not found | Stream not found |
| `RESOURCE_EXHAUSTED` (8) | Resource exhausted | Max streams reached |
| `UNAVAILABLE` (14) | Unavailable | Server not ready |
| `INTERNAL` (13) | Internal error | Server error |

### RTMP Error Responses

RTMP server closes connections with appropriate status on error:

| Status | Meaning |
|--------|---------|
| 400 | Bad request (invalid RTMP format) |
| 403 | Forbidden (stream key mismatch, max streams reached) |
| 404 | Not found (stream doesn't exist) |
| 500 | Internal error (transcoding failure) |

## Rate Limiting

- **Max Concurrent Streams**: Configurable via `RTMP_MAX_STREAMS` (default: 100)
- **Max Bitrate per Stream**: Configurable via `RTMP_MAX_BITRATE` Mbps (default: 10)
- **Max Resolution**: Configurable via `RTMP_MAX_RESOLUTION` (default: 1080p)

## Authentication

Current version supports:
- **No Authentication**: Open RTMP endpoint (intended for internal/trusted networks)
- **Future Enhancement**: Token-based authentication planned

For production deployments, restrict network access using firewall rules or network policies.

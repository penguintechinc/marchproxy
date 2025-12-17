# proxy-rtmp Testing Guide

## Overview

This document provides comprehensive testing procedures for proxy-rtmp, including unit tests, integration tests, and performance benchmarks.

## Test Structure

```
proxy-rtmp/
├── internal/
│   ├── rtmp/
│   │   └── *_test.go
│   ├── transcode/
│   │   └── *_test.go
│   ├── config/
│   │   └── *_test.go
│   └── grpc/
│       └── *_test.go
├── tests/
│   ├── integration/
│   │   ├── rtmp_stream_test.go
│   │   ├── grpc_api_test.go
│   │   ├── transcoding_test.go
│   │   └── performance_test.go
│   └── fixtures/
│       ├── sample_video.mp4
│       └── config_test.yaml
└── cmd/rtmp/
    └── main_test.go
```

## Running Tests

### Unit Tests

Run all unit tests:

```bash
go test ./...
```

Run tests in specific package:

```bash
go test ./internal/config
go test ./internal/rtmp
go test ./internal/transcode
go test ./internal/grpc
```

Run with coverage:

```bash
go test -cover ./...
```

Generate coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests

Run integration tests:

```bash
go test -tags=integration ./tests/integration
```

Run specific test:

```bash
go test -run TestRTMPStreamPublishing -v
go test -run TestGRPCAPIStatus -v
```

### Long-Running Tests

Run only short tests (excludes integration/performance):

```bash
go test -short ./...
```

Run full test suite including performance:

```bash
go test -timeout 30m ./...
```

## Test Categories

### 1. Configuration Tests

**File**: `internal/config/config_test.go`

Tests configuration loading and validation:

- Load from environment variables
- Load from YAML file
- Load from command-line defaults
- Configuration validation
- Invalid encoder/preset rejection
- Port range validation

**Run**:
```bash
go test ./internal/config -v
```

**Expected Coverage**: >90%

### 2. RTMP Server Tests

**File**: `internal/rtmp/server_test.go`

Tests RTMP protocol functionality:

- Server startup and shutdown
- Client connection handling
- Stream publishing
- Stream key validation
- Session management
- Concurrent stream limits
- Graceful disconnection

**Run**:
```bash
go test ./internal/rtmp -v
```

**Expected Coverage**: >85%

### 3. Transcoding Tests

**File**: `internal/transcode/detector_test.go`, `internal/transcode/ffmpeg_test.go`

Tests encoder detection and FFmpeg management:

- GPU detection (NVIDIA, AMD)
- Encoder selection logic
- Fallback to CPU on missing GPU
- FFmpeg process lifecycle
- Output format generation (HLS, DASH)
- Bitrate/resolution constraints

**Run**:
```bash
go test ./internal/transcode -v
```

**Expected Coverage**: >80%

### 4. gRPC API Tests

**File**: `internal/grpc/server_test.go`

Tests ModuleService implementation:

- GetStatus endpoint
- GetRoutes endpoint
- GetMetrics endpoint
- HealthCheck endpoint
- Error handling
- Concurrent client handling

**Run**:
```bash
go test ./internal/grpc -v
```

**Expected Coverage**: >85%

### 5. Integration Tests

**File**: `tests/integration/*.go`

End-to-end tests combining all components:

- Full RTMP publishing flow
- Transcoding pipeline
- HLS/DASH output generation
- gRPC API availability
- Stream playback verification
- Resource cleanup

**Run**:
```bash
go test -tags=integration ./tests/integration -v
```

**Expected Coverage**: >80% of public APIs

### 6. Performance Tests

**File**: `tests/integration/performance_test.go`

Benchmarks for performance-critical paths:

- RTMP connection throughput
- Transcoding latency (various encoders)
- Memory usage under load
- FFmpeg process startup time
- gRPC API response time

**Run**:
```bash
go test -run=BenchmarkRTMPPublish -bench=. -benchtime=10s
go test -run=BenchmarkTranscode -bench=. -benchtime=30s
```

## Mock Testing

### Mocking FFmpeg

For unit tests, FFmpeg is mocked to avoid dependencies:

```go
// Mock FFmpeg manager for testing
type MockFFmpegManager struct {
    mock.Mock
}

func (m *MockFFmpegManager) Transcode(stream *rtmp.Stream) error {
    return m.Called(stream).Error(0)
}
```

### Mocking gRPC

For testing gRPC client/server:

```go
// Test gRPC service
func TestGetStatus(t *testing.T) {
    mockServer := NewMockModuleService()
    conn := grpc.Dial(":50053")
    defer conn.Close()

    client := pb.NewModuleServiceClient(conn)
    resp, err := client.GetStatus(context.Background(), &pb.Empty{})

    assert.NoError(t, err)
    assert.Equal(t, "healthy", resp.Status)
}
```

## Continuous Integration Testing

### GitHub Actions Workflow

Tests automatically run on every push:

```yaml
name: Unit Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - run: go test -cover -race ./...
```

### Coverage Requirements

- **Minimum coverage**: 80%
- **Critical paths**: 95% (RTMP, gRPC, config)
- **Utility functions**: 70%

## Testing Fixtures

### Sample Video Files

Located in `tests/fixtures/`:

- `sample_video.mp4` (1080p, 30fps, 30 seconds)
- `sample_audio.aac` (stereo, 48kHz)

Generate test video:

```bash
ffmpeg -f lavfi -i testsrc=s=1920x1080:d=30 \
       -f lavfi -i sine=f=440:d=30 \
       -c:v libx264 -c:a aac \
       tests/fixtures/sample_video.mp4
```

### Configuration Fixtures

Located in `tests/fixtures/`:

- `config_default.yaml` - Default configuration
- `config_gpu.yaml` - GPU-enabled configuration
- `config_low_latency.yaml` - Low-latency preset
- `config_invalid.yaml` - Invalid configurations for error testing

## Manual Testing

### Test Publishing with FFmpeg

```bash
# Start RTMP server
go run ./cmd/rtmp/main.go --port 1935

# In another terminal, publish stream
ffmpeg -re -i tests/fixtures/sample_video.mp4 \
       -c:v libx264 -preset veryfast -b:v 6000k \
       -c:a aac -b:a 192k \
       -f flv rtmp://localhost:1935/live/test_stream
```

### Test Publishing with OBS

1. Start proxy-rtmp server
2. Open OBS Studio
3. Settings → Stream:
   - Service: Custom
   - Server: `rtmp://localhost:1935/live`
   - Stream Key: `obs_test`
4. Start streaming

### Test Publishing with GStreamer

```bash
gst-launch-1.0 -v filesrc location=tests/fixtures/sample_video.mp4 ! \
  decodebin ! \
  video/x-raw ! videoscale ! video/x-raw,width=1920,height=1080 ! \
  x264enc ! queue ! \
  flvmux name=mux ! \
  rtmpsink location="rtmp://localhost:1935/live/gst_test"
```

### Test HLS Playback

After publishing stream:

```bash
# Verify HLS master playlist exists
curl http://localhost/streams/test_stream/master.m3u8

# Play in VLC
open http://localhost/streams/test_stream/master.m3u8

# Play with ffplay
ffplay http://localhost/streams/test_stream/master.m3u8
```

### Test DASH Playback

```bash
# Verify DASH manifest exists
curl http://localhost/streams/test_stream/dash/manifest.mpd

# Play in VLC
open http://localhost/streams/test_stream/dash/manifest.mpd
```

### Test gRPC API

Using gRPC client:

```bash
# Using grpcurl
grpcurl -plaintext localhost:50053 marchproxy.module.ModuleService/GetStatus

# Using go client
go run ./examples/grpc_client.go
```

## Load Testing

### Stress Test: Multiple Streams

```bash
#!/bin/bash
# Start 10 concurrent streams
for i in {1..10}; do
  ffmpeg -re -i tests/fixtures/sample_video.mp4 \
         -c:v libx264 -preset veryfast \
         -f flv rtmp://localhost:1935/live/stream_$i &
done
wait
```

### Stress Test: Bitrate Limits

```bash
# Test bitrate constraint
ffmpeg -re -i tests/fixtures/sample_video.mp4 \
       -b:v 50m \
       -f flv rtmp://localhost:1935/live/high_bitrate

# Should be limited to configured max (e.g., 10 Mbps)
```

### Memory Leak Testing

```bash
# Monitor memory usage during long streams
watch -n 1 'ps aux | grep rtmp-proxy | grep -v grep'

# Or with valgrind (if built with debug symbols)
valgrind --leak-check=full ./rtmp-proxy
```

## Debugging

### Debug Logging

Run with debug log level:

```bash
RTMP_LOG_LEVEL=debug go run ./cmd/rtmp/main.go
```

### FFmpeg Debug Output

Enable FFmpeg debug logging:

```bash
RTMP_FFMPEG_DEBUG=1 go run ./cmd/rtmp/main.go
```

### Go Race Detector

Run tests with race detector:

```bash
go test -race ./...
```

### Profiling

Generate CPU profile:

```bash
go test -cpuprofile=cpu.prof -bench=Benchmark ./tests
go tool pprof cpu.prof
```

Generate memory profile:

```bash
go test -memprofile=mem.prof -bench=Benchmark ./tests
go tool pprof mem.prof
```

## Test Coverage Goals

| Component | Coverage | Notes |
|-----------|----------|-------|
| Config | >95% | Critical path, all validators tested |
| RTMP Server | >85% | Protocol handling, connection mgmt |
| Transcoding | >80% | GPU detection, encoder selection |
| gRPC API | >90% | All endpoints and error cases |
| Integration | >75% | End-to-end flows |
| Overall | >80% | Minimum requirement |

## Known Limitations

1. **GPU Testing**: Requires actual GPU hardware; skipped in CI
2. **FFmpeg**: Some tests require ffmpeg binary in PATH
3. **Network**: Integration tests use localhost only
4. **Performance**: Benchmarks may vary by system specs

## Troubleshooting Tests

### "ffmpeg not found"
```bash
# Install ffmpeg
apt-get install ffmpeg        # Debian/Ubuntu
brew install ffmpeg           # macOS
```

### "Connection refused" on RTMP test
Ensure server is running before starting client:
```bash
go run ./cmd/rtmp/main.go &   # Start server
sleep 1                        # Wait for startup
go test -run TestRTMP ./tests
```

### Test timeout
Increase timeout:
```bash
go test -timeout 5m ./tests/integration
```

### GPU tests skipped
Tests automatically skip if no GPU detected:
```bash
SKIP_GPU_TESTS=0 go test ./tests
```

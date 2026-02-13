# proxy-rtmp Configuration Guide

## Overview

proxy-rtmp supports flexible configuration through multiple methods with the following priority:

1. **Command-line arguments** (highest priority)
2. **Environment variables**
3. **Configuration file** (`/etc/marchproxy/rtmp.yaml`)
4. **Built-in defaults** (lowest priority)

## Configuration Methods

### 1. Environment Variables

All configuration options are available as environment variables with the `RTMP_` prefix.

```bash
export RTMP_HOST=0.0.0.0
export RTMP_PORT=1935
export RTMP_ENCODER=nvenc_h264
export RTMP_PRESET=medium
export RTMP_MAX_STREAMS=100
```

Full list of environment variables:

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `RTMP_HOST` | string | `0.0.0.0` | RTMP server bind address |
| `RTMP_PORT` | int | `1935` | RTMP server port |
| `RTMP_GRPC_PORT` | int | `50053` | gRPC server port |
| `RTMP_LOG_LEVEL` | string | `info` | Log level: debug, info, warn, error |
| `RTMP_ENCODER` | string | `auto` | Video encoder selection |
| `RTMP_PRESET` | string | `medium` | Encoding preset/speed |
| `RTMP_OUTPUT_DIR` | string | `/var/lib/marchproxy/streams` | Output directory for streams |
| `RTMP_ENABLE_HLS` | bool | `true` | Enable HLS output |
| `RTMP_ENABLE_DASH` | bool | `true` | Enable DASH output |
| `RTMP_SEGMENT_DURATION` | int | `6` | HLS/DASH segment duration (seconds) |
| `RTMP_MAX_BITRATE` | int | `10` | Max bitrate per stream (Mbps) |
| `RTMP_MAX_STREAMS` | int | `100` | Max concurrent streams |
| `RTMP_MAX_RESOLUTION` | int | `1080` | Max resolution height (pixels) |
| `RTMP_FFMPEG_PATH` | string | `ffmpeg` | Path to ffmpeg binary |
| `RTMP_FFPROBE_PATH` | string | `ffprobe` | Path to ffprobe binary |
| `RTMP_HEALTH_CHECK_INTERVAL` | int | `30` | Health check interval (seconds) |

### 2. Configuration File

Create `/etc/marchproxy/rtmp.yaml`:

```yaml
# Server configuration
host: 0.0.0.0
port: 1935
grpc-port: 50053

# Logging
log-level: info

# Encoder settings
encoder: auto          # auto, x264, x265, nvenc_h264, nvenc_h265, amf_h264, amf_h265
preset: medium         # ultrafast, fast, medium, slow

# Output configuration
output-dir: /var/lib/marchproxy/streams
enable-hls: true
enable-dash: true
segment-duration: 6

# Rate limiting
max-bitrate: 10        # Mbps per stream
max-streams: 100       # Concurrent streams
max-resolution: 1080   # Height in pixels

# FFmpeg paths
ffmpeg-path: ffmpeg
ffprobe-path: ffprobe

# Advanced encoder parameters
encoder-params:
  tune: zerolatency
  profile: high

# Health check
health-check-interval: 30
```

### 3. Command-line Arguments

```bash
./rtmp-proxy \
  --host 0.0.0.0 \
  --port 1935 \
  --grpc-port 50053 \
  --log-level info \
  --encoder nvenc_h264 \
  --preset medium \
  --output-dir /var/lib/marchproxy/streams \
  --enable-hls \
  --enable-dash \
  --segment-duration 6 \
  --config /etc/marchproxy/rtmp.yaml
```

## Encoder Configuration

### Automatic Selection (`encoder: auto`)

proxy-rtmp automatically detects available hardware and selects the optimal encoder:

1. **NVIDIA GPU detected** → `nvenc_h264`
2. **AMD GPU detected** → `amf_h264`
3. **No GPU** → `x264` (CPU software encoder)

### Manual Selection

Override automatic selection:

```yaml
encoder: nvenc_h264    # Force NVIDIA NVENC H.264
```

Valid encoder values:

| Encoder | GPU | Codec | Variants |
|---------|-----|-------|----------|
| `x264` | None | H.264 | AVC, Profile High |
| `x265` | None | H.265 | HEVC, Profile Main |
| `nvenc_h264` | NVIDIA | H.264 | AVC, NVIDIA-optimized |
| `nvenc_h265` | NVIDIA | H.265 | HEVC, NVIDIA-optimized |
| `amf_h264` | AMD | H.264 | AVC, AMD-optimized |
| `amf_h265` | AMD | H.265 | HEVC, AMD-optimized |

### Encoding Presets

Control encoding speed vs. quality trade-off:

| Preset | Speed | Quality | CPU/GPU | Use Case |
|--------|-------|---------|---------|----------|
| `ultrafast` | Fastest | Lowest | Minimum | Ultra-low latency |
| `superfast` | Very Fast | Low | Low | Low-latency streaming |
| `veryfast` | Fast | Medium | Medium | Live streaming |
| `faster` | Fast | Medium-High | Medium | Balanced |
| `fast` | Medium | Good | Medium-High | Default for most |
| `medium` | Moderate | High | Medium-High | High quality (default) |
| `slow` | Slow | Very High | High | Archive/VOD |
| `slower` | Very Slow | Excellent | Very High | High-quality archival |
| `veryslow` | Slowest | Maximum | Maximum | Offline processing |

### Advanced Encoder Parameters

Configure codec-specific parameters:

```yaml
encoder-params:
  tune: zerolatency          # x264/x265: zerolatency for live
  profile: high              # x264/x265: baseline, main, high
  rc: vbr                    # NVENC: rate control (cbr, vbr, ll_2pass_quality)
  gpu: 0                     # NVENC/AMF: GPU device ID (0-based)
  hwaccel: cuda              # FFmpeg hardware acceleration
```

## Output Configuration

### Stream Segments

Configure adaptive bitrate ladder:

```yaml
# Segment duration affects latency
segment-duration: 6         # 6 seconds (default, good latency)
# Alternatives:
# segment-duration: 2       # 2 seconds (lower latency, higher overhead)
# segment-duration: 10      # 10 seconds (higher latency, lower overhead)
```

### HLS Output

```yaml
enable-hls: true            # Enable HLS output
# Output: http://server/streams/<stream_key>/master.m3u8
```

Default adaptive bitrate ladder:
- 1080p: 5000 Kbps
- 720p: 3000 Kbps
- 480p: 1500 Kbps
- 360p: 800 Kbps

### DASH Output

```yaml
enable-dash: true           # Enable DASH output
# Output: http://server/streams/<stream_key>/dash/manifest.mpd
```

Same adaptive bitrate ladder as HLS.

## Rate Limiting Configuration

### Max Bitrate

```yaml
max-bitrate: 10             # Mbps per stream (default)
```

- Limits input bitrate per stream
- Prevents single stream from consuming all bandwidth
- Recommended range: 5-50 Mbps

### Max Streams

```yaml
max-streams: 100            # Concurrent streams (default)
```

- Limits number of simultaneous streams
- Prevents resource exhaustion
- Depends on hardware: 3-5 for CPU, 20-50 for GPU

### Max Resolution

```yaml
max-resolution: 1080        # Height in pixels (default)
```

- Limits maximum input resolution
- Lower resolutions reduce CPU/GPU load
- Valid values: 360, 480, 720, 1080, 2160

## Logging Configuration

### Log Levels

```yaml
log-level: info             # debug, info, warn, error
```

| Level | Output | Use Case |
|-------|--------|----------|
| `debug` | Verbose, includes internal details | Development, troubleshooting |
| `info` | General informational messages | Production (default-like) |
| `warn` | Warning messages only | High-performance deployments |
| `error` | Errors only | Minimal logging |

### Log Format

JSON format for easy parsing:

```json
{
  "level":"info",
  "msg":"Starting MarchProxy RTMP Container",
  "time":"2024-01-15T10:30:45Z",
  "version":"1.0.0",
  "host":"0.0.0.0",
  "port":1935,
  "encoder":"nvenc_h264"
}
```

## Docker Configuration

### Environment Variables

```dockerfile
ENV RTMP_HOST=0.0.0.0
ENV RTMP_PORT=1935
ENV RTMP_GRPC_PORT=50053
ENV RTMP_ENCODER=auto
ENV RTMP_PRESET=medium
ENV RTMP_OUTPUT_DIR=/var/lib/marchproxy/streams
ENV RTMP_ENABLE_HLS=true
ENV RTMP_ENABLE_DASH=true
ENV RTMP_LOG_LEVEL=info
ENV RTMP_MAX_STREAMS=100
ENV RTMP_MAX_BITRATE=10
ENV RTMP_MAX_RESOLUTION=1080
```

### Volume Mounts

```bash
docker run -v /path/to/rtmp.yaml:/etc/marchproxy/rtmp.yaml \
           -v /var/lib/marchproxy/streams:/var/lib/marchproxy/streams \
           marchproxy-rtmp:latest
```

### Docker Compose

```yaml
version: '3.8'

services:
  rtmp:
    image: marchproxy-rtmp:latest
    ports:
      - "1935:1935"     # RTMP
      - "50053:50053"   # gRPC
      - "80:80"         # HTTP (HLS/DASH streaming)
    volumes:
      - /var/lib/marchproxy/streams:/var/lib/marchproxy/streams
      - ./rtmp.yaml:/etc/marchproxy/rtmp.yaml
    environment:
      RTMP_ENCODER: auto
      RTMP_PRESET: medium
      RTMP_LOG_LEVEL: info
      RTMP_MAX_STREAMS: 100
    healthcheck:
      test: ["CMD", "./rtmp-proxy", "--healthcheck"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Performance Tuning

### For Low Latency

```yaml
preset: superfast          # Faster encoding
segment-duration: 2        # Shorter segments (2s)
encoder-params:
  tune: zerolatency
```

Achieves: 4-6 seconds latency

### For High Quality

```yaml
preset: slow               # Better compression
segment-duration: 10       # Longer segments (10s)
encoder-params:
  tune: film
  profile: high
```

Achieves: 10-15 seconds latency, excellent quality

### For Maximum Throughput

```yaml
encoder: nvenc_h264        # Hardware acceleration
preset: fast
max-streams: 50            # Increase concurrency
max-bitrate: 25            # Higher bitrate per stream
```

### For Low Resource Usage

```yaml
encoder: x264              # CPU (vs GPU)
preset: ultrafast
segment-duration: 10
max-streams: 5             # Lower concurrency
max-bitrate: 5             # Lower bitrate
```

## Example Configurations

### Development Environment

```yaml
host: localhost
port: 1935
grpc-port: 50053
log-level: debug
encoder: auto
preset: fast
output-dir: ./streams
enable-hls: true
enable-dash: true
segment-duration: 6
max-streams: 10
max-bitrate: 50
```

### Production CPU-Only

```yaml
host: 0.0.0.0
port: 1935
grpc-port: 50053
log-level: warn
encoder: x264
preset: medium
output-dir: /var/lib/marchproxy/streams
enable-hls: true
enable-dash: false
segment-duration: 6
max-streams: 20
max-bitrate: 10
max-resolution: 720
```

### Production GPU-Accelerated

```yaml
host: 0.0.0.0
port: 1935
grpc-port: 50053
log-level: info
encoder: nvenc_h264
preset: medium
output-dir: /var/lib/marchproxy/streams
enable-hls: true
enable-dash: true
segment-duration: 6
max-streams: 100
max-bitrate: 25
encoder-params:
  tune: zerolatency
  rc: vbr
```

## Configuration Validation

proxy-rtmp validates configuration on startup:

```bash
./rtmp-proxy --config /etc/marchproxy/rtmp.yaml
```

Common validation errors:

| Error | Cause | Solution |
|-------|-------|----------|
| `invalid port: X` | Port out of range (1-65535) | Use valid port number |
| `invalid encoder: X` | Unknown encoder type | Check encoder name in docs |
| `invalid preset: X` | Unknown preset name | Use valid preset (ultra, fast, medium, slow) |
| `segment duration must be 1-60 seconds` | Out of range | Use 1-60 second duration |
| `failed to create output directory` | Permission denied | Check directory permissions |

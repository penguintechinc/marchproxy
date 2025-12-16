# MarchProxy RTMP Container

FFmpeg-based RTMP streaming container with GPU acceleration support for the MarchProxy Unified NLB Architecture.

## Features

- **RTMP Protocol Support**: Full RTMP server implementation
- **FFmpeg Transcoding**: Industry-standard video transcoding
- **CPU Encoding**: x264 (H.264) and x265 (H.265) software encoders
- **GPU Acceleration**:
  - NVIDIA NVENC (H.264/H.265)
  - AMD AMF (H.264/H.265)
- **Adaptive Streaming**:
  - HLS (HTTP Live Streaming)
  - DASH (Dynamic Adaptive Streaming over HTTP)
- **Multi-Bitrate**: Automatic adaptive bitrate ladder (1080p, 720p, 480p, 360p)
- **gRPC Integration**: ModuleService implementation for NLB communication

## Architecture

```
RTMP Input → RTMP Server → FFmpeg Transcoder → HLS/DASH Output
                ↓                  ↓
           Session Mgmt      GPU Detection
                ↓                  ↓
           gRPC Server     Encoder Selection
```

## Docker Images

Three variants available:

1. **CPU**: Software encoding (x264/x265)
   ```bash
   docker build -f Dockerfile -t marchproxy-rtmp:cpu .
   ```

2. **NVIDIA**: Hardware encoding (NVENC)
   ```bash
   docker build -f Dockerfile.nvidia -t marchproxy-rtmp:nvidia .
   ```

3. **AMD**: Hardware encoding (AMF)
   ```bash
   docker build -f Dockerfile.amd -t marchproxy-rtmp:amd .
   ```

## Usage

### CPU Version
```bash
docker run -d \
  -p 1935:1935 \
  -p 50053:50053 \
  -v /path/to/streams:/var/lib/marchproxy/streams \
  -e RTMP_ENCODER=x264 \
  marchproxy-rtmp:cpu
```

### NVIDIA GPU Version
```bash
docker run -d \
  --runtime=nvidia \
  --gpus all \
  -p 1935:1935 \
  -p 50053:50053 \
  -v /path/to/streams:/var/lib/marchproxy/streams \
  -e RTMP_ENCODER=nvenc_h264 \
  -e NVIDIA_VISIBLE_DEVICES=all \
  -e NVIDIA_DRIVER_CAPABILITIES=compute,video,utility \
  marchproxy-rtmp:nvidia
```

### AMD GPU Version
```bash
docker run -d \
  --device=/dev/kfd \
  --device=/dev/dri \
  --group-add video \
  --group-add render \
  -p 1935:1935 \
  -p 50053:50053 \
  -v /path/to/streams:/var/lib/marchproxy/streams \
  -e RTMP_ENCODER=amf_h264 \
  marchproxy-rtmp:amd
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `RTMP_HOST` | `0.0.0.0` | RTMP server host |
| `RTMP_PORT` | `1935` | RTMP server port |
| `RTMP_GRPC_PORT` | `50053` | gRPC server port |
| `RTMP_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `RTMP_ENCODER` | `auto` | Video encoder (auto, x264, x265, nvenc_h264, nvenc_h265, amf_h264, amf_h265) |
| `RTMP_PRESET` | `medium` | Encoding preset (ultrafast, fast, medium, slow) |
| `RTMP_OUTPUT_DIR` | `/var/lib/marchproxy/streams` | Output directory for streams |
| `RTMP_ENABLE_HLS` | `true` | Enable HLS output |
| `RTMP_ENABLE_DASH` | `true` | Enable DASH output |
| `RTMP_SEGMENT_DURATION` | `6` | Segment duration in seconds |
| `RTMP_MAX_BITRATE` | `10` | Max bitrate in Mbps |
| `RTMP_MAX_STREAMS` | `100` | Max concurrent streams |
| `RTMP_MAX_RESOLUTION` | `1080` | Max resolution height |

### Configuration File

Create `/etc/marchproxy/rtmp.yaml`:

```yaml
host: 0.0.0.0
port: 1935
grpc-port: 50053
log-level: info

encoder: auto
preset: medium

output-dir: /var/lib/marchproxy/streams
enable-hls: true
enable-dash: true
segment-duration: 6

max-bitrate: 10
max-streams: 100
max-resolution: 1080

encoder-params:
  tune: zerolatency
  profile: high
```

## Encoder Selection

The container automatically detects available GPUs and selects the best encoder:

1. **NVIDIA GPU detected** → `nvenc_h264`
2. **AMD GPU detected** → `amf_h264`
3. **No GPU** → `x264` (CPU)

Override with `RTMP_ENCODER` environment variable.

## Streaming

### OBS Studio Configuration

1. **Settings → Stream**
   - Service: Custom
   - Server: `rtmp://your-server:1935/live`
   - Stream Key: `your_stream_key`

2. **Settings → Output**
   - Output Mode: Advanced
   - Encoder: x264 (will be transcoded by container)
   - Bitrate: 6000 Kbps

### FFmpeg Publishing

```bash
ffmpeg -re -i input.mp4 \
  -c:v libx264 -preset veryfast -b:v 6000k \
  -c:a aac -b:a 192k \
  -f flv rtmp://your-server:1935/live/your_stream_key
```

## Playback

### HLS
```
http://your-server/streams/your_stream_key/master.m3u8
```

### DASH
```
http://your-server/streams/your_stream_key/dash/manifest.mpd
```

## Performance

### CPU (x264 medium preset)
- Single stream: ~2-3 CPU cores
- 1080p30: ~50-70% CPU on modern CPU
- Latency: 6-12 seconds

### NVIDIA NVENC
- Single stream: ~5-10% GPU, <5% CPU
- Can handle 10+ concurrent 1080p streams on RTX 3060
- Latency: 6-12 seconds

### AMD AMF
- Single stream: ~5-10% GPU, <5% CPU
- Can handle 8+ concurrent 1080p streams on RX 6600
- Latency: 6-12 seconds

## Adaptive Bitrate Ladder

Default ladder (configurable):

| Profile | Resolution | Bitrate | Audio |
|---------|------------|---------|-------|
| 1080p | 1920x1080 | 5000 Kbps | 192 Kbps |
| 720p | 1280x720 | 3000 Kbps | 128 Kbps |
| 480p | 854x480 | 1500 Kbps | 128 Kbps |
| 360p | 640x360 | 800 Kbps | 96 Kbps |

## gRPC API

Implements ModuleService for NLB integration:

- `GetStatus()` - Module status and stats
- `GetRoutes()` - Active streams
- `GetMetrics()` - Transcoding metrics
- `HealthCheck()` - Health status

## Building from Source

```bash
# CPU version
go build -o rtmp-proxy ./cmd/rtmp

# Run
./rtmp-proxy \
  --host 0.0.0.0 \
  --port 1935 \
  --grpc-port 50053 \
  --encoder auto \
  --preset medium
```

## License

Limited AGPL3 - See LICENSE file

## Support

- Documentation: https://docs.marchproxy.com
- Issues: https://github.com/penguintech/marchproxy/issues
- Website: https://www.penguintech.io

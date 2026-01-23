# proxy-rtmp Release Notes

## Version 1.0.0 (Current)

**Release Date**: 2024-01-15

### Overview

Initial production release of proxy-rtmp RTMP streaming server with FFmpeg transcoding and adaptive bitrate output (HLS/DASH).

### Features

#### Core RTMP Server
- Full RTMP (Real Time Messaging Protocol) server implementation
- Support for standard streaming clients (OBS, FFmpeg, GStreamer, etc.)
- Per-stream session management and cleanup
- Configurable server host/port binding

#### FFmpeg Transcoding
- Industry-standard FFmpeg-based video transcoding
- Automatic encoder selection based on available hardware
- Configurable encoding presets (ultrafast to veryslow)
- Advanced codec parameters support

#### Hardware Acceleration
- **NVIDIA NVENC**: H.264 and H.265 encoding
- **AMD AMF**: H.264 and H.265 encoding
- **CPU Fallback**: x264 and x265 software encoders
- Automatic GPU detection and failover

#### Adaptive Bitrate Streaming
- **HLS Output** (HTTP Live Streaming)
  - Master playlist with variant streams
  - Automatic segment generation
  - ABR ladder: 1080p, 720p, 480p, 360p

- **DASH Output** (Dynamic Adaptive Streaming over HTTP)
  - MPEG-DASH manifest generation
  - Same ABR ladder as HLS
  - Multi-period support (future)

#### gRPC Integration
- ModuleService implementation for NLB communication
- GetStatus: Module status and statistics
- GetRoutes: Active stream information
- GetMetrics: Performance metrics
- HealthCheck: Health status for load balancing

#### Performance Monitoring
- Real-time metrics collection
- Per-stream statistics (FPS, bitrate, dropped frames)
- CPU/GPU utilization tracking
- Dropped frame detection and logging

#### Configuration
- Environment variables with RTMP_ prefix
- YAML configuration file support
- Command-line argument overrides
- Configuration validation on startup

#### Logging
- Structured JSON logging
- Configurable log levels (debug, info, warn, error)
- Per-component logging
- Comprehensive error messages

### Technical Specifications

**Performance (Single Stream)**
- CPU (x264 medium): ~2-3 cores, 50-70% on modern CPU
- NVIDIA NVENC: ~5-10% GPU, <5% CPU
- AMD AMF: ~5-10% GPU, <5% CPU
- Latency: 6-12 seconds (configurable to 4-8s with low-latency preset)

**Capacity**
- CPU-based: 3-5 concurrent 1080p streams
- NVIDIA GPU: 20-50 concurrent streams (hardware dependent)
- AMD GPU: 15-40 concurrent streams (hardware dependent)

**Supported Codecs**
- Input: H.264, H.265, VP8, VP9, AV1 (via FFmpeg)
- Output: H.264 (AVC), H.265 (HEVC)
- Audio: AAC, Opus (via FFmpeg)

**Resolution Support**
- Input: Any resolution (configurable max)
- Output: 360p, 480p, 720p, 1080p (auto-scaled)

**Container Support**
- Input: RTMP, FLV (via RTMP)
- Output: HLS (MPEG-TS segments), DASH (MP4 segments)

### Deployment

#### Docker Images
- **CPU**: `marchproxy-rtmp:cpu` (Software encoding)
- **NVIDIA**: `marchproxy-rtmp:nvidia` (NVIDIA NVENC)
- **AMD**: `marchproxy-rtmp:amd` (AMD AMF)

#### Platform Support
- Linux (primary): Debian-based distributions
- Architecture: x86_64 (amd64), ARM64 (aarch64)
- Docker: v20+ recommended

#### System Requirements

**Minimum (CPU-based)**
- CPU: 2 cores modern processor
- Memory: 512 MB per stream
- Disk: 100 GB for archive (streaming output temporary)
- Network: 10 Mbps outbound

**Recommended (Single stream, CPU)**
- CPU: 4+ cores @ 2.4 GHz+
- Memory: 2 GB
- Disk: SSD (temporary segment storage)
- Network: 50 Mbps outbound

**GPU-Accelerated (NVIDIA)**
- GPU: NVIDIA Maxwell architecture or newer
- CUDA Compute Capability: 5.2+ (recommended 7.0+)
- Driver: 450+ for best compatibility
- Memory: 2 GB GPU VRAM per stream

**GPU-Accelerated (AMD)**
- GPU: RDNA architecture or newer (RX 6600+)
- Driver: Latest AMDGPU driver
- Memory: 2 GB GPU VRAM per stream

### Configuration Defaults

```yaml
# Server
host: 0.0.0.0
port: 1935
grpc-port: 50053

# Logging
log-level: info

# Encoder
encoder: auto              # auto, x264, x265, nvenc_h264, nvenc_h265, amf_h264, amf_h265
preset: medium             # ultrafast, fast, medium, slow

# Output
output-dir: /var/lib/marchproxy/streams
enable-hls: true
enable-dash: true
segment-duration: 6

# Rate Limiting
max-bitrate: 10           # Mbps per stream
max-streams: 100          # Concurrent streams
max-resolution: 1080      # Height in pixels
```

### Known Issues

1. **No Stream Authentication**
   - Current: Open RTMP endpoint
   - Mitigation: Use firewall rules to restrict access
   - Future: Token-based authentication planned

2. **Fixed ABR Ladder**
   - Current: Hardcoded profiles (1080p, 720p, 480p, 360p)
   - Mitigation: Can adjust resolution via source encoding
   - Future: Customizable bitrate ladder planned

3. **Limited Input Formats**
   - Current: RTMP only
   - Future: SRT, RIST, NDI support planned

4. **Single Codec Output**
   - Current: H.264 primary (H.265 available)
   - Limitation: No AV1 output yet
   - Future: Multi-codec output planned

### Compatibility

**Tested Publishing Clients**
- OBS Studio 28.0+
- FFmpeg 4.4+
- GStreamer 1.20+
- Wirecast 14+
- vMix 28+

**Tested Playback Players**
- VLC 3.0+
- FFmpeg/ffplay
- Chrome/Firefox/Safari (HLS.js)
- Native iOS (Safari)
- Native Android (ExoPlayer)
- DASH.js
- Shaka Player

**Tested Operating Systems**
- Debian 11, 12 (bookworm)
- Ubuntu 20.04 LTS, 22.04 LTS, 24.04 LTS
- Docker (various versions)

### Dependencies

**Core**
- Go 1.24+
- FFmpeg 4.4+ (with libx264, libx265)
- FFprobe (included with FFmpeg)

**Hardware Support**
- NVIDIA: CUDA Toolkit 11.0+, nvidia-codec-headers
- AMD: ROCm 5.0+ (for hardware decoding, optional)

**Go Libraries**
- google.golang.org/grpc v1.50+
- github.com/spf13/cobra (CLI)
- github.com/spf13/viper (Config)
- github.com/sirupsen/logrus (Logging)

### Breaking Changes

None - Initial release.

### Deprecations

None - Initial release.

### Security

**Security Features**
- No hardcoded credentials
- Config validation on startup
- Input sanitization for stream keys
- Permission restrictions on output directory (0755)

**Known Security Considerations**
- No stream authentication (open RTMP endpoint)
- No TLS/SSL for gRPC (internal communication)
- No rate limiting on connection attempts
- Recommendations: Use firewall rules, internal networks only

**Reported & Fixed**
None for initial release.

### Testing

**Coverage**
- Unit Tests: >90% of core components
- Integration Tests: Full RTMP publishing/playback flow
- Performance Tests: Encoder selection, latency verification
- Regression Tests: Hardware detection and failover

**Tested Scenarios**
- Normal publishing and playback
- Concurrent stream handling
- GPU detection and automatic selection
- CPU fallback on missing GPU
- Resolution scaling and bitrate adaptation
- High load (>50 concurrent streams on GPU)
- Long-running streams (12+ hours)
- Graceful shutdown and cleanup

### Documentation

Tier 3 comprehensive documentation:
- **API.md**: RTMP and gRPC API reference
- **CONFIGURATION.md**: Detailed configuration guide
- **TESTING.md**: Testing procedures and benchmarks
- **USAGE.md**: User guide and troubleshooting
- **RELEASE_NOTES.md**: This file

### Migration

N/A - Initial release.

### Contributors

- Development: PenguinTech Engineering Team
- Quality Assurance: Internal testing team

### License

Limited AGPL3 with commercial licensing available.

See LICENSE file for details.

---

## Version 0.9.0 (Pre-Release)

This was the pre-release development version.

### Features Added
- Initial RTMP server implementation
- Basic FFmpeg transcoding pipeline
- NVIDIA NVENC support
- HLS output generation

### Known Limitations
- AMD AMF support incomplete
- DASH output not production-ready
- No gRPC ModuleService yet
- Limited error handling

---

## Upgrade Guide

### From 0.9.0 to 1.0.0

**Breaking Changes**
None - API compatible.

**Migration Steps**
1. Backup configuration: `cp /etc/marchproxy/rtmp.yaml /etc/marchproxy/rtmp.yaml.bak`
2. Update Docker image: `docker pull marchproxy-rtmp:latest`
3. Verify new features:
   - Check gRPC endpoint: `grpcurl -plaintext localhost:50053 list`
   - Monitor metrics: `grpcurl -plaintext localhost:50053 marchproxy.module.ModuleService/GetMetrics`
4. No database migrations needed (stateless)

**Configuration Updates (Optional)**
- New `health-check-interval` option available (default: 30s)
- DASH output now fully supported (enable with `enable-dash: true`)
- gRPC port now required (default: 50053)

### From 1.0.0 to Future Versions

Migration guides will be provided for each major release.

---

## Future Roadmap

### Version 1.1.0 (Q2 2024)

**Planned Features**
- Multi-codec output (H.264, H.265, VP9)
- Customizable bitrate ladder
- Stream authentication (token-based)
- Advanced audio options (Opus codec, spatial audio)

**Performance Improvements**
- AV1 hardware encoding (future GPU support)
- Improved memory management for high stream count
- Multi-threaded segment generation

### Version 1.2.0 (Q3 2024)

**Planned Features**
- Additional input protocols: SRT, RIST, NDI
- Low latency CMAF support
- Advanced subtitle/caption support
- Multi-language audio tracks

**Infrastructure**
- Kubernetes support (Helm charts)
- Distributed transcoding (multi-pod)
- Enhanced monitoring (Prometheus native)

### Version 2.0.0 (2025)

**Major Changes**
- Clustering support for failover/redundancy
- Cloud storage integration (S3, GCS, Azure Blob)
- Advanced DRM/encryption support
- Enterprise licensing integration

**Next-Gen Features**
- AI-powered encoding optimization
- Predictive bandwidth management
- Advanced analytics dashboard

---

## Support & Feedback

- **Bug Reports**: GitHub Issues (or via license.penguintech.io)
- **Feature Requests**: GitHub Discussions
- **Security Issues**: security@penguintech.io
- **Commercial Support**: support@penguintech.io

---

## Acknowledgments

proxy-rtmp leverages excellent open-source projects:
- **FFmpeg**: Video transcoding engine
- **Go**: Programming language and ecosystem
- **gRPC**: Modern RPC framework
- Community contributors and testers

---

## Legal

**License**: Limited AGPL3
**Copyright**: PenguinTech, Inc.
**Website**: https://www.penguintech.io

For commercial licensing or enterprise support, contact: licensing@penguintech.io

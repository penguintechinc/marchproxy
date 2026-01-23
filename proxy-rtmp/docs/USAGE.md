# proxy-rtmp Usage Guide

## Quick Start

### 1. Start the RTMP Server

Using Docker (recommended):

```bash
docker run -d \
  --name marchproxy-rtmp \
  -p 1935:1935 \
  -p 50053:50053 \
  -p 80:80 \
  -v /var/lib/marchproxy/streams:/var/lib/marchproxy/streams \
  -e RTMP_ENCODER=auto \
  -e RTMP_PRESET=medium \
  marchproxy-rtmp:latest
```

Or from source:

```bash
go run ./cmd/rtmp/main.go \
  --host 0.0.0.0 \
  --port 1935 \
  --encoder auto
```

### 2. Publish a Stream

Using FFmpeg:

```bash
ffmpeg -re -i input.mp4 \
  -c:v libx264 -preset veryfast -b:v 6000k \
  -c:a aac -b:a 192k \
  -f flv rtmp://localhost:1935/live/my_stream
```

Using OBS Studio:
1. Settings → Stream
2. Service: Custom
3. Server: `rtmp://localhost:1935/live`
4. Stream Key: `my_stream`
5. Click "Start Streaming"

### 3. Watch the Stream

HLS (Recommended for web):
```
http://localhost/streams/my_stream/master.m3u8
```

DASH (Alternative):
```
http://localhost/streams/my_stream/dash/manifest.mpd
```

## Publishing Methods

### FFmpeg Publishing

Basic streaming:

```bash
ffmpeg -i input.mp4 \
  -c:v libx264 -b:v 6000k \
  -c:a aac -b:a 192k \
  -f flv rtmp://server:1935/live/stream_key
```

Real-time streaming (with `-re` flag):

```bash
ffmpeg -re -i input.mp4 \
  -c:v libx264 -preset veryfast -b:v 6000k \
  -c:a aac -b:a 192k \
  -f flv rtmp://server:1935/live/stream_key
```

From webcam (Linux):

```bash
ffmpeg -f v4l2 -i /dev/video0 \
  -f pulse -i default \
  -c:v libx264 -preset veryfast -b:v 2000k \
  -c:a aac -b:a 128k \
  -f flv rtmp://server:1935/live/webcam_stream
```

From screen capture (Linux):

```bash
ffmpeg -f x11grab -i :0.0+0,0 \
  -f pulse -i default \
  -c:v libx264 -preset veryfast -b:v 4000k \
  -c:a aac -b:a 128k \
  -f flv rtmp://server:1935/live/screen_stream
```

### OBS Studio

1. **Create Scene** with sources (camera, display, media, etc.)

2. **Configure Stream Settings**:
   - Settings → Stream
   - Service: Custom
   - Server: `rtmp://your-server:1935/live`
   - Stream Key: `unique_stream_name`

3. **Output Settings** (optional):
   - Output Mode: Advanced
   - Encoder: x264 (or hardware encoder)
   - Rate Control: CBR
   - Bitrate: 6000 Kbps (adjust based on quality/bandwidth)

4. **Start Streaming**: Click "Start Streaming" button

5. **Monitor**: View stream stats in OBS status bar

### GStreamer

Live camera stream:

```bash
gst-launch-1.0 -v \
  v4l2src device=/dev/video0 ! \
  video/x-raw,width=1280,height=720,framerate=30/1 ! \
  x264enc speed-preset=3 bitrate=2000 ! \
  queue ! \
  flvmux name=mux ! \
  rtmpsink location="rtmp://server:1935/live/gstream_cam"
```

File streaming:

```bash
gst-launch-1.0 -v \
  filesrc location=input.mp4 ! \
  decodebin ! \
  videoscale ! video/x-raw,width=1920,height=1080 ! \
  x264enc ! queue ! \
  flvmux ! \
  rtmpsink location="rtmp://server:1935/live/gstream_file"
```

## Playback Methods

### HLS Playback

**VLC Player**:
1. File → Open Network Stream
2. Enter: `http://server/streams/stream_key/master.m3u8`
3. Click "Play"

**FFmpeg**:
```bash
ffplay http://server/streams/stream_key/master.m3u8
```

**Web Browser** (with HLS.js or Video.js):
```html
<!DOCTYPE html>
<html>
<head>
    <script src="https://cdn.jsdelivr.net/npm/hls.js@latest"></script>
</head>
<body>
    <video id="video" width="800" height="600" controls></video>
    <script>
        var video = document.getElementById('video');
        var hls = new Hls();
        hls.loadSource('http://server/streams/stream_key/master.m3u8');
        hls.attachMedia(video);
    </script>
</body>
</html>
```

### DASH Playback

**VLC Player**:
1. File → Open Network Stream
2. Enter: `http://server/streams/stream_key/dash/manifest.mpd`
3. Click "Play"

**Web Browser** (with DASH.js):
```html
<!DOCTYPE html>
<html>
<head>
    <script src="https://cdn.dashjs.org/latest/dash.all.min.js"></script>
</head>
<body>
    <video id="videoPlayer" width="800" height="600" controls></video>
    <script>
        var url = "http://server/streams/stream_key/dash/manifest.mpd";
        var player = dashjs.MediaPlayer().create();
        player.initialize(document.querySelector("#videoPlayer"), url, true);
    </script>
</body>
</html>
```

### RTMP Playback (if enabled)

```bash
ffplay rtmp://server:1935/live/stream_key
```

**Note**: RTMP playback requires RTMP-compatible player. HLS/DASH recommended for web.

## Stream Management

### Listing Active Streams

Using gRPC API:

```bash
grpcurl -plaintext localhost:50053 \
  marchproxy.module.ModuleService/GetRoutes
```

Using Go client:

```go
client := pb.NewModuleServiceClient(conn)
routes, _ := client.GetRoutes(context.Background(), &pb.Empty{})
for _, route := range routes.Routes {
    fmt.Printf("Stream: %s\n", route.StreamKey)
}
```

### Stream Statistics

Get real-time metrics:

```bash
grpcurl -plaintext localhost:50053 \
  marchproxy.module.ModuleService/GetMetrics
```

Includes:
- CPU/GPU utilization
- Per-stream metrics (FPS, bitrate, dropped frames)
- Memory usage

### Stopping a Stream

Stop publishing from source:
1. Stop FFmpeg/OBS client
2. Server automatically cleans up stream
3. HLS/DASH outputs remain available for playback

Remove stored stream:

```bash
rm -rf /var/lib/marchproxy/streams/stream_key
```

## Performance Optimization

### Low Latency Streaming

For applications requiring minimal delay (e.g., live events):

**Configuration**:
```yaml
preset: superfast
segment-duration: 2
encoder-params:
  tune: zerolatency
```

**Client settings** (OBS):
- Bitrate: 4000-6000 Kbps
- Encoder preset: veryfast

**Expected latency**: 4-8 seconds (end-to-end)

### High Quality Streaming

For on-demand content with flexible latency requirements:

**Configuration**:
```yaml
preset: slow
segment-duration: 10
encoder-params:
  tune: film
```

**Client settings** (FFmpeg):
- Bitrate: 8000+ Kbps
- Preset: medium or slower

**Expected latency**: 10-15 seconds, excellent quality

### Maximum Throughput

For handling many concurrent streams:

**Configuration**:
```yaml
encoder: nvenc_h264
preset: fast
max-streams: 100
max-bitrate: 25
```

**Use GPU acceleration**: NVIDIA NVENC recommended

**Expected capacity**: 50-100 concurrent 1080p streams

### Minimal Resource Usage

For low-power systems:

**Configuration**:
```yaml
encoder: x264
preset: ultrafast
max-streams: 5
max-bitrate: 5
max-resolution: 720
segment-duration: 10
```

**Expected: <20% CPU, <500MB memory for single stream

## Common Use Cases

### Event Broadcasting

1. **Setup**:
   - Configure for low latency
   - Use hardware acceleration if available
   - Enable both HLS and DASH for compatibility

2. **Publishing** (from event location):
   ```bash
   ffmpeg -f dshow -i "video=Logitech Webcam" \
          -f dshow -i "audio=Microphone" \
          -c:v libx264 -preset veryfast -b:v 5000k \
          -c:a aac -b:a 192k \
          -f flv rtmp://broadcast-server:1935/live/event_2024
   ```

3. **Playback** (for viewers):
   - Distribute `http://server/streams/event_2024/master.m3u8`
   - Embed in website with HLS.js or Video.js

### Game Streaming

1. **OBS Configuration**:
   - Scene: Game window capture + camera overlay
   - Bitrate: 6000-10000 Kbps for 1080p60
   - Encoder: NVIDIA NVENC (if available)
   - Server: `rtmp://server:1935/live`
   - Stream key: `game_stream_1`

2. **Viewers**:
   - Stream to twitch/YouTube while running local instance
   - Or embed in personal website

### Archive & VOD

1. **Publish Live Stream** (as above)

2. **Convert to MP4 for Archive**:
   ```bash
   ffmpeg -i /var/lib/marchproxy/streams/archive/stream_key/master.m3u8 \
          -c copy -bsf:a aac_adtstoasc archive.mp4
   ```

3. **Serve via HTTP** or upload to cloud storage

### Multi-Bitrate Ladder (Custom)

Customize adaptive profiles:

**Configuration** (future enhancement):
```yaml
# Will be customizable in future versions
# Currently fixed to: 1080p, 720p, 480p, 360p
```

## Troubleshooting

### "Connection refused" when publishing

**Symptom**: `rtmp connection refused at rtmp://...`

**Solutions**:
1. Verify server is running: `ps aux | grep rtmp-proxy`
2. Check port is open: `netstat -tlnp | grep 1935`
3. Check firewall: `ufw status` (UFW) or `firewall-cmd` (firewalld)
4. Verify RTMP_PORT env var: `echo $RTMP_PORT`

### "Stream not found" on playback

**Symptom**: HLS/DASH playlists return 404

**Solutions**:
1. Verify stream key matches exactly (case-sensitive)
2. Check stream is actively publishing (look at active logs)
3. Verify output directory has files: `ls /var/lib/marchproxy/streams/stream_key/`
4. Check RTMP_OUTPUT_DIR is correct

### "Encoder not found" error

**Symptom**: Server logs show encoder error, no transcoding

**Solutions**:
1. Verify GPU driver installed (for NVENC/AMF)
2. Check RTMP_ENCODER env var: `echo $RTMP_ENCODER`
3. Force CPU encoder: `RTMP_ENCODER=x264`
4. Verify ffmpeg installed: `which ffmpeg`

### Latency is high (>20 seconds)

**Symptom**: Noticeable delay between publish and playback

**Solutions**:
1. Reduce segment duration: `RTMP_SEGMENT_DURATION=2`
2. Lower encoding preset: `RTMP_PRESET=fast`
3. Check CPU/GPU utilization: `nvidia-smi` or `top`
4. Reduce resolution/bitrate at source

### CPU/GPU usage very high

**Symptom**: Process using >90% CPU or GPU

**Solutions**:
1. Increase preset (slower/higher quality): `RTMP_PRESET=slow`
2. Reduce input resolution at source
3. Reduce concurrent streams: `RTMP_MAX_STREAMS=10`
4. Reduce maximum bitrate: `RTMP_MAX_BITRATE=10`
5. Switch to lower-resolution outputs

### Playback buffering frequently

**Symptom**: Video stops/buffers during playback

**Solutions**:
1. Check network bandwidth: confirm sufficient bandwidth available
2. Reduce publishing bitrate
3. Switch to DASH (better adaptive) or HLS
4. Verify server has sufficient CPU/GPU resources
5. Check disk I/O: `iostat -x 1`

## Advanced Usage

### Custom Bitrate Ladder (Future)

Custom adaptive profiles will be supported in future versions:

```yaml
# Coming soon
bitrate-ladder:
  - name: "4K"
    width: 3840
    height: 2160
    bitrate: 15000
  - name: "1080p"
    width: 1920
    height: 1080
    bitrate: 5000
```

### Stream Authentication (Future)

Token-based authentication planned:

```yaml
# Coming soon
authentication:
  enabled: true
  token-validation: "http://auth-service/validate"
```

### Custom Output Formats (Future)

Additional output formats planned:
- CMAF (Common Media Application Format)
- Progressive MP4 (download)
- Subtitles/TTML support

### Multi-Protocol Support (Future)

Additional input protocols planned:
- SRT (Secure Reliable Transport)
- RIST (Reliable Internet Stream Transport)
- NDI (Network Device Interface)

## Support & Resources

- **Documentation**: See other docs/ files for detailed reference
- **Issues**: Report bugs at GitHub issue tracker
- **Community**: Join project discussions
- **Website**: https://www.penguintech.io

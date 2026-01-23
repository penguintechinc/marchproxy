# MarchProxy OBS Studio Plugin

A Lua script for OBS Studio that simplifies configuration for streaming to MarchProxy with support for RTMP, SRT, and WebRTC/WHIP protocols.

## Features

- **RTMP Support**: Standard RTMP streaming with customizable ports
- **SRT Support**: Low-latency SRT streaming with configurable latency and encryption
- **WebRTC/WHIP Support**: Ultra-low latency WebRTC streaming (experimental)
- **Resolution Presets**: Quick selection of common resolutions (720p to 8K)
- **Bitrate Management**: Automatic bitrate recommendations based on resolution

## Requirements

- OBS Studio 28.0 or later
- For WebRTC/WHIP: OBS 30.0+ with WHIP output support

## Installation

### Linux

```bash
./install-linux.sh
# Or manually:
cp marchproxy-stream.lua ~/.config/obs-studio/scripts/
```

### macOS

```bash
./install-macos.sh
# Or manually:
cp marchproxy-stream.lua ~/Library/Application\ Support/obs-studio/scripts/
```

### Windows

```powershell
.\install-windows.ps1
# Or manually:
# Copy marchproxy-stream.lua to %APPDATA%\obs-studio\scripts\
```

## Usage

1. Open OBS Studio
2. Go to **Tools** -> **Scripts**
3. Click the **+** button and select `marchproxy-stream.lua`
4. Configure your MarchProxy settings:
   - **Server Host**: Your MarchProxy server hostname or IP
   - **Stream Key**: Your stream key
   - **Protocol**: RTMP, SRT, or WebRTC/WHIP
5. Configure protocol-specific settings if needed
6. Select your resolution preset
7. Click **Apply to Stream Settings**
8. Start streaming!

## Configuration Options

### Connection Settings

| Setting | Description |
|---------|-------------|
| Server Host | MarchProxy server hostname or IP address |
| Stream Key | Your unique stream key |
| Protocol | Streaming protocol (RTMP, SRT, or WHIP) |

### Protocol Settings

#### RTMP
| Setting | Default | Description |
|---------|---------|-------------|
| RTMP Port | 1935 | RTMP server port |

#### SRT
| Setting | Default | Description |
|---------|---------|-------------|
| SRT Port | 8890 | SRT server port |
| SRT Latency | 120 | Latency in milliseconds (20-8000) |
| SRT Passphrase | (empty) | Optional encryption passphrase |

#### WebRTC/WHIP
| Setting | Default | Description |
|---------|---------|-------------|
| WHIP Port | 8080 | WHIP HTTP server port |
| Use HTTPS | false | Enable TLS for WHIP endpoint |

### Video Settings

| Preset | Resolution | Recommended Bitrate |
|--------|------------|---------------------|
| 720p (HD) | 1280x720 | 3,000 kbps |
| 1080p (Full HD) | 1920x1080 | 5,000 kbps |
| 1440p (2K) | 2560x1440 | 8,000 kbps |
| 2160p (4K) | 3840x2160 | 20,000 kbps |
| 4320p (8K) | 7680x4320 | 60,000 kbps |

## Troubleshooting

### Common Issues

**Stream won't connect**
- Verify the server hostname and port are correct
- Check that your stream key is valid
- Ensure MarchProxy is running and accepting connections
- Check firewall settings

**SRT connection fails**
- Ensure SRT is enabled on your MarchProxy server
- Verify the latency setting is appropriate for your network
- Check if passphrase is required

**WebRTC/WHIP not available**
- WebRTC requires OBS 30.0 or later
- Ensure WHIP output plugin is installed
- Check if HTTPS is required by your server

### Getting Help

- MarchProxy Documentation: https://marchproxy.penguintech.io/docs
- Support Email: support@penguintech.io
- GitHub Issues: https://github.com/penguintech/marchproxy/issues

## License

This plugin is part of MarchProxy and is licensed under the Limited AGPL-3.0 license.
See the main MarchProxy repository for license details.

---

**MarchProxy** by Penguin Tech Inc - https://penguintech.io

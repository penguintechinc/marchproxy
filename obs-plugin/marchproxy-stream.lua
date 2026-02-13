--[[
    MarchProxy Streaming Configuration for OBS Studio

    This Lua script provides easy configuration for streaming to MarchProxy
    with support for RTMP, SRT, and WebRTC protocols.

    Installation:
    1. Copy this file to your OBS scripts folder:
       - Windows: %APPDATA%\obs-studio\scripts
       - macOS: ~/Library/Application Support/obs-studio/scripts
       - Linux: ~/.config/obs-studio/scripts
    2. In OBS, go to Tools -> Scripts
    3. Click the + button and select marchproxy-stream.lua

    Usage:
    1. Enter your MarchProxy server URL
    2. Enter your stream key
    3. Select protocol (RTMP, SRT, or WebRTC/WHIP)
    4. Configure optional settings
    5. Click "Apply to Stream Settings"
--]]

obs = obslua
settings = nil
applied = false

-- Script information
function script_description()
    return [[<center><h2>MarchProxy Streaming Configuration</h2></center>
<p>Configure OBS to stream to MarchProxy with support for:</p>
<ul>
<li><b>RTMP</b> - Standard protocol, widest compatibility</li>
<li><b>SRT</b> - Low latency, improved reliability</li>
<li><b>WebRTC/WHIP</b> - Ultra-low latency (experimental)</li>
</ul>
<p>Enter your MarchProxy server details below and click "Apply" to update OBS streaming settings.</p>
<hr>
<p><small>MarchProxy by Penguin Tech Inc - https://penguintech.io</small></p>]]
end

-- Default values
local DEFAULT_SERVER = "rtmp://localhost:1935/live"
local DEFAULT_SRT_PORT = 8890
local DEFAULT_WHIP_PORT = 8080
local DEFAULT_LATENCY = 120

-- Protocol definitions
local PROTOCOLS = {
    { name = "RTMP (Standard)", value = "rtmp" },
    { name = "SRT (Low Latency)", value = "srt" },
    { name = "WebRTC/WHIP (Ultra-Low Latency)", value = "whip" }
}

-- Resolution presets
local RESOLUTIONS = {
    { name = "1080p (Full HD)", width = 1920, height = 1080, bitrate = 5000 },
    { name = "1440p (2K)", width = 2560, height = 1440, bitrate = 8000 },
    { name = "2160p (4K)", width = 3840, height = 2160, bitrate = 20000 },
    { name = "4320p (8K)", width = 7680, height = 4320, bitrate = 60000 },
    { name = "720p (HD)", width = 1280, height = 720, bitrate = 3000 },
    { name = "480p (SD)", width = 854, height = 480, bitrate = 1500 }
}

-- Define script properties (UI)
function script_properties()
    local props = obs.obs_properties_create()

    -- Connection Settings Group
    local conn_group = obs.obs_properties_create()

    obs.obs_properties_add_text(conn_group, "server_host", "Server Host", obs.OBS_TEXT_DEFAULT)
    obs.obs_properties_add_text(conn_group, "stream_key", "Stream Key", obs.OBS_TEXT_PASSWORD)

    -- Protocol dropdown
    local proto_list = obs.obs_properties_add_list(conn_group, "protocol", "Protocol",
        obs.OBS_COMBO_TYPE_LIST, obs.OBS_COMBO_FORMAT_STRING)
    for _, proto in ipairs(PROTOCOLS) do
        obs.obs_property_list_add_string(proto_list, proto.name, proto.value)
    end
    obs.obs_property_set_modified_callback(proto_list, on_protocol_changed)

    obs.obs_properties_add_group(props, "connection_group", "Connection Settings",
        obs.OBS_GROUP_NORMAL, conn_group)

    -- Protocol-Specific Settings Group
    local proto_group = obs.obs_properties_create()

    -- RTMP port (usually not changed)
    local rtmp_port = obs.obs_properties_add_int(proto_group, "rtmp_port", "RTMP Port", 1, 65535, 1)

    -- SRT settings
    local srt_port = obs.obs_properties_add_int(proto_group, "srt_port", "SRT Port", 1, 65535, 1)
    local srt_latency = obs.obs_properties_add_int(proto_group, "srt_latency", "SRT Latency (ms)", 20, 8000, 10)
    local srt_passphrase = obs.obs_properties_add_text(proto_group, "srt_passphrase", "SRT Passphrase (optional)", obs.OBS_TEXT_PASSWORD)

    -- WebRTC settings
    local whip_port = obs.obs_properties_add_int(proto_group, "whip_port", "WHIP Port", 1, 65535, 1)
    local whip_https = obs.obs_properties_add_bool(proto_group, "whip_https", "Use HTTPS")

    obs.obs_properties_add_group(props, "protocol_group", "Protocol Settings",
        obs.OBS_GROUP_NORMAL, proto_group)

    -- Video Settings Group
    local video_group = obs.obs_properties_create()

    -- Resolution preset
    local res_list = obs.obs_properties_add_list(video_group, "resolution_preset", "Resolution Preset",
        obs.OBS_COMBO_TYPE_LIST, obs.OBS_COMBO_FORMAT_INT)
    for i, res in ipairs(RESOLUTIONS) do
        obs.obs_property_list_add_int(res_list, res.name, i)
    end
    obs.obs_property_set_modified_callback(res_list, on_resolution_changed)

    -- Custom bitrate override
    obs.obs_properties_add_int(video_group, "custom_bitrate", "Video Bitrate (kbps)", 500, 100000, 100)

    obs.obs_properties_add_group(props, "video_group", "Video Settings",
        obs.OBS_GROUP_NORMAL, video_group)

    -- Action buttons
    obs.obs_properties_add_button(props, "apply_button", "Apply to Stream Settings", apply_settings)
    obs.obs_properties_add_button(props, "test_button", "Test Connection", test_connection)

    -- Status display
    obs.obs_properties_add_text(props, "status_text", "Status", obs.OBS_TEXT_INFO)

    return props
end

-- Set default values
function script_defaults(settings)
    obs.obs_data_set_default_string(settings, "server_host", "localhost")
    obs.obs_data_set_default_string(settings, "stream_key", "")
    obs.obs_data_set_default_string(settings, "protocol", "rtmp")
    obs.obs_data_set_default_int(settings, "rtmp_port", 1935)
    obs.obs_data_set_default_int(settings, "srt_port", DEFAULT_SRT_PORT)
    obs.obs_data_set_default_int(settings, "srt_latency", DEFAULT_LATENCY)
    obs.obs_data_set_default_string(settings, "srt_passphrase", "")
    obs.obs_data_set_default_int(settings, "whip_port", DEFAULT_WHIP_PORT)
    obs.obs_data_set_default_bool(settings, "whip_https", false)
    obs.obs_data_set_default_int(settings, "resolution_preset", 1) -- 1080p
    obs.obs_data_set_default_int(settings, "custom_bitrate", 5000)
    obs.obs_data_set_default_string(settings, "status_text", "Ready")
end

-- Update script settings
function script_update(s)
    settings = s
end

-- Protocol changed callback
function on_protocol_changed(props, property, settings)
    local protocol = obs.obs_data_get_string(settings, "protocol")

    -- Show/hide relevant settings based on protocol
    -- Note: OBS Lua doesn't support dynamic visibility well,
    -- so all fields remain visible but some won't be used

    return true
end

-- Resolution changed callback
function on_resolution_changed(props, property, settings)
    local preset_idx = obs.obs_data_get_int(settings, "resolution_preset")
    if preset_idx > 0 and preset_idx <= #RESOLUTIONS then
        local res = RESOLUTIONS[preset_idx]
        obs.obs_data_set_int(settings, "custom_bitrate", res.bitrate)
    end
    return true
end

-- Build streaming URL based on protocol
function build_stream_url(settings)
    local host = obs.obs_data_get_string(settings, "server_host")
    local stream_key = obs.obs_data_get_string(settings, "stream_key")
    local protocol = obs.obs_data_get_string(settings, "protocol")

    if protocol == "rtmp" then
        local port = obs.obs_data_get_int(settings, "rtmp_port")
        return string.format("rtmp://%s:%d/live", host, port), stream_key

    elseif protocol == "srt" then
        local port = obs.obs_data_get_int(settings, "srt_port")
        local latency = obs.obs_data_get_int(settings, "srt_latency")
        local passphrase = obs.obs_data_get_string(settings, "srt_passphrase")

        local url = string.format("srt://%s:%d?streamid=%s&latency=%d",
            host, port, stream_key, latency * 1000) -- Convert ms to microseconds

        if passphrase and passphrase ~= "" then
            url = url .. "&passphrase=" .. passphrase
        end

        return url, nil -- SRT includes key in URL

    elseif protocol == "whip" then
        local port = obs.obs_data_get_int(settings, "whip_port")
        local https = obs.obs_data_get_bool(settings, "whip_https")
        local scheme = https and "https" or "http"

        return string.format("%s://%s:%d/whip/%s", scheme, host, port, stream_key), nil
    end

    return nil, nil
end

-- Apply settings to OBS stream configuration
function apply_settings(props, p)
    if settings == nil then
        obs.script_log(obs.LOG_WARNING, "Settings not loaded")
        return false
    end

    local protocol = obs.obs_data_get_string(settings, "protocol")
    local url, key = build_stream_url(settings)

    if url == nil then
        obs.obs_data_set_string(settings, "status_text", "Error: Invalid configuration")
        return false
    end

    -- Get the current streaming service
    local service = obs.obs_frontend_get_streaming_service()
    local service_settings = obs.obs_service_get_settings(service)

    if protocol == "rtmp" then
        -- Configure RTMP custom server
        obs.obs_data_set_string(service_settings, "service", "custom")
        obs.obs_data_set_string(service_settings, "server", url)
        obs.obs_data_set_string(service_settings, "key", key)

    elseif protocol == "srt" then
        -- Configure SRT output
        -- Note: OBS native SRT support varies by version
        obs.obs_data_set_string(service_settings, "service", "custom")
        obs.obs_data_set_string(service_settings, "server", url)
        obs.obs_data_set_string(service_settings, "key", "")

    elseif protocol == "whip" then
        -- Configure WHIP output
        -- Note: WHIP support requires OBS 30+ with WHIP output plugin
        obs.obs_data_set_string(service_settings, "service", "custom")
        obs.obs_data_set_string(service_settings, "server", url)
        obs.obs_data_set_string(service_settings, "key", "")
    end

    -- Apply service settings
    obs.obs_service_update(service, service_settings)
    obs.obs_data_release(service_settings)
    obs.obs_frontend_save_streaming_service()

    -- Update video output settings
    local preset_idx = obs.obs_data_get_int(settings, "resolution_preset")
    local bitrate = obs.obs_data_get_int(settings, "custom_bitrate")

    if preset_idx > 0 and preset_idx <= #RESOLUTIONS then
        local res = RESOLUTIONS[preset_idx]
        update_video_settings(res.width, res.height, bitrate)
    end

    applied = true
    local status_msg = string.format("Applied: %s to %s", protocol:upper(), url)
    obs.obs_data_set_string(settings, "status_text", status_msg)
    obs.script_log(obs.LOG_INFO, status_msg)

    return true
end

-- Update video output settings
function update_video_settings(width, height, bitrate)
    -- Get current output settings
    local output = obs.obs_frontend_get_streaming_output()
    if output == nil then
        return
    end

    -- Note: Resolution changes require going through Video Settings
    -- This updates the encoder bitrate
    local encoder = obs.obs_output_get_video_encoder(output)
    if encoder then
        local enc_settings = obs.obs_encoder_get_settings(encoder)
        obs.obs_data_set_int(enc_settings, "bitrate", bitrate)
        obs.obs_encoder_update(encoder, enc_settings)
        obs.obs_data_release(enc_settings)
    end

    obs.obs_output_release(output)

    obs.script_log(obs.LOG_INFO, string.format("Updated bitrate to %d kbps", bitrate))
end

-- Test connection to server
function test_connection(props, p)
    if settings == nil then
        return false
    end

    local host = obs.obs_data_get_string(settings, "server_host")
    local protocol = obs.obs_data_get_string(settings, "protocol")

    -- Simple connectivity test
    -- Note: Full implementation would use socket or HTTP library
    local status_msg = string.format("Testing connection to %s...", host)
    obs.obs_data_set_string(settings, "status_text", status_msg)
    obs.script_log(obs.LOG_INFO, status_msg)

    -- In a real implementation, we would:
    -- 1. For RTMP: Try TCP connect to port
    -- 2. For SRT: Try UDP handshake
    -- 3. For WHIP: Make HTTP OPTIONS request

    status_msg = "Connection test: Please verify manually (OBS Lua has limited networking)"
    obs.obs_data_set_string(settings, "status_text", status_msg)

    return true
end

-- Script loaded
function script_load(settings)
    obs.script_log(obs.LOG_INFO, "MarchProxy streaming plugin loaded")
end

-- Script unloaded
function script_unload()
    obs.script_log(obs.LOG_INFO, "MarchProxy streaming plugin unloaded")
end

/**
 * Media Module API Client for MarchProxy WebUI
 *
 * Provides API methods for managing media streaming configuration,
 * active streams, hardware capabilities, and admin settings.
 */

import { apiClient } from './api';

// Types

export interface MediaStream {
  id: number;
  stream_key: string;
  protocol: 'rtmp' | 'srt' | 'webrtc';
  codec: 'h264' | 'h265' | 'av1' | null;
  resolution: string | null;
  bitrate_kbps: number | null;
  status: 'active' | 'idle' | 'error';
  client_ip: string;
  started_at: string;
  ended_at?: string;
  bytes_in: number;
  bytes_out: number;
}

export interface MediaSettings {
  admin_max_resolution: number | null;
  admin_max_bitrate_kbps: number | null;
  enforce_codec: string | null;
  transcode_ladder_enabled: boolean;
  transcode_ladder_resolutions: number[];
  updated_at?: string;
}

export interface HardwareCapabilities {
  gpu_type: 'none' | 'nvidia' | 'amd';
  gpu_model: string | null;
  vram_gb: number;
  hardware_max_resolution: number;
  av1_supported: boolean;
  supports_8k: boolean;
  supports_4k: boolean;
}

export interface MediaCapabilities {
  hardware: HardwareCapabilities;
  settings: MediaSettings;
  effective_max_resolution: number;
}

export interface ResolutionInfo {
  height: number;
  label: string;
  supported: boolean;
  requires_gpu: boolean;
  disabled_reason?: string;
}

export interface CodecInfo {
  name: string;
  id: string;
  supported: boolean;
  hardware_accelerated: boolean;
}

export interface MediaStats {
  active_streams: number;
  total_bytes_in: number;
  total_bytes_out: number;
  by_protocol: Record<string, number>;
  timestamp: string;
}

export interface RestreamDestination {
  platform: string;
  rtmp_url: string;
  stream_key: string;
  quality: string;
  enabled: boolean;
}

export interface UpdateMediaSettingsRequest {
  admin_max_resolution?: number | null;
  admin_max_bitrate_kbps?: number | null;
  enforce_codec?: string | null;
  transcode_ladder_enabled?: boolean;
  transcode_ladder_resolutions?: number[];
}

// API Methods

/**
 * Get media module configuration
 */
export const getMediaConfig = async (): Promise<MediaSettings> => {
  const response = await apiClient.get<{ config: MediaSettings; status: string }>(
    '/api/v1/modules/rtmp/config'
  );
  return response.data.config;
};

/**
 * Update media module configuration
 */
export const updateMediaConfig = async (
  config: Partial<MediaSettings>
): Promise<MediaSettings> => {
  const response = await apiClient.put<{ config: MediaSettings; status: string }>(
    '/api/v1/modules/rtmp/config',
    config
  );
  return response.data.config;
};

/**
 * Get list of active media streams
 */
export const getActiveStreams = async (): Promise<MediaStream[]> => {
  const response = await apiClient.get<{ streams: MediaStream[]; count: number }>(
    '/api/v1/modules/rtmp/streams'
  );
  return response.data.streams;
};

/**
 * Get details of a specific stream
 */
export const getStream = async (streamKey: string): Promise<MediaStream> => {
  const response = await apiClient.get<{ stream: MediaStream }>(
    `/api/v1/modules/rtmp/streams/${streamKey}`
  );
  return response.data.stream;
};

/**
 * Stop a specific stream
 */
export const stopStream = async (
  streamKey: string
): Promise<{ status: string; stream_key: string }> => {
  const response = await apiClient.delete<{ status: string; stream_key: string }>(
    `/api/v1/modules/rtmp/streams/${streamKey}`
  );
  return response.data;
};

/**
 * Get hardware capabilities and current limits
 */
export const getCapabilities = async (): Promise<MediaCapabilities> => {
  const response = await apiClient.get<MediaCapabilities>(
    '/api/v1/modules/rtmp/capabilities'
  );
  return response.data;
};

/**
 * Get media module statistics
 */
export const getMediaStats = async (): Promise<MediaStats> => {
  const response = await apiClient.get<{ stats: MediaStats }>(
    '/api/v1/modules/rtmp/stats'
  );
  return response.data.stats;
};

// Admin API Methods (Super Admin Only)

/**
 * Get global admin media settings
 */
export const getAdminMediaSettings = async (): Promise<{
  settings: MediaSettings;
  hardware_capabilities: HardwareCapabilities;
  effective_max_resolution: number;
}> => {
  const response = await apiClient.get<{
    settings: MediaSettings;
    hardware_capabilities: HardwareCapabilities;
    effective_max_resolution: number;
  }>('/api/v1/admin/media/settings');
  return response.data;
};

/**
 * Update global admin media settings
 */
export const updateAdminMediaSettings = async (
  settings: UpdateMediaSettingsRequest
): Promise<{ status: string; settings: MediaSettings }> => {
  const response = await apiClient.put<{ status: string; settings: MediaSettings }>(
    '/api/v1/admin/media/settings',
    settings
  );
  return response.data;
};

/**
 * Reset admin resolution override to hardware default
 */
export const resetAdminOverride = async (): Promise<{
  status: string;
  message: string;
  settings: MediaSettings;
}> => {
  const response = await apiClient.post<{
    status: string;
    message: string;
    settings: MediaSettings;
  }>('/api/v1/admin/media/settings/reset');
  return response.data;
};

/**
 * Get detailed hardware capabilities report
 */
export const getAdminCapabilities = async (): Promise<{
  hardware: HardwareCapabilities;
  resolutions: ResolutionInfo[];
  supported_codecs: CodecInfo[];
}> => {
  const response = await apiClient.get<{
    hardware: HardwareCapabilities;
    resolutions: ResolutionInfo[];
    supported_codecs: CodecInfo[];
  }>('/api/v1/admin/media/capabilities');
  return response.data;
};

// Restreaming API Methods

/**
 * Get restream destinations for a stream
 */
export const getRestreamDestinations = async (
  streamKey: string
): Promise<RestreamDestination[]> => {
  const response = await apiClient.get<{
    stream_key: string;
    destinations: RestreamDestination[];
  }>(`/api/v1/modules/rtmp/streams/${streamKey}/restream`);
  return response.data.destinations;
};

/**
 * Add restream destination
 */
export const addRestreamDestination = async (
  streamKey: string,
  destination: Omit<RestreamDestination, 'enabled'> & { enabled?: boolean }
): Promise<{ status: string; stream_key: string; destination: Partial<RestreamDestination> }> => {
  const response = await apiClient.post<{
    status: string;
    stream_key: string;
    destination: Partial<RestreamDestination>;
  }>(`/api/v1/modules/rtmp/streams/${streamKey}/restream`, destination);
  return response.data;
};

/**
 * Remove restream destination
 */
export const removeRestreamDestination = async (
  streamKey: string
): Promise<{ status: string; stream_key: string }> => {
  const response = await apiClient.delete<{ status: string; stream_key: string }>(
    `/api/v1/modules/rtmp/streams/${streamKey}/restream`
  );
  return response.data;
};

// Utility Functions

/**
 * Get resolution label for display
 */
export const getResolutionLabel = (height: number): string => {
  const labels: Record<number, string> = {
    360: '360p',
    480: '480p (SD)',
    540: '540p',
    720: '720p (HD)',
    1080: '1080p (Full HD)',
    1440: '1440p (2K)',
    2160: '2160p (4K)',
    4320: '4320p (8K)',
  };
  return labels[height] || `${height}p`;
};

/**
 * Get protocol display name
 */
export const getProtocolLabel = (protocol: string): string => {
  const labels: Record<string, string> = {
    rtmp: 'RTMP',
    srt: 'SRT',
    webrtc: 'WebRTC',
    whip: 'WHIP',
    whep: 'WHEP',
  };
  return labels[protocol] || protocol.toUpperCase();
};

/**
 * Get codec display name
 */
export const getCodecLabel = (codec: string): string => {
  const labels: Record<string, string> = {
    h264: 'H.264',
    h265: 'H.265/HEVC',
    av1: 'AV1',
  };
  return labels[codec] || codec.toUpperCase();
};

/**
 * Format bytes to human readable
 */
export const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
};

/**
 * Format bitrate to human readable
 */
export const formatBitrate = (kbps: number): string => {
  if (kbps >= 1000) {
    return `${(kbps / 1000).toFixed(1)} Mbps`;
  }
  return `${kbps} kbps`;
};

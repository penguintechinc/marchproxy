/**
 * Media Components Index
 *
 * Reusable components for the Media Streaming module.
 */

export { default as ResolutionSelector } from './ResolutionSelector';
export { default as ProtocolBadge } from './ProtocolBadge';
export { default as CodecBadge } from './CodecBadge';

// Re-export types from mediaApi for convenience
export type {
  MediaStream,
  MediaSettings,
  HardwareCapabilities,
  MediaCapabilities,
  ResolutionInfo,
  CodecInfo,
} from '@services/mediaApi';

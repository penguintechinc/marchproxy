/**
 * Codec Badge Component
 *
 * Displays video codec as a colored badge.
 */

import React from 'react';
import { Chip, ChipProps } from '@mui/material';
import { getCodecLabel } from '@services/mediaApi';

interface CodecBadgeProps {
  codec: string | null;
  hardwareAccelerated?: boolean;
  size?: 'small' | 'medium';
}

const getCodecColor = (codec: string | null): ChipProps['color'] => {
  switch (codec?.toLowerCase()) {
    case 'h264':
      return 'success';
    case 'h265':
    case 'hevc':
      return 'info';
    case 'av1':
      return 'warning';
    default:
      return 'default';
  }
};

const CodecBadge: React.FC<CodecBadgeProps> = ({
  codec,
  hardwareAccelerated = false,
  size = 'small',
}) => {
  if (!codec) {
    return <Chip label="Unknown" color="default" size={size} variant="outlined" />;
  }

  const label = hardwareAccelerated
    ? `${getCodecLabel(codec)} (HW)`
    : getCodecLabel(codec);

  return <Chip label={label} color={getCodecColor(codec)} size={size} variant="filled" />;
};

export default CodecBadge;

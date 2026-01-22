/**
 * Protocol Badge Component
 *
 * Displays streaming protocol as a colored badge.
 */

import React from 'react';
import { Chip, ChipProps } from '@mui/material';
import { getProtocolLabel } from '@services/mediaApi';

interface ProtocolBadgeProps {
  protocol: string;
  size?: 'small' | 'medium';
}

const getProtocolColor = (protocol: string): ChipProps['color'] => {
  switch (protocol.toLowerCase()) {
    case 'rtmp':
      return 'primary';
    case 'srt':
      return 'secondary';
    case 'webrtc':
    case 'whip':
    case 'whep':
      return 'info';
    default:
      return 'default';
  }
};

const ProtocolBadge: React.FC<ProtocolBadgeProps> = ({ protocol, size = 'small' }) => {
  return (
    <Chip
      label={getProtocolLabel(protocol)}
      color={getProtocolColor(protocol)}
      size={size}
      variant="filled"
    />
  );
};

export default ProtocolBadge;

/**
 * Resolution Selector Component
 *
 * Dropdown selector for video resolutions with disabled options
 * showing tooltips explaining why certain resolutions are unavailable.
 */

import React from 'react';
import {
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Tooltip,
  Box,
  SelectChangeEvent,
} from '@mui/material';
import InfoIcon from '@mui/icons-material/Info';
import { getResolutionLabel } from '@services/mediaApi';

interface ResolutionOption {
  value: number;
  label: string;
  enabled: boolean;
  disabledReason?: string;
}

interface ResolutionSelectorProps {
  value: number;
  onChange: (value: number) => void;
  effectiveMax: number;
  hardwareMax: number;
  adminMax: number | null;
  gpuType: string;
  label?: string;
  fullWidth?: boolean;
  disabled?: boolean;
}

const getDisabledReason = (
  resolution: number,
  effectiveMax: number,
  hardwareMax: number,
  adminMax: number | null,
  gpuType: string
): string | undefined => {
  if (resolution <= effectiveMax) return undefined;

  // Admin limit is the bottleneck
  if (adminMax !== null && resolution > adminMax) {
    return `Administrator limit: ${getResolutionLabel(adminMax)} maximum`;
  }

  // Hardware is the bottleneck
  if (gpuType === 'none') {
    return 'Requires GPU hardware acceleration';
  }
  if (resolution > hardwareMax) {
    return `GPU does not support ${getResolutionLabel(resolution)} (requires more VRAM)`;
  }

  return 'Not available';
};

const ResolutionSelector: React.FC<ResolutionSelectorProps> = ({
  value,
  onChange,
  effectiveMax,
  hardwareMax,
  adminMax,
  gpuType,
  label = 'Maximum Resolution',
  fullWidth = true,
  disabled = false,
}) => {
  const resolutions = [360, 480, 540, 720, 1080, 1440, 2160, 4320];

  const options: ResolutionOption[] = resolutions.map((height) => ({
    value: height,
    label: getResolutionLabel(height),
    enabled: height <= effectiveMax,
    disabledReason: getDisabledReason(height, effectiveMax, hardwareMax, adminMax, gpuType),
  }));

  const handleChange = (event: SelectChangeEvent<number>) => {
    onChange(event.target.value as number);
  };

  return (
    <FormControl fullWidth={fullWidth} disabled={disabled}>
      <InputLabel>{label}</InputLabel>
      <Select value={value} onChange={handleChange} label={label}>
        {options.map((opt) => (
          <Tooltip
            key={opt.value}
            title={opt.disabledReason || ''}
            placement="right"
            disableHoverListener={opt.enabled}
          >
            <span>
              <MenuItem
                value={opt.value}
                disabled={!opt.enabled}
                sx={{
                  opacity: opt.enabled ? 1 : 0.5,
                  '&.Mui-disabled': { pointerEvents: 'auto' },
                }}
              >
                <Box display="flex" alignItems="center" gap={1} width="100%">
                  {opt.label}
                  {!opt.enabled && (
                    <InfoIcon sx={{ ml: 'auto', fontSize: 16, color: 'text.secondary' }} />
                  )}
                </Box>
              </MenuItem>
            </span>
          </Tooltip>
        ))}
      </Select>
    </FormControl>
  );
};

export default ResolutionSelector;

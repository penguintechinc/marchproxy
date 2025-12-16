/**
 * TrafficSlider Component
 *
 * Interactive slider for controlling traffic weights in blue/green deployments.
 * Displays percentage split between blue and green versions with visual feedback.
 */

import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Slider,
  Grid,
  Paper,
  Chip,
  Button,
} from '@mui/material';
import SaveIcon from '@mui/icons-material/Save';

interface TrafficSliderProps {
  blueVersion: string;
  greenVersion: string;
  initialBlueWeight: number;
  initialGreenWeight: number;
  onWeightChange: (blueWeight: number, greenWeight: number) => void;
  onSave?: () => void;
  disabled?: boolean;
  showSaveButton?: boolean;
}

const TrafficSlider: React.FC<TrafficSliderProps> = ({
  blueVersion,
  greenVersion,
  initialBlueWeight,
  initialGreenWeight,
  onWeightChange,
  onSave,
  disabled = false,
  showSaveButton = false,
}) => {
  const [blueWeight, setBlueWeight] = useState(initialBlueWeight);
  const [hasChanges, setHasChanges] = useState(false);

  useEffect(() => {
    setBlueWeight(initialBlueWeight);
    setHasChanges(false);
  }, [initialBlueWeight]);

  const handleSliderChange = (_event: Event, newValue: number | number[]) => {
    const blue = newValue as number;
    const green = 100 - blue;
    setBlueWeight(blue);
    setHasChanges(true);
    onWeightChange(blue, green);
  };

  const handlePresetClick = (blue: number) => {
    const green = 100 - blue;
    setBlueWeight(blue);
    setHasChanges(true);
    onWeightChange(blue, green);
  };

  const greenWeight = 100 - blueWeight;

  const getBlueGradient = () => {
    return `linear-gradient(90deg, #2196F3 0%, #2196F3 ${blueWeight}%, #4CAF50 ${blueWeight}%, #4CAF50 100%)`;
  };

  return (
    <Paper elevation={2} sx={{ p: 3 }}>
      <Typography variant="h6" gutterBottom>
        Traffic Distribution
      </Typography>

      <Grid container spacing={3}>
        {/* Version Info */}
        <Grid item xs={12} md={6}>
          <Box
            sx={{
              p: 2,
              bgcolor: '#E3F2FD',
              borderRadius: 1,
              border: 2,
              borderColor: '#2196F3',
            }}
          >
            <Typography variant="caption" color="text.secondary">
              Blue Version
            </Typography>
            <Typography variant="h6" fontWeight="bold" color="#2196F3">
              {blueVersion}
            </Typography>
            <Typography variant="h4" fontWeight="bold" color="#2196F3" mt={1}>
              {blueWeight.toFixed(0)}%
            </Typography>
          </Box>
        </Grid>

        <Grid item xs={12} md={6}>
          <Box
            sx={{
              p: 2,
              bgcolor: '#E8F5E9',
              borderRadius: 1,
              border: 2,
              borderColor: '#4CAF50',
            }}
          >
            <Typography variant="caption" color="text.secondary">
              Green Version
            </Typography>
            <Typography variant="h6" fontWeight="bold" color="#4CAF50">
              {greenVersion}
            </Typography>
            <Typography variant="h4" fontWeight="bold" color="#4CAF50" mt={1}>
              {greenWeight.toFixed(0)}%
            </Typography>
          </Box>
        </Grid>

        {/* Slider */}
        <Grid item xs={12}>
          <Box sx={{ px: 2, py: 3 }}>
            <Slider
              value={blueWeight}
              onChange={handleSliderChange}
              disabled={disabled}
              min={0}
              max={100}
              step={5}
              marks={[
                { value: 0, label: '0%' },
                { value: 25, label: '25%' },
                { value: 50, label: '50%' },
                { value: 75, label: '75%' },
                { value: 100, label: '100%' },
              ]}
              sx={{
                '& .MuiSlider-track': {
                  background: getBlueGradient(),
                  border: 'none',
                },
                '& .MuiSlider-rail': {
                  background: '#4CAF50',
                },
                '& .MuiSlider-thumb': {
                  bgcolor: '#fff',
                  border: 3,
                  borderColor: blueWeight > 50 ? '#2196F3' : '#4CAF50',
                  width: 24,
                  height: 24,
                  '&:hover': {
                    boxShadow: '0 0 0 8px rgba(33, 150, 243, 0.16)',
                  },
                },
              }}
            />
          </Box>
        </Grid>

        {/* Quick Presets */}
        <Grid item xs={12}>
          <Typography variant="caption" color="text.secondary" gutterBottom display="block">
            Quick Presets
          </Typography>
          <Box display="flex" gap={1} flexWrap="wrap">
            <Chip
              label="All Blue"
              onClick={() => handlePresetClick(100)}
              color="primary"
              variant={blueWeight === 100 ? 'filled' : 'outlined'}
              disabled={disabled}
            />
            <Chip
              label="75/25 Blue"
              onClick={() => handlePresetClick(75)}
              color="primary"
              variant={blueWeight === 75 ? 'filled' : 'outlined'}
              disabled={disabled}
            />
            <Chip
              label="50/50 Split"
              onClick={() => handlePresetClick(50)}
              variant={blueWeight === 50 ? 'filled' : 'outlined'}
              disabled={disabled}
            />
            <Chip
              label="25/75 Green"
              onClick={() => handlePresetClick(25)}
              color="success"
              variant={blueWeight === 25 ? 'filled' : 'outlined'}
              disabled={disabled}
            />
            <Chip
              label="All Green"
              onClick={() => handlePresetClick(0)}
              color="success"
              variant={blueWeight === 0 ? 'filled' : 'outlined'}
              disabled={disabled}
            />
          </Box>
        </Grid>

        {/* Save Button */}
        {showSaveButton && onSave && (
          <Grid item xs={12}>
            <Button
              variant="contained"
              startIcon={<SaveIcon />}
              onClick={onSave}
              disabled={disabled || !hasChanges}
              fullWidth
            >
              Apply Traffic Changes
            </Button>
          </Grid>
        )}
      </Grid>

      {hasChanges && !showSaveButton && (
        <Box mt={2}>
          <Typography variant="caption" color="warning.main">
            Traffic weights have changed. Make sure to save your changes.
          </Typography>
        </Box>
      )}
    </Paper>
  );
};

export default TrafficSlider;

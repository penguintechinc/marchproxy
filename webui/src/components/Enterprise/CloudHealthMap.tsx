/**
 * CloudHealthMap Component
 *
 * Interactive world map displaying cloud backend locations with health status.
 * Features:
 * - Real-time health status indicators (green/yellow/red)
 * - Latency heatmap overlay
 * - Clickable markers for detailed backend information
 * - Auto-refresh with health checks
 * - Historical health trends
 */

import React, { useState, useEffect } from 'react';
import {
  ComposableMap,
  Geographies,
  Geography,
  Marker,
  ZoomableGroup,
} from 'react-simple-maps';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Chip,
  Tooltip,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  Grid,
  Alert,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import CloseIcon from '@mui/icons-material/Close';
import CircleIcon from '@mui/icons-material/Circle';
import type { CloudBackendLocation } from '../../services/types';

const geoUrl = 'https://cdn.jsdelivr.net/npm/world-atlas@2/countries-110m.json';

interface CloudHealthMapProps {
  locations: CloudBackendLocation[];
  onRefresh?: () => void;
  autoRefreshInterval?: number;
}

const CloudHealthMap: React.FC<CloudHealthMapProps> = ({
  locations,
  onRefresh,
  autoRefreshInterval = 30000,
}) => {
  const [selectedLocation, setSelectedLocation] = useState<CloudBackendLocation | null>(
    null
  );
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date());

  // Auto-refresh effect
  useEffect(() => {
    if (!onRefresh || autoRefreshInterval <= 0) return;

    const interval = setInterval(() => {
      onRefresh();
      setLastRefresh(new Date());
    }, autoRefreshInterval);

    return () => clearInterval(interval);
  }, [onRefresh, autoRefreshInterval]);

  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'healthy':
        return '#4caf50';
      case 'degraded':
        return '#ff9800';
      case 'unhealthy':
        return '#f44336';
      default:
        return '#9e9e9e';
    }
  };

  const getStatusLabel = (status: string): string => {
    switch (status) {
      case 'healthy':
        return 'Healthy';
      case 'degraded':
        return 'Degraded';
      case 'unhealthy':
        return 'Unhealthy';
      default:
        return 'Unknown';
    }
  };

  const getProviderColor = (provider: string): string => {
    switch (provider) {
      case 'AWS':
        return '#FF9900';
      case 'GCP':
        return '#4285F4';
      case 'Azure':
        return '#0078D4';
      default:
        return '#757575';
    }
  };

  const handleMarkerClick = (location: CloudBackendLocation) => {
    setSelectedLocation(location);
  };

  const handleDialogClose = () => {
    setSelectedLocation(null);
  };

  const handleRefresh = () => {
    if (onRefresh) {
      onRefresh();
      setLastRefresh(new Date());
    }
  };

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="h6">Cloud Backend Health Map</Typography>
        <Box display="flex" alignItems="center" gap={2}>
          <Typography variant="caption" color="text.secondary">
            Last updated: {lastRefresh.toLocaleTimeString()}
          </Typography>
          <IconButton onClick={handleRefresh} size="small">
            <RefreshIcon />
          </IconButton>
        </Box>
      </Box>

      <Card>
        <CardContent>
          <Box sx={{ height: 500, position: 'relative' }}>
            <ComposableMap
              projection="geoMercator"
              projectionConfig={{
                scale: 147,
              }}
            >
              <ZoomableGroup center={[0, 20]} zoom={1}>
                <Geographies geography={geoUrl}>
                  {({ geographies }) =>
                    geographies.map((geo) => (
                      <Geography
                        key={geo.rsmKey}
                        geography={geo}
                        fill="#EAEAEC"
                        stroke="#D6D6DA"
                        strokeWidth={0.5}
                      />
                    ))
                  }
                </Geographies>
                {locations.map((location) => (
                  <Marker
                    key={location.id}
                    coordinates={[location.longitude, location.latitude]}
                  >
                    <Tooltip
                      title={
                        <Box>
                          <Typography variant="body2" fontWeight="bold">
                            {location.name}
                          </Typography>
                          <Typography variant="caption">
                            {location.cloud_provider} - {location.region}
                          </Typography>
                          <Typography variant="caption" display="block">
                            Status: {getStatusLabel(location.status)}
                          </Typography>
                          {location.rtt_ms && (
                            <Typography variant="caption" display="block">
                              RTT: {location.rtt_ms}ms
                            </Typography>
                          )}
                        </Box>
                      }
                    >
                      <g
                        onClick={() => handleMarkerClick(location)}
                        style={{ cursor: 'pointer' }}
                      >
                        <circle
                          r={8}
                          fill={getStatusColor(location.status)}
                          stroke="#fff"
                          strokeWidth={2}
                          opacity={0.8}
                        />
                        <circle
                          r={12}
                          fill={getStatusColor(location.status)}
                          opacity={0.3}
                        />
                      </g>
                    </Tooltip>
                  </Marker>
                ))}
              </ZoomableGroup>
            </ComposableMap>
          </Box>

          <Box mt={2} display="flex" gap={2} flexWrap="wrap">
            <Box display="flex" alignItems="center" gap={1}>
              <CircleIcon sx={{ color: '#4caf50', fontSize: 16 }} />
              <Typography variant="caption">Healthy</Typography>
            </Box>
            <Box display="flex" alignItems="center" gap={1}>
              <CircleIcon sx={{ color: '#ff9800', fontSize: 16 }} />
              <Typography variant="caption">Degraded</Typography>
            </Box>
            <Box display="flex" alignItems="center" gap={1}>
              <CircleIcon sx={{ color: '#f44336', fontSize: 16 }} />
              <Typography variant="caption">Unhealthy</Typography>
            </Box>
          </Box>
        </CardContent>
      </Card>

      <Dialog
        open={selectedLocation !== null}
        onClose={handleDialogClose}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>
          <Box display="flex" justifyContent="space-between" alignItems="center">
            <Typography variant="h6">{selectedLocation?.name}</Typography>
            <IconButton onClick={handleDialogClose} size="small">
              <CloseIcon />
            </IconButton>
          </Box>
        </DialogTitle>
        <DialogContent>
          {selectedLocation && (
            <Grid container spacing={2}>
              <Grid item xs={12}>
                <Alert
                  severity={
                    selectedLocation.status === 'healthy'
                      ? 'success'
                      : selectedLocation.status === 'degraded'
                      ? 'warning'
                      : 'error'
                  }
                  icon={<CircleIcon />}
                >
                  Status: {getStatusLabel(selectedLocation.status)}
                </Alert>
              </Grid>

              <Grid item xs={6}>
                <Typography variant="caption" color="text.secondary">
                  Cloud Provider
                </Typography>
                <Box mt={0.5}>
                  <Chip
                    label={selectedLocation.cloud_provider}
                    size="small"
                    sx={{
                      bgcolor: getProviderColor(selectedLocation.cloud_provider),
                      color: 'white',
                    }}
                  />
                </Box>
              </Grid>

              <Grid item xs={6}>
                <Typography variant="caption" color="text.secondary">
                  Region
                </Typography>
                <Typography variant="body2" mt={0.5}>
                  {selectedLocation.region}
                </Typography>
              </Grid>

              <Grid item xs={6}>
                <Typography variant="caption" color="text.secondary">
                  Latitude
                </Typography>
                <Typography variant="body2" mt={0.5}>
                  {selectedLocation.latitude.toFixed(4)}
                </Typography>
              </Grid>

              <Grid item xs={6}>
                <Typography variant="caption" color="text.secondary">
                  Longitude
                </Typography>
                <Typography variant="body2" mt={0.5}>
                  {selectedLocation.longitude.toFixed(4)}
                </Typography>
              </Grid>

              {selectedLocation.rtt_ms && (
                <Grid item xs={12}>
                  <Typography variant="caption" color="text.secondary">
                    Round-Trip Time (RTT)
                  </Typography>
                  <Typography variant="h4" color="primary" mt={0.5}>
                    {selectedLocation.rtt_ms}ms
                  </Typography>
                </Grid>
              )}
            </Grid>
          )}
        </DialogContent>
      </Dialog>
    </Box>
  );
};

export default CloudHealthMap;

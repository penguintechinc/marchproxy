/**
 * ModuleCard Component
 *
 * Displays status, health, and metrics for a single module (NLB, ALB, DBLB, AILB, RTMP).
 * Provides quick actions for enable/disable, configure routes, and view metrics.
 */

import React from 'react';
import {
  Card,
  CardContent,
  CardActions,
  Typography,
  Box,
  Chip,
  IconButton,
  Button,
  LinearProgress,
  Tooltip,
  Grid,
} from '@mui/material';
import {
  Settings as SettingsIcon,
  PowerSettingsNew as PowerIcon,
  ShowChart as MetricsIcon,
  Route as RouteIcon,
  Cloud as CloudIcon,
  Storage as StorageIcon,
  SmartToy as AIIcon,
  VideoLibrary as VideoIcon,
  NetworkCheck as NetworkIcon,
} from '@mui/icons-material';
import type { Module, ModuleMetrics } from '../../services/types';

interface ModuleCardProps {
  module: Module;
  metrics?: ModuleMetrics;
  onEnable: (id: number) => void;
  onDisable: (id: number) => void;
  onConfigure: (id: number) => void;
  onViewMetrics: (id: number) => void;
  loading?: boolean;
}

const MODULE_ICONS: Record<Module['type'], React.ReactElement> = {
  NLB: <NetworkIcon fontSize="large" />,
  ALB: <CloudIcon fontSize="large" />,
  DBLB: <StorageIcon fontSize="large" />,
  AILB: <AIIcon fontSize="large" />,
  RTMP: <VideoIcon fontSize="large" />,
};

const MODULE_COLORS: Record<Module['type'], string> = {
  NLB: '#1976D2',
  ALB: '#2E7D32',
  DBLB: '#ED6C02',
  AILB: '#9C27B0',
  RTMP: '#D32F2F',
};

const ModuleCard: React.FC<ModuleCardProps> = ({
  module,
  metrics,
  onEnable,
  onDisable,
  onConfigure,
  onViewMetrics,
  loading = false,
}) => {
  const getHealthColor = (status: Module['health_status']): string => {
    switch (status) {
      case 'healthy':
        return 'success';
      case 'degraded':
        return 'warning';
      case 'unhealthy':
        return 'error';
      default:
        return 'default';
    }
  };

  const formatMetric = (value: number | undefined, suffix: string = ''): string => {
    if (value === undefined) return 'N/A';
    return `${value.toFixed(1)}${suffix}`;
  };

  return (
    <Card
      sx={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        borderTop: 3,
        borderColor: MODULE_COLORS[module.type],
        opacity: module.is_enabled ? 1 : 0.7,
        transition: 'all 0.3s',
        '&:hover': {
          boxShadow: 6,
        },
      }}
    >
      <CardContent sx={{ flexGrow: 1, pb: 1 }}>
        {loading && <LinearProgress />}

        <Box display="flex" alignItems="center" justifyContent="space-between" mb={2}>
          <Box display="flex" alignItems="center" gap={2}>
            <Box sx={{ color: MODULE_COLORS[module.type] }}>
              {MODULE_ICONS[module.type]}
            </Box>
            <Box>
              <Typography variant="h6" fontWeight="bold">
                {module.name}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {module.type} Module
              </Typography>
            </Box>
          </Box>

          <Box display="flex" gap={1}>
            <Chip
              label={module.is_enabled ? 'Enabled' : 'Disabled'}
              color={module.is_enabled ? 'success' : 'default'}
              size="small"
            />
            <Chip
              label={module.health_status}
              color={getHealthColor(module.health_status) as any}
              size="small"
            />
          </Box>
        </Box>

        <Typography variant="body2" color="text.secondary" mb={2}>
          {module.description}
        </Typography>

        <Box
          sx={{
            bgcolor: 'background.default',
            p: 1.5,
            borderRadius: 1,
            mb: 1,
          }}
        >
          <Grid container spacing={1}>
            <Grid item xs={6}>
              <Typography variant="caption" color="text.secondary" display="block">
                Version
              </Typography>
              <Typography variant="body2" fontWeight="medium">
                {module.version || 'N/A'}
              </Typography>
            </Grid>
            <Grid item xs={6}>
              <Typography variant="caption" color="text.secondary" display="block">
                gRPC Address
              </Typography>
              <Typography variant="body2" fontWeight="medium" noWrap>
                {module.grpc_address || 'N/A'}
              </Typography>
            </Grid>
          </Grid>
        </Box>

        {metrics && module.is_enabled && (
          <Box
            sx={{
              bgcolor: 'background.default',
              p: 1.5,
              borderRadius: 1,
            }}
          >
            <Grid container spacing={1}>
              <Grid item xs={6}>
                <Typography variant="caption" color="text.secondary" display="block">
                  CPU Usage
                </Typography>
                <Typography variant="body2" fontWeight="medium">
                  {formatMetric(metrics.cpu_percent, '%')}
                </Typography>
              </Grid>
              <Grid item xs={6}>
                <Typography variant="caption" color="text.secondary" display="block">
                  Memory Usage
                </Typography>
                <Typography variant="body2" fontWeight="medium">
                  {formatMetric(metrics.memory_percent, '%')}
                </Typography>
              </Grid>
              <Grid item xs={6}>
                <Typography variant="caption" color="text.secondary" display="block">
                  Active Connections
                </Typography>
                <Typography variant="body2" fontWeight="medium">
                  {metrics.active_connections || 0}
                </Typography>
              </Grid>
              <Grid item xs={6}>
                <Typography variant="caption" color="text.secondary" display="block">
                  Requests/sec
                </Typography>
                <Typography variant="body2" fontWeight="medium">
                  {formatMetric(metrics.requests_per_second)}
                </Typography>
              </Grid>
            </Grid>
          </Box>
        )}
      </CardContent>

      <CardActions sx={{ justifyContent: 'space-between', px: 2, pb: 2 }}>
        <Box display="flex" gap={1}>
          <Tooltip title={module.is_enabled ? 'Disable module' : 'Enable module'}>
            <IconButton
              size="small"
              color={module.is_enabled ? 'error' : 'success'}
              onClick={() =>
                module.is_enabled ? onDisable(module.id) : onEnable(module.id)
              }
              disabled={loading}
            >
              <PowerIcon />
            </IconButton>
          </Tooltip>

          <Tooltip title="Configure routes">
            <IconButton
              size="small"
              onClick={() => onConfigure(module.id)}
              disabled={loading || !module.is_enabled}
            >
              <RouteIcon />
            </IconButton>
          </Tooltip>

          <Tooltip title="View metrics">
            <IconButton
              size="small"
              onClick={() => onViewMetrics(module.id)}
              disabled={loading || !module.is_enabled}
            >
              <MetricsIcon />
            </IconButton>
          </Tooltip>
        </Box>

        <Button
          size="small"
          startIcon={<SettingsIcon />}
          onClick={() => onConfigure(module.id)}
          disabled={loading || !module.is_enabled}
        >
          Configure
        </Button>
      </CardActions>
    </Card>
  );
};

export default ModuleCard;

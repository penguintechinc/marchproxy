/**
 * Kong Dashboard - Status and Statistics Overview
 */
import React, { useEffect, useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  Grid,
  Typography,
  CircularProgress,
  Alert,
  Chip,
} from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import DnsIcon from '@mui/icons-material/Dns';
import RouteIcon from '@mui/icons-material/AltRoute';
import StorageIcon from '@mui/icons-material/Storage';
import PeopleIcon from '@mui/icons-material/People';
import ExtensionIcon from '@mui/icons-material/Extension';
import SecurityIcon from '@mui/icons-material/Security';
import { kongApi, KongStatus } from '../../services/kongApi';

interface EntityCount {
  services: number;
  routes: number;
  upstreams: number;
  consumers: number;
  plugins: number;
  certificates: number;
}

const KongDashboard: React.FC = () => {
  const [status, setStatus] = useState<KongStatus | null>(null);
  const [counts, setCounts] = useState<EntityCount | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = async () => {
    setLoading(true);
    setError(null);
    try {
      const [statusRes, servicesRes, routesRes, upstreamsRes, consumersRes, pluginsRes, certsRes] =
        await Promise.all([
          kongApi.getStatus(),
          kongApi.getServices(),
          kongApi.getRoutes(),
          kongApi.getUpstreams(),
          kongApi.getConsumers(),
          kongApi.getPlugins(),
          kongApi.getCertificates(),
        ]);

      setStatus(statusRes.data);
      setCounts({
        services: servicesRes.data.data?.length || 0,
        routes: routesRes.data.data?.length || 0,
        upstreams: upstreamsRes.data.data?.length || 0,
        consumers: consumersRes.data.data?.length || 0,
        plugins: pluginsRes.data.data?.length || 0,
        certificates: certsRes.data.data?.length || 0,
      });
    } catch (err) {
      setError('Failed to connect to Kong Admin API');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 30000); // Refresh every 30s
    return () => clearInterval(interval);
  }, []);

  if (loading && !status) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight={400}>
        <CircularProgress />
      </Box>
    );
  }

  const StatCard: React.FC<{ title: string; value: number; icon: React.ReactNode; color: string }> = ({
    title,
    value,
    icon,
    color,
  }) => (
    <Card>
      <CardContent>
        <Box display="flex" alignItems="center" justifyContent="space-between">
          <Box>
            <Typography color="textSecondary" gutterBottom variant="body2">
              {title}
            </Typography>
            <Typography variant="h4">{value}</Typography>
          </Box>
          <Box sx={{ color, opacity: 0.7 }}>{icon}</Box>
        </Box>
      </CardContent>
    </Card>
  );

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        Kong Gateway Dashboard
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      {/* Status Card */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box display="flex" alignItems="center" gap={2}>
            <Typography variant="h6">Gateway Status</Typography>
            {status?.database?.reachable ? (
              <Chip
                icon={<CheckCircleIcon />}
                label="Connected"
                color="success"
                size="small"
              />
            ) : (
              <Chip
                icon={<ErrorIcon />}
                label="Disconnected"
                color="error"
                size="small"
              />
            )}
          </Box>
          {status && (
            <Box mt={2}>
              <Typography variant="body2" color="textSecondary">
                Active Connections: Reading {status.server?.connections_reading || 0}, Writing{' '}
                {status.server?.connections_writing || 0}
              </Typography>
            </Box>
          )}
        </CardContent>
      </Card>

      {/* Entity Counts Grid */}
      {counts && (
        <Grid container spacing={3}>
          <Grid item xs={12} sm={6} md={4}>
            <StatCard
              title="Services"
              value={counts.services}
              icon={<DnsIcon sx={{ fontSize: 48 }} />}
              color="#2196f3"
            />
          </Grid>
          <Grid item xs={12} sm={6} md={4}>
            <StatCard
              title="Routes"
              value={counts.routes}
              icon={<RouteIcon sx={{ fontSize: 48 }} />}
              color="#4caf50"
            />
          </Grid>
          <Grid item xs={12} sm={6} md={4}>
            <StatCard
              title="Upstreams"
              value={counts.upstreams}
              icon={<StorageIcon sx={{ fontSize: 48 }} />}
              color="#ff9800"
            />
          </Grid>
          <Grid item xs={12} sm={6} md={4}>
            <StatCard
              title="Consumers"
              value={counts.consumers}
              icon={<PeopleIcon sx={{ fontSize: 48 }} />}
              color="#9c27b0"
            />
          </Grid>
          <Grid item xs={12} sm={6} md={4}>
            <StatCard
              title="Plugins"
              value={counts.plugins}
              icon={<ExtensionIcon sx={{ fontSize: 48 }} />}
              color="#00bcd4"
            />
          </Grid>
          <Grid item xs={12} sm={6} md={4}>
            <StatCard
              title="Certificates"
              value={counts.certificates}
              icon={<SecurityIcon sx={{ fontSize: 48 }} />}
              color="#f44336"
            />
          </Grid>
        </Grid>
      )}
    </Box>
  );
};

export default KongDashboard;
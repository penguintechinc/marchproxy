/**
 * Multi-Cloud Routing Page
 *
 * Enterprise feature for intelligent cloud routing with health monitoring.
 */

import React, { useEffect, useState } from 'react';
import {
  Box,
  Container,
  Typography,
  Button,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Chip,
  Card,
  CardContent,
  Grid,
  Alert,
  LinearProgress
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  CloudQueue as CloudIcon,
  CheckCircle as HealthyIcon,
  Error as UnhealthyIcon,
  Assessment as AnalyticsIcon
} from '@mui/icons-material';

import LicenseGate from '../../components/Common/LicenseGate';
import { useMultiCloud } from '../../hooks/useMultiCloud';

const MultiCloudRouting: React.FC = () => {
  const {
    routeTables,
    loading,
    error,
    hasAccess,
    fetchRouteTables,
    deleteRouteTable,
    getCostAnalytics
  } = useMultiCloud();

  const [costData, setCostData] = useState<any>(null);

  useEffect(() => {
    fetchRouteTables();
    loadCostAnalytics();
  }, [fetchRouteTables]);

  const loadCostAnalytics = async () => {
    const data = await getCostAnalytics({ days: 7 });
    setCostData(data);
  };

  const handleDelete = async (id: number) => {
    if (window.confirm('Are you sure you want to delete this route table?')) {
      await deleteRouteTable(id);
    }
  };

  const getAlgorithmLabel = (algo: string) => {
    const labels: Record<string, string> = {
      latency: 'Latency-Based',
      cost: 'Cost-Optimized',
      geo: 'Geo-Proximity',
      weighted_rr: 'Weighted Round-Robin',
      failover: 'Active-Passive'
    };
    return labels[algo] || algo;
  };

  const getProviderColor = (provider: string) => {
    const colors: Record<string, any> = {
      aws: 'warning',
      gcp: 'info',
      azure: 'primary',
      on_prem: 'default'
    };
    return colors[provider] || 'default';
  };

  return (
    <LicenseGate
      featureName="Multi-Cloud Intelligent Routing"
      hasAccess={hasAccess}
      isLoading={loading && routeTables.length === 0}
    >
      <Container maxWidth="xl" sx={{ mt: 4, mb: 4 }}>
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
          <Box>
            <Typography variant="h4" gutterBottom>
              <CloudIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
              Multi-Cloud Routing
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Intelligent routing across AWS, GCP, Azure, and on-premise infrastructure
            </Typography>
          </Box>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => alert('Create route table dialog - implementation placeholder')}
          >
            Create Route Table
          </Button>
        </Box>

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {/* Cost Analytics Dashboard */}
        <Grid container spacing={3} mb={3}>
          <Grid item xs={12} md={3}>
            <Card>
              <CardContent>
                <Typography variant="h6" color="text.secondary" gutterBottom>
                  Total Cost (7d)
                </Typography>
                <Typography variant="h4">
                  ${costData?.total_cost_usd?.toFixed(2) || '0.00'}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={3}>
            <Card>
              <CardContent>
                <Typography variant="h6" color="text.secondary" gutterBottom>
                  Active Routes
                </Typography>
                <Typography variant="h4">
                  {routeTables.reduce((acc, rt) =>
                    acc + rt.routes.filter(r => r.is_active).length, 0
                  )}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={3}>
            <Card>
              <CardContent>
                <Typography variant="h6" color="text.secondary" gutterBottom>
                  Healthy Endpoints
                </Typography>
                <Typography variant="h4" color="success.main">
                  {routeTables.reduce((acc, rt) =>
                    acc + (rt.health_status?.filter(h => h.is_healthy).length || 0), 0
                  )}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={3}>
            <Card>
              <CardContent>
                <Typography variant="h6" color="text.secondary" gutterBottom>
                  Avg RTT
                </Typography>
                <Typography variant="h4">
                  {routeTables.reduce((acc, rt) => {
                    const avgRtt = rt.health_status?.reduce((sum, h) =>
                      sum + (h.rtt_ms || 0), 0
                    ) || 0;
                    return acc + (avgRtt / (rt.health_status?.length || 1));
                  }, 0).toFixed(1)} ms
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>

        {/* Route Tables */}
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Algorithm</TableCell>
                <TableCell>Routes</TableCell>
                <TableCell>Health Status</TableCell>
                <TableCell>Auto Failover</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {routeTables.map((table) => (
                <TableRow key={table.id}>
                  <TableCell>
                    <Typography variant="body2" fontWeight="bold">
                      {table.name}
                    </Typography>
                    {table.description && (
                      <Typography variant="caption" color="text.secondary">
                        {table.description}
                      </Typography>
                    )}
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={getAlgorithmLabel(table.algorithm)}
                      size="small"
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell>
                    <Box display="flex" gap={0.5} flexWrap="wrap">
                      {table.routes.map((route, idx) => (
                        <Chip
                          key={idx}
                          label={`${route.provider.toUpperCase()} ${route.region}`}
                          size="small"
                          color={getProviderColor(route.provider)}
                          icon={route.is_active ? <HealthyIcon /> : <UnhealthyIcon />}
                        />
                      ))}
                    </Box>
                  </TableCell>
                  <TableCell>
                    {table.health_status ? (
                      <Box>
                        {table.health_status.filter(h => h.is_healthy).length} /{' '}
                        {table.health_status.length} healthy
                        <LinearProgress
                          variant="determinate"
                          value={
                            (table.health_status.filter(h => h.is_healthy).length /
                              table.health_status.length) * 100
                          }
                          color={
                            table.health_status.every(h => h.is_healthy)
                              ? 'success'
                              : 'warning'
                          }
                          sx={{ mt: 1 }}
                        />
                      </Box>
                    ) : (
                      <Typography variant="caption" color="text.secondary">
                        No data
                      </Typography>
                    )}
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={table.enable_auto_failover ? 'Enabled' : 'Disabled'}
                      color={table.enable_auto_failover ? 'success' : 'default'}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    <IconButton
                      size="small"
                      onClick={() => alert('Edit dialog - placeholder')}
                      title="Edit"
                    >
                      <EditIcon />
                    </IconButton>
                    <IconButton
                      size="small"
                      onClick={() => alert('View analytics - placeholder')}
                      title="Analytics"
                    >
                      <AnalyticsIcon />
                    </IconButton>
                    <IconButton
                      size="small"
                      onClick={() => handleDelete(table.id)}
                      title="Delete"
                    >
                      <DeleteIcon />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
              {routeTables.length === 0 && !loading && (
                <TableRow>
                  <TableCell colSpan={6} align="center">
                    <Typography variant="body2" color="text.secondary" py={4}>
                      No route tables configured. Click "Create Route Table" to get started.
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>

        {/* Cost Breakdown by Provider */}
        {costData && Object.keys(costData.by_provider || {}).length > 0 && (
          <Paper sx={{ mt: 3, p: 3 }}>
            <Typography variant="h6" gutterBottom>
              Cost Breakdown by Provider
            </Typography>
            <Grid container spacing={2}>
              {Object.entries(costData.by_provider || {}).map(([provider, cost]: any) => (
                <Grid item xs={12} sm={6} md={3} key={provider}>
                  <Card variant="outlined">
                    <CardContent>
                      <Chip
                        label={provider.toUpperCase()}
                        color={getProviderColor(provider)}
                        size="small"
                        sx={{ mb: 1 }}
                      />
                      <Typography variant="h5">
                        ${cost.toFixed(2)}
                      </Typography>
                    </CardContent>
                  </Card>
                </Grid>
              ))}
            </Grid>
          </Paper>
        )}
      </Container>
    </LicenseGate>
  );
};

export default MultiCloudRouting;

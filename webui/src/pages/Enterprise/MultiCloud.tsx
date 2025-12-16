/**
 * Multi-Cloud Routing Dashboard
 *
 * Enterprise feature for intelligent multi-cloud routing:
 * - Route table editor (AWS, GCP, Azure)
 * - Backend health visualization
 * - RTT measurement display
 * - Cost analytics dashboard
 * - Routing algorithm configuration
 * - Failover management
 */

import React, { useState, useEffect } from 'react';
import {
  Box,
  Container,
  Typography,
  Button,
  Card,
  CardContent,
  Grid,
  TextField,
  MenuItem,
  Alert,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Chip,
  Tabs,
  Tab,
  Select,
  FormControl,
  InputLabel,
} from '@mui/material';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip as RechartsTooltip, Legend, ResponsiveContainer } from 'recharts';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import SaveIcon from '@mui/icons-material/Save';
import CloudIcon from '@mui/icons-material/Cloud';
import LicenseGate from '../../components/Common/LicenseGate';
import CloudHealthMap from '../../components/Enterprise/CloudHealthMap';
import {
  getCloudRoutes,
  createCloudRoute,
  updateCloudRoute,
  deleteCloudRoute,
  getBackendHealth,
  getCloudBackendLocations,
  getRoutingAlgorithms,
  updateRoutingAlgorithm,
} from '../../services/enterpriseApi';
import type {
  CloudRoute,
  BackendHealth,
  CloudBackendLocation,
  RoutingAlgorithm,
} from '../../services/types';
import { useLicense } from '../../hooks/useLicense';

const CLOUD_PROVIDERS = [
  { value: 'AWS', label: 'Amazon Web Services', color: '#FF9900' },
  { value: 'GCP', label: 'Google Cloud Platform', color: '#4285F4' },
  { value: 'Azure', label: 'Microsoft Azure', color: '#0078D4' },
  { value: 'Custom', label: 'Custom Backend', color: '#757575' },
];

const MultiCloud: React.FC = () => {
  const { isEnterprise, hasFeature, loading: licenseLoading } = useLicense();
  const hasEnterpriseAccess = isEnterprise || hasFeature('multi_cloud_routing');
  const [activeTab, setActiveTab] = useState(0);
  const [routes, setRoutes] = useState<CloudRoute[]>([]);
  const [backendHealth, setBackendHealth] = useState<BackendHealth[]>([]);
  const [locations, setLocations] = useState<CloudBackendLocation[]>([]);
  const [algorithms, setAlgorithms] = useState<RoutingAlgorithm[]>([]);
  const [selectedAlgorithm, setSelectedAlgorithm] = useState<string>('latency');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingRoute, setEditingRoute] = useState<Partial<CloudRoute> | null>(null);

  const [formData, setFormData] = useState<Partial<CloudRoute>>({
    cloud_provider: 'AWS',
    weight: 100,
    priority: 1,
    health_check_interval_seconds: 30,
    is_active: true,
  });

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const [routesData, healthData, locationsData, algorithmsData] = await Promise.all([
        getCloudRoutes(),
        getBackendHealth(),
        getCloudBackendLocations(),
        getRoutingAlgorithms(),
      ]);
      setRoutes(routesData);
      setBackendHealth(healthData);
      setLocations(locationsData);
      setAlgorithms(algorithmsData);
      setError(null);
    } catch (err) {
      setError('Failed to load multi-cloud routing data');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleOpenDialog = (route?: CloudRoute) => {
    if (route) {
      setEditingRoute(route);
      setFormData(route);
    } else {
      setEditingRoute(null);
      setFormData({
        cloud_provider: 'AWS',
        weight: 100,
        priority: 1,
        health_check_interval_seconds: 30,
        is_active: true,
      });
    }
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
    setEditingRoute(null);
  };

  const handleSaveRoute = async () => {
    try {
      if (editingRoute && editingRoute.id) {
        await updateCloudRoute(editingRoute.id, formData);
      } else {
        await createCloudRoute(formData);
      }
      await loadData();
      handleCloseDialog();
    } catch (err) {
      setError('Failed to save cloud route');
      console.error(err);
    }
  };

  const handleDeleteRoute = async (id: number) => {
    if (!confirm('Are you sure you want to delete this route?')) return;
    try {
      await deleteCloudRoute(id);
      await loadData();
    } catch (err) {
      setError('Failed to delete route');
      console.error(err);
    }
  };

  const handleAlgorithmChange = async (algorithm: string) => {
    try {
      setSelectedAlgorithm(algorithm);
      // Update for all services - in production, this would be per-service
      await updateRoutingAlgorithm(1, algorithm);
    } catch (err) {
      setError('Failed to update routing algorithm');
      console.error(err);
    }
  };

  const getStatusColor = (status: string): string => {
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

  const getProviderColor = (provider: string): string => {
    return CLOUD_PROVIDERS.find((p) => p.value === provider)?.color || '#757575';
  };

  const rttChartData = routes
    .filter((r) => r.rtt_ms)
    .map((r) => ({
      name: `${r.cloud_provider} - ${r.region}`,
      rtt: r.rtt_ms,
      fill: getProviderColor(r.cloud_provider),
    }));

  return (
    <LicenseGate
      featureName="Multi-Cloud Intelligent Routing"
      hasAccess={hasEnterpriseAccess}
      isLoading={licenseLoading}
    >
      <Container maxWidth="xl">
        <Box py={4}>
          <Box display="flex" justifyContent="space-between" alignItems="center" mb={4}>
            <Typography variant="h4" fontWeight="bold">
              Multi-Cloud Routing
            </Typography>
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={() => handleOpenDialog()}
            >
              Add Route
            </Button>
          </Box>

          {error && (
            <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
              {error}
            </Alert>
          )}

          <Box mb={3}>
            <Card>
              <CardContent>
                <Box display="flex" justifyContent="space-between" alignItems="center">
                  <Typography variant="h6">Routing Algorithm</Typography>
                  <FormControl sx={{ minWidth: 250 }}>
                    <InputLabel>Algorithm</InputLabel>
                    <Select
                      value={selectedAlgorithm}
                      onChange={(e) => handleAlgorithmChange(e.target.value)}
                      label="Algorithm"
                    >
                      {algorithms.map((algo) => (
                        <MenuItem key={algo.type} value={algo.type}>
                          {algo.name}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Box>
                {algorithms.find((a) => a.type === selectedAlgorithm)?.description && (
                  <Typography variant="body2" color="text.secondary" mt={2}>
                    {algorithms.find((a) => a.type === selectedAlgorithm)?.description}
                  </Typography>
                )}
              </CardContent>
            </Card>
          </Box>

          <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)} sx={{ mb: 3 }}>
            <Tab label="Routes" />
            <Tab label="Health Map" />
            <Tab label="Performance" />
          </Tabs>

          {activeTab === 0 && (
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Cloud Routes
                </Typography>
                <TableContainer>
                  <Table>
                    <TableHead>
                      <TableRow>
                        <TableCell>Provider</TableCell>
                        <TableCell>Region</TableCell>
                        <TableCell>Backend URL</TableCell>
                        <TableCell>Weight</TableCell>
                        <TableCell>Priority</TableCell>
                        <TableCell>Health Status</TableCell>
                        <TableCell>RTT</TableCell>
                        <TableCell>Active</TableCell>
                        <TableCell align="right">Actions</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {routes.map((route) => (
                        <TableRow key={route.id}>
                          <TableCell>
                            <Chip
                              icon={<CloudIcon />}
                              label={route.cloud_provider}
                              size="small"
                              sx={{
                                bgcolor: getProviderColor(route.cloud_provider),
                                color: 'white',
                              }}
                            />
                          </TableCell>
                          <TableCell>{route.region}</TableCell>
                          <TableCell>{route.backend_url}</TableCell>
                          <TableCell>{route.weight}</TableCell>
                          <TableCell>{route.priority}</TableCell>
                          <TableCell>
                            <Chip
                              label={route.health_status}
                              size="small"
                              color={getStatusColor(route.health_status) as any}
                            />
                          </TableCell>
                          <TableCell>{route.rtt_ms ? `${route.rtt_ms}ms` : '-'}</TableCell>
                          <TableCell>
                            <Chip
                              label={route.is_active ? 'Yes' : 'No'}
                              size="small"
                              color={route.is_active ? 'success' : 'default'}
                            />
                          </TableCell>
                          <TableCell align="right">
                            <IconButton
                              size="small"
                              onClick={() => handleOpenDialog(route)}
                            >
                              <EditIcon />
                            </IconButton>
                            <IconButton
                              size="small"
                              onClick={() => handleDeleteRoute(route.id)}
                            >
                              <DeleteIcon />
                            </IconButton>
                          </TableCell>
                        </TableRow>
                      ))}
                      {routes.length === 0 && (
                        <TableRow>
                          <TableCell colSpan={9} align="center">
                            <Typography variant="body2" color="text.secondary" py={2}>
                              No routes configured
                            </Typography>
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </TableContainer>
              </CardContent>
            </Card>
          )}

          {activeTab === 1 && (
            <CloudHealthMap locations={locations} onRefresh={loadData} />
          )}

          {activeTab === 2 && (
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Backend RTT Performance
                </Typography>
                {rttChartData.length > 0 ? (
                  <ResponsiveContainer width="100%" height={400}>
                    <BarChart data={rttChartData}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="name" angle={-45} textAnchor="end" height={100} />
                      <YAxis label={{ value: 'RTT (ms)', angle: -90, position: 'insideLeft' }} />
                      <RechartsTooltip />
                      <Legend />
                      <Bar dataKey="rtt" name="Round-Trip Time (ms)" />
                    </BarChart>
                  </ResponsiveContainer>
                ) : (
                  <Box textAlign="center" py={4}>
                    <Typography variant="body2" color="text.secondary">
                      No RTT data available
                    </Typography>
                  </Box>
                )}
              </CardContent>
            </Card>
          )}
        </Box>

        <Dialog open={dialogOpen} onClose={handleCloseDialog} maxWidth="md" fullWidth>
          <DialogTitle>
            {editingRoute ? 'Edit Cloud Route' : 'Add Cloud Route'}
          </DialogTitle>
          <DialogContent>
            <Box pt={2}>
              <Grid container spacing={3}>
                <Grid item xs={12} sm={6}>
                  <TextField
                    fullWidth
                    label="Service ID"
                    type="number"
                    value={formData.service_id || ''}
                    onChange={(e) =>
                      setFormData({ ...formData, service_id: parseInt(e.target.value) })
                    }
                    required
                  />
                </Grid>

                <Grid item xs={12} sm={6}>
                  <TextField
                    fullWidth
                    select
                    label="Cloud Provider"
                    value={formData.cloud_provider || 'AWS'}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        cloud_provider: e.target.value as any,
                      })
                    }
                  >
                    {CLOUD_PROVIDERS.map((provider) => (
                      <MenuItem key={provider.value} value={provider.value}>
                        {provider.label}
                      </MenuItem>
                    ))}
                  </TextField>
                </Grid>

                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    label="Backend URL"
                    value={formData.backend_url || ''}
                    onChange={(e) =>
                      setFormData({ ...formData, backend_url: e.target.value })
                    }
                    required
                  />
                </Grid>

                <Grid item xs={12} sm={6}>
                  <TextField
                    fullWidth
                    label="Backend IP"
                    value={formData.backend_ip || ''}
                    onChange={(e) =>
                      setFormData({ ...formData, backend_ip: e.target.value })
                    }
                    required
                  />
                </Grid>

                <Grid item xs={12} sm={6}>
                  <TextField
                    fullWidth
                    label="Region"
                    value={formData.region || ''}
                    onChange={(e) =>
                      setFormData({ ...formData, region: e.target.value })
                    }
                    required
                  />
                </Grid>

                <Grid item xs={12} sm={4}>
                  <TextField
                    fullWidth
                    label="Weight"
                    type="number"
                    value={formData.weight || 100}
                    onChange={(e) =>
                      setFormData({ ...formData, weight: parseInt(e.target.value) })
                    }
                    inputProps={{ min: 0, max: 1000 }}
                  />
                </Grid>

                <Grid item xs={12} sm={4}>
                  <TextField
                    fullWidth
                    label="Priority"
                    type="number"
                    value={formData.priority || 1}
                    onChange={(e) =>
                      setFormData({ ...formData, priority: parseInt(e.target.value) })
                    }
                    inputProps={{ min: 1 }}
                  />
                </Grid>

                <Grid item xs={12} sm={4}>
                  <TextField
                    fullWidth
                    label="Health Check Interval (s)"
                    type="number"
                    value={formData.health_check_interval_seconds || 30}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        health_check_interval_seconds: parseInt(e.target.value),
                      })
                    }
                    inputProps={{ min: 5 }}
                  />
                </Grid>

                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    label="Health Check URL (optional)"
                    value={formData.health_check_url || ''}
                    onChange={(e) =>
                      setFormData({ ...formData, health_check_url: e.target.value })
                    }
                  />
                </Grid>
              </Grid>
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={handleCloseDialog}>Cancel</Button>
            <Button
              variant="contained"
              startIcon={<SaveIcon />}
              onClick={handleSaveRoute}
            >
              Save
            </Button>
          </DialogActions>
        </Dialog>
      </Container>
    </LicenseGate>
  );
};

export default MultiCloud;

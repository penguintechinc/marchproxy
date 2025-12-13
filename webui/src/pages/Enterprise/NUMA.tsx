/**
 * NUMA Configuration Page
 *
 * Enterprise feature for configuring Non-Uniform Memory Access:
 * - NUMA topology visualization
 * - CPU affinity configuration
 * - Worker allocation per NUMA node
 * - Performance metrics by NUMA node
 * - Memory locality optimization
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
  FormControlLabel,
  Switch,
  Alert,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  MenuItem,
  LinearProgress,
  Tooltip,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
} from '@mui/material';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip as RechartsTooltip, Legend, ResponsiveContainer } from 'recharts';
import SaveIcon from '@mui/icons-material/Save';
import RefreshIcon from '@mui/icons-material/Refresh';
import RestoreIcon from '@mui/icons-material/Restore';
import MemoryIcon from '@mui/icons-material/Memory';
import SpeedIcon from '@mui/icons-material/Speed';
import LicenseGate from '../../components/Common/LicenseGate';
import {
  getNUMATopology,
  getNUMAConfig,
  updateNUMAConfig,
  getNUMAMetrics,
  resetNUMAConfig,
} from '../../services/enterpriseApi';
import type {
  NUMATopology,
  NUMAConfig,
  NUMAMetrics,
  WorkerAllocation,
  NUMANode,
} from '../../services/types';
import { useLicense } from '../../hooks/useLicense';

const NUMA: React.FC = () => {
  const { isEnterprise, hasFeature, loading: licenseLoading } = useLicense();
  const hasEnterpriseAccess = isEnterprise || hasFeature('numa_optimization');
  const [selectedProxyId, setSelectedProxyId] = useState<number>(1);
  const [topology, setTopology] = useState<NUMATopology | null>(null);
  const [config, setConfig] = useState<NUMAConfig | null>(null);
  const [metrics, setMetrics] = useState<NUMAMetrics[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingNode, setEditingNode] = useState<number | null>(null);
  const [workerCount, setWorkerCount] = useState<number>(0);

  useEffect(() => {
    loadNUMAData();
    const interval = setInterval(loadNUMAData, 30000);
    return () => clearInterval(interval);
  }, [selectedProxyId]);

  const loadNUMAData = async () => {
    try {
      setLoading(true);
      const [topologyData, configData, metricsData] = await Promise.all([
        getNUMATopology(selectedProxyId),
        getNUMAConfig(selectedProxyId),
        getNUMAMetrics(selectedProxyId),
      ]);
      setTopology(topologyData);
      setConfig(configData);
      setMetrics(metricsData);
      setError(null);
    } catch (err) {
      setError('Failed to load NUMA configuration');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleSaveConfig = async () => {
    if (!config) return;
    try {
      await updateNUMAConfig(selectedProxyId, config);
      await loadNUMAData();
      setError(null);
    } catch (err) {
      setError('Failed to save NUMA configuration');
      console.error(err);
    }
  };

  const handleResetConfig = async () => {
    if (!confirm('Are you sure you want to reset NUMA configuration to defaults?')) return;
    try {
      await resetNUMAConfig(selectedProxyId);
      await loadNUMAData();
      setError(null);
    } catch (err) {
      setError('Failed to reset NUMA configuration');
      console.error(err);
    }
  };

  const handleOpenWorkerDialog = (nodeId: number) => {
    setEditingNode(nodeId);
    const allocation = config?.worker_allocation.find((a) => a.numa_node === nodeId);
    setWorkerCount(allocation?.worker_count || 0);
    setDialogOpen(true);
  };

  const handleSaveWorkerAllocation = () => {
    if (!config || editingNode === null) return;

    const newAllocations = [...config.worker_allocation];
    const existingIndex = newAllocations.findIndex((a) => a.numa_node === editingNode);

    if (existingIndex >= 0) {
      newAllocations[existingIndex] = {
        ...newAllocations[existingIndex],
        worker_count: workerCount,
      };
    } else {
      newAllocations.push({
        numa_node: editingNode,
        worker_count: workerCount,
        cpu_pinning: [],
      });
    }

    setConfig({
      ...config,
      worker_allocation: newAllocations,
    });

    setDialogOpen(false);
    setEditingNode(null);
  };

  const getNodeMetrics = (nodeId: number): NUMAMetrics | undefined => {
    return metrics.find((m) => m.node_id === nodeId);
  };

  const getWorkerCount = (nodeId: number): number => {
    return (
      config?.worker_allocation.find((a) => a.numa_node === nodeId)?.worker_count || 0
    );
  };

  const metricsChartData = metrics.map((m) => ({
    node: `Node ${m.node_id}`,
    cpu: m.cpu_usage_percent,
    memory: m.memory_usage_percent,
    local_access: m.local_memory_access_percent,
    throughput: m.throughput_mbps,
  }));

  return (
    <LicenseGate
      featureName="NUMA Performance Optimization"
      hasAccess={hasEnterpriseAccess}
      isLoading={licenseLoading}
    >
      <Container maxWidth="xl">
        <Box py={4}>
          <Box display="flex" justifyContent="space-between" alignItems="center" mb={4}>
            <Typography variant="h4" fontWeight="bold">
              NUMA Configuration
            </Typography>
            <Box display="flex" gap={2}>
              <TextField
                select
                size="small"
                label="Proxy Server"
                value={selectedProxyId}
                onChange={(e) => setSelectedProxyId(Number(e.target.value))}
                sx={{ minWidth: 200 }}
              >
                <MenuItem value={1}>Proxy Server 1</MenuItem>
                <MenuItem value={2}>Proxy Server 2</MenuItem>
                <MenuItem value={3}>Proxy Server 3</MenuItem>
              </TextField>
              <IconButton onClick={loadNUMAData}>
                <RefreshIcon />
              </IconButton>
            </Box>
          </Box>

          {error && (
            <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
              {error}
            </Alert>
          )}

          {topology && config && (
            <>
              <Grid container spacing={3} mb={3}>
                <Grid item xs={12} sm={6} md={3}>
                  <Card>
                    <CardContent>
                      <Box display="flex" alignItems="center" gap={1} mb={1}>
                        <MemoryIcon color="primary" />
                        <Typography variant="caption" color="text.secondary">
                          NUMA Nodes
                        </Typography>
                      </Box>
                      <Typography variant="h4" fontWeight="bold">
                        {topology.node_count}
                      </Typography>
                    </CardContent>
                  </Card>
                </Grid>

                <Grid item xs={12} sm={6} md={3}>
                  <Card>
                    <CardContent>
                      <Box display="flex" alignItems="center" gap={1} mb={1}>
                        <SpeedIcon color="secondary" />
                        <Typography variant="caption" color="text.secondary">
                          Total CPUs
                        </Typography>
                      </Box>
                      <Typography variant="h4" fontWeight="bold">
                        {topology.total_cpus}
                      </Typography>
                    </CardContent>
                  </Card>
                </Grid>

                <Grid item xs={12} sm={6} md={3}>
                  <Card>
                    <CardContent>
                      <Box display="flex" alignItems="center" gap={1} mb={1}>
                        <MemoryIcon color="info" />
                        <Typography variant="caption" color="text.secondary">
                          Total Memory
                        </Typography>
                      </Box>
                      <Typography variant="h4" fontWeight="bold">
                        {topology.total_memory_gb} GB
                      </Typography>
                    </CardContent>
                  </Card>
                </Grid>

                <Grid item xs={12} sm={6} md={3}>
                  <Card>
                    <CardContent>
                      <Box display="flex" alignItems="center" gap={1} mb={1}>
                        <SpeedIcon color="success" />
                        <Typography variant="caption" color="text.secondary">
                          NUMA Status
                        </Typography>
                      </Box>
                      <Chip
                        label={config.enabled ? 'Enabled' : 'Disabled'}
                        color={config.enabled ? 'success' : 'default'}
                        sx={{ mt: 1 }}
                      />
                    </CardContent>
                  </Card>
                </Grid>
              </Grid>

              <Card sx={{ mb: 3 }}>
                <CardContent>
                  <Typography variant="h6" gutterBottom>
                    Global Configuration
                  </Typography>
                  <Grid container spacing={3}>
                    <Grid item xs={12} sm={6}>
                      <FormControlLabel
                        control={
                          <Switch
                            checked={config.enabled}
                            onChange={(e) =>
                              setConfig({ ...config, enabled: e.target.checked })
                            }
                          />
                        }
                        label="Enable NUMA Optimization"
                      />
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <FormControlLabel
                        control={
                          <Switch
                            checked={config.auto_affinity}
                            onChange={(e) =>
                              setConfig({ ...config, auto_affinity: e.target.checked })
                            }
                            disabled={!config.enabled}
                          />
                        }
                        label="Automatic CPU Affinity"
                      />
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <FormControlLabel
                        control={
                          <Switch
                            checked={config.memory_locality_optimization}
                            onChange={(e) =>
                              setConfig({
                                ...config,
                                memory_locality_optimization: e.target.checked,
                              })
                            }
                            disabled={!config.enabled}
                          />
                        }
                        label="Memory Locality Optimization"
                      />
                    </Grid>
                  </Grid>
                  <Box mt={3} display="flex" gap={2}>
                    <Button
                      variant="contained"
                      startIcon={<SaveIcon />}
                      onClick={handleSaveConfig}
                    >
                      Save Configuration
                    </Button>
                    <Button
                      variant="outlined"
                      startIcon={<RestoreIcon />}
                      onClick={handleResetConfig}
                    >
                      Reset to Defaults
                    </Button>
                  </Box>
                </CardContent>
              </Card>

              <Card sx={{ mb: 3 }}>
                <CardContent>
                  <Typography variant="h6" gutterBottom>
                    NUMA Node Configuration
                  </Typography>
                  <TableContainer>
                    <Table>
                      <TableHead>
                        <TableRow>
                          <TableCell>Node ID</TableCell>
                          <TableCell>CPUs</TableCell>
                          <TableCell>Memory</TableCell>
                          <TableCell>Workers</TableCell>
                          <TableCell>CPU Usage</TableCell>
                          <TableCell>Memory Usage</TableCell>
                          <TableCell>Local Access</TableCell>
                          <TableCell>Actions</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {topology.nodes.map((node) => {
                          const nodeMetrics = getNodeMetrics(node.node_id);
                          return (
                            <TableRow key={node.node_id}>
                              <TableCell>
                                <Chip label={`Node ${node.node_id}`} size="small" />
                              </TableCell>
                              <TableCell>
                                <Tooltip title={node.cpu_list.join(', ')}>
                                  <span>{node.cpu_count} CPUs</span>
                                </Tooltip>
                              </TableCell>
                              <TableCell>{node.memory_gb} GB</TableCell>
                              <TableCell>{getWorkerCount(node.node_id)}</TableCell>
                              <TableCell>
                                {nodeMetrics && (
                                  <Box>
                                    <LinearProgress
                                      variant="determinate"
                                      value={nodeMetrics.cpu_usage_percent}
                                      sx={{ mb: 0.5 }}
                                    />
                                    <Typography variant="caption">
                                      {nodeMetrics.cpu_usage_percent.toFixed(1)}%
                                    </Typography>
                                  </Box>
                                )}
                              </TableCell>
                              <TableCell>
                                {nodeMetrics && (
                                  <Box>
                                    <LinearProgress
                                      variant="determinate"
                                      value={nodeMetrics.memory_usage_percent}
                                      sx={{ mb: 0.5 }}
                                    />
                                    <Typography variant="caption">
                                      {nodeMetrics.memory_usage_percent.toFixed(1)}%
                                    </Typography>
                                  </Box>
                                )}
                              </TableCell>
                              <TableCell>
                                {nodeMetrics && (
                                  <Typography variant="caption">
                                    {nodeMetrics.local_memory_access_percent.toFixed(1)}%
                                  </Typography>
                                )}
                              </TableCell>
                              <TableCell>
                                <Button
                                  size="small"
                                  onClick={() => handleOpenWorkerDialog(node.node_id)}
                                >
                                  Configure
                                </Button>
                              </TableCell>
                            </TableRow>
                          );
                        })}
                      </TableBody>
                    </Table>
                  </TableContainer>
                </CardContent>
              </Card>

              <Card>
                <CardContent>
                  <Typography variant="h6" gutterBottom>
                    Performance Metrics
                  </Typography>
                  <ResponsiveContainer width="100%" height={300}>
                    <BarChart data={metricsChartData}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="node" />
                      <YAxis
                        label={{
                          value: 'Percentage / Mbps',
                          angle: -90,
                          position: 'insideLeft',
                        }}
                      />
                      <RechartsTooltip />
                      <Legend />
                      <Bar dataKey="cpu" name="CPU Usage %" fill="#8884d8" />
                      <Bar dataKey="memory" name="Memory Usage %" fill="#82ca9d" />
                      <Bar
                        dataKey="local_access"
                        name="Local Memory Access %"
                        fill="#ffc658"
                      />
                    </BarChart>
                  </ResponsiveContainer>
                </CardContent>
              </Card>
            </>
          )}
        </Box>

        <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
          <DialogTitle>Configure Workers for Node {editingNode}</DialogTitle>
          <DialogContent>
            <Box pt={2}>
              <TextField
                fullWidth
                label="Worker Count"
                type="number"
                value={workerCount}
                onChange={(e) => setWorkerCount(parseInt(e.target.value) || 0)}
                inputProps={{ min: 0, max: 128 }}
                helperText="Number of worker threads to allocate to this NUMA node"
              />
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setDialogOpen(false)}>Cancel</Button>
            <Button variant="contained" onClick={handleSaveWorkerAllocation}>
              Save
            </Button>
          </DialogActions>
        </Dialog>
      </Container>
    </LicenseGate>
  );
};

export default NUMA;

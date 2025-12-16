/**
 * AutoScaling Page
 *
 * Configure auto-scaling policies for modules based on metrics.
 * Supports CPU, memory, connections, and latency-based scaling.
 */

import React, { useState, useEffect } from 'react';
import {
  Container,
  Typography,
  Box,
  Button,
  Alert,
  Card,
  CardContent,
  Grid,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  MenuItem,
  FormControlLabel,
  Switch,
  LinearProgress,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  TrendingUp as ScaleUpIcon,
  TrendingDown as ScaleDownIcon,
} from '@mui/icons-material';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import {
  getModules,
  getAutoScalingPolicies,
  createAutoScalingPolicy,
  updateAutoScalingPolicy,
  deleteAutoScalingPolicy,
  getModuleInstances,
  triggerScaling,
} from '../../services/modulesApi';
import type { Module, AutoScalingPolicy, ModuleInstance } from '../../services/types';

const METRIC_TYPES = [
  { value: 'cpu', label: 'CPU Usage (%)', suffix: '%' },
  { value: 'memory', label: 'Memory Usage (%)', suffix: '%' },
  { value: 'connections', label: 'Active Connections', suffix: '' },
  { value: 'latency', label: 'Average Latency (ms)', suffix: 'ms' },
];

const AutoScaling: React.FC = () => {
  const [modules, setModules] = useState<Module[]>([]);
  const [policies, setPolicies] = useState<AutoScalingPolicy[]>([]);
  const [instances, setInstances] = useState<Record<number, ModuleInstance[]>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingPolicy, setEditingPolicy] = useState<AutoScalingPolicy | undefined>(undefined);
  const [selectedModule, setSelectedModule] = useState<number | null>(null);

  const [formData, setFormData] = useState<Partial<AutoScalingPolicy>>({
    metric_type: 'cpu',
    scale_up_threshold: 70,
    scale_down_threshold: 30,
    min_instances: 1,
    max_instances: 10,
    cooldown_seconds: 300,
    is_enabled: true,
  });

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 30000); // Refresh every 30s
    return () => clearInterval(interval);
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const [modulesData, policiesData] = await Promise.all([
        getModules(),
        getAllPolicies(),
      ]);
      setModules(modulesData.filter((m) => m.is_enabled));
      setPolicies(policiesData);

      // Load instances for each module
      const instancesMap: Record<number, ModuleInstance[]> = {};
      for (const module of modulesData.filter((m) => m.is_enabled)) {
        try {
          const moduleInstances = await getModuleInstances(module.id);
          instancesMap[module.id] = moduleInstances;
        } catch {
          instancesMap[module.id] = [];
        }
      }
      setInstances(instancesMap);

      setError(null);
    } catch (err: any) {
      setError(err.message || 'Failed to load auto-scaling data');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const getAllPolicies = async (): Promise<AutoScalingPolicy[]> => {
    const modulesData = await getModules();
    const policiesPromises = modulesData.map(async (module) => {
      try {
        return await getAutoScalingPolicies(module.id);
      } catch {
        return [];
      }
    });
    const results = await Promise.all(policiesPromises);
    return results.flat();
  };

  const handleOpenDialog = (moduleId: number, policy?: AutoScalingPolicy) => {
    setSelectedModule(moduleId);
    if (policy) {
      setEditingPolicy(policy);
      setFormData(policy);
    } else {
      setEditingPolicy(undefined);
      setFormData({
        metric_type: 'cpu',
        scale_up_threshold: 70,
        scale_down_threshold: 30,
        min_instances: 1,
        max_instances: 10,
        cooldown_seconds: 300,
        is_enabled: true,
      });
    }
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
    setEditingPolicy(undefined);
    setSelectedModule(null);
  };

  const handleSavePolicy = async () => {
    if (!selectedModule) return;
    try {
      if (editingPolicy?.id) {
        await updateAutoScalingPolicy(selectedModule, editingPolicy.id, formData);
      } else {
        await createAutoScalingPolicy(selectedModule, formData);
      }
      await loadData();
      handleCloseDialog();
    } catch (err: any) {
      setError(err.message || 'Failed to save policy');
    }
  };

  const handleDeletePolicy = async (moduleId: number, policyId: number) => {
    if (!confirm('Are you sure you want to delete this policy?')) return;
    try {
      await deleteAutoScalingPolicy(moduleId, policyId);
      await loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to delete policy');
    }
  };

  const handleManualScale = async (moduleId: number, direction: 'up' | 'down') => {
    try {
      await triggerScaling(moduleId, direction, 1);
      await loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to trigger scaling');
    }
  };

  const getModulePolicies = (moduleId: number) => {
    return policies.filter((p) => p.module_id === moduleId);
  };

  const getMetricLabel = (type: AutoScalingPolicy['metric_type']) => {
    return METRIC_TYPES.find((m) => m.value === type)?.label || type;
  };

  return (
    <Container maxWidth="xl">
      <Box py={4}>
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={4}>
          <Box>
            <Typography variant="h4" fontWeight="bold">
              Auto-Scaling Policies
            </Typography>
            <Typography variant="body2" color="text.secondary" mt={1}>
              Configure automatic scaling based on module metrics
            </Typography>
          </Box>
        </Box>

        {error && (
          <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
            {error}
          </Alert>
        )}

        {loading && <LinearProgress sx={{ mb: 3 }} />}

        <Grid container spacing={3}>
          {modules.map((module) => {
            const modulePolicies = getModulePolicies(module.id);
            const moduleInstances = instances[module.id] || [];
            const runningInstances = moduleInstances.filter((i) => i.status === 'running').length;

            return (
              <Grid item xs={12} key={module.id}>
                <Card>
                  <CardContent>
                    <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
                      <Box>
                        <Typography variant="h6" fontWeight="bold">
                          {module.name}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          {module.type} Module - {runningInstances} instance(s) running
                        </Typography>
                      </Box>
                      <Box display="flex" gap={1}>
                        <Button
                          size="small"
                          startIcon={<ScaleUpIcon />}
                          onClick={() => handleManualScale(module.id, 'up')}
                          variant="outlined"
                          color="success"
                        >
                          Scale Up
                        </Button>
                        <Button
                          size="small"
                          startIcon={<ScaleDownIcon />}
                          onClick={() => handleManualScale(module.id, 'down')}
                          variant="outlined"
                          color="error"
                          disabled={runningInstances <= 1}
                        >
                          Scale Down
                        </Button>
                        <Button
                          size="small"
                          startIcon={<AddIcon />}
                          onClick={() => handleOpenDialog(module.id)}
                          variant="contained"
                        >
                          Add Policy
                        </Button>
                      </Box>
                    </Box>

                    {modulePolicies.length === 0 ? (
                      <Box textAlign="center" py={3}>
                        <Typography variant="body2" color="text.secondary">
                          No auto-scaling policies configured
                        </Typography>
                      </Box>
                    ) : (
                      <TableContainer>
                        <Table size="small">
                          <TableHead>
                            <TableRow>
                              <TableCell>Metric</TableCell>
                              <TableCell>Scale Up</TableCell>
                              <TableCell>Scale Down</TableCell>
                              <TableCell>Instance Range</TableCell>
                              <TableCell>Cooldown</TableCell>
                              <TableCell>Status</TableCell>
                              <TableCell align="right">Actions</TableCell>
                            </TableRow>
                          </TableHead>
                          <TableBody>
                            {modulePolicies.map((policy) => (
                              <TableRow key={policy.id}>
                                <TableCell>{getMetricLabel(policy.metric_type)}</TableCell>
                                <TableCell>
                                  {policy.scale_up_threshold}
                                  {METRIC_TYPES.find((m) => m.value === policy.metric_type)?.suffix}
                                </TableCell>
                                <TableCell>
                                  {policy.scale_down_threshold}
                                  {METRIC_TYPES.find((m) => m.value === policy.metric_type)?.suffix}
                                </TableCell>
                                <TableCell>
                                  {policy.min_instances} - {policy.max_instances}
                                </TableCell>
                                <TableCell>{policy.cooldown_seconds}s</TableCell>
                                <TableCell>
                                  <Chip
                                    label={policy.is_enabled ? 'Enabled' : 'Disabled'}
                                    size="small"
                                    color={policy.is_enabled ? 'success' : 'default'}
                                  />
                                </TableCell>
                                <TableCell align="right">
                                  <IconButton
                                    size="small"
                                    onClick={() => handleOpenDialog(module.id, policy)}
                                  >
                                    <EditIcon />
                                  </IconButton>
                                  <IconButton
                                    size="small"
                                    onClick={() => handleDeletePolicy(module.id, policy.id)}
                                  >
                                    <DeleteIcon />
                                  </IconButton>
                                </TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </TableContainer>
                    )}
                  </CardContent>
                </Card>
              </Grid>
            );
          })}
        </Grid>
      </Box>

      {/* Policy Editor Dialog */}
      <Dialog open={dialogOpen} onClose={handleCloseDialog} maxWidth="md" fullWidth>
        <DialogTitle>
          {editingPolicy ? 'Edit Auto-Scaling Policy' : 'Create Auto-Scaling Policy'}
        </DialogTitle>
        <DialogContent>
          <Box pt={2}>
            <Grid container spacing={3}>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  select
                  label="Metric Type"
                  value={formData.metric_type || 'cpu'}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      metric_type: e.target.value as AutoScalingPolicy['metric_type'],
                    })
                  }
                >
                  {METRIC_TYPES.map((metric) => (
                    <MenuItem key={metric.value} value={metric.value}>
                      {metric.label}
                    </MenuItem>
                  ))}
                </TextField>
              </Grid>

              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Scale Up Threshold"
                  type="number"
                  value={formData.scale_up_threshold || 70}
                  onChange={(e) =>
                    setFormData({ ...formData, scale_up_threshold: parseInt(e.target.value) })
                  }
                  helperText="Trigger scale up when metric exceeds this value"
                />
              </Grid>

              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Scale Down Threshold"
                  type="number"
                  value={formData.scale_down_threshold || 30}
                  onChange={(e) =>
                    setFormData({ ...formData, scale_down_threshold: parseInt(e.target.value) })
                  }
                  helperText="Trigger scale down when metric falls below this value"
                />
              </Grid>

              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Minimum Instances"
                  type="number"
                  value={formData.min_instances || 1}
                  onChange={(e) =>
                    setFormData({ ...formData, min_instances: parseInt(e.target.value) })
                  }
                  inputProps={{ min: 1 }}
                />
              </Grid>

              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Maximum Instances"
                  type="number"
                  value={formData.max_instances || 10}
                  onChange={(e) =>
                    setFormData({ ...formData, max_instances: parseInt(e.target.value) })
                  }
                  inputProps={{ min: 1 }}
                />
              </Grid>

              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Cooldown Period (seconds)"
                  type="number"
                  value={formData.cooldown_seconds || 300}
                  onChange={(e) =>
                    setFormData({ ...formData, cooldown_seconds: parseInt(e.target.value) })
                  }
                  helperText="Wait time between scaling actions"
                  inputProps={{ min: 60 }}
                />
              </Grid>

              <Grid item xs={12}>
                <FormControlLabel
                  control={
                    <Switch
                      checked={formData.is_enabled ?? true}
                      onChange={(e) =>
                        setFormData({ ...formData, is_enabled: e.target.checked })
                      }
                    />
                  }
                  label="Enable Policy"
                />
              </Grid>
            </Grid>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button variant="contained" onClick={handleSavePolicy}>
            Save Policy
          </Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
};

export default AutoScaling;

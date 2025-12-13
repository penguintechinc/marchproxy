/**
 * Traffic Shaping & QoS Configuration Page
 *
 * Enterprise feature for configuring Quality of Service policies:
 * - Priority levels (P0-P3)
 * - Bandwidth allocation
 * - Token bucket configuration
 * - DSCP marking
 * - Per-service rate limits
 * - Visual bandwidth allocation charts
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
  Slider,
  FormControlLabel,
  Switch,
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
} from '@mui/material';
import { PieChart, Pie, Cell, ResponsiveContainer, Legend, Tooltip } from 'recharts';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import SaveIcon from '@mui/icons-material/Save';
import LicenseGate from '../../components/Common/LicenseGate';
import { getQoSPolicies, createQoSPolicy, updateQoSPolicy, deleteQoSPolicy } from '../../services/enterpriseApi';
import type { QoSPolicy, BandwidthAllocation } from '../../services/types';
import { useLicense } from '../../hooks/useLicense';

const PRIORITY_COLORS: Record<string, string> = {
  P0: '#f44336',
  P1: '#ff9800',
  P2: '#2196f3',
  P3: '#4caf50',
};

const DSCP_VALUES = [
  { value: 0, label: 'Best Effort (0)' },
  { value: 8, label: 'CS1 (8)' },
  { value: 16, label: 'CS2 (16)' },
  { value: 24, label: 'CS3 (24)' },
  { value: 32, label: 'CS4 (32)' },
  { value: 40, label: 'CS5 (40)' },
  { value: 46, label: 'EF (46)' },
  { value: 48, label: 'CS6 (48)' },
];

const TrafficShaping: React.FC = () => {
  const { isEnterprise, hasFeature, loading: licenseLoading } = useLicense();
  const hasEnterpriseAccess = isEnterprise || hasFeature('traffic_shaping');
  const [policies, setPolicies] = useState<QoSPolicy[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingPolicy, setEditingPolicy] = useState<Partial<QoSPolicy> | null>(null);

  // Form state
  const [formData, setFormData] = useState<Partial<QoSPolicy>>({
    priority: 'P2',
    bandwidth_limit_mbps: 100,
    burst_size_kb: 1024,
    rate_limit_pps: 10000,
    dscp_marking: 0,
    token_bucket_enabled: false,
    token_bucket_rate: 1000,
    token_bucket_burst: 5000,
  });

  useEffect(() => {
    loadPolicies();
  }, []);

  const loadPolicies = async () => {
    try {
      setLoading(true);
      const data = await getQoSPolicies();
      setPolicies(data);
      setError(null);
    } catch (err) {
      setError('Failed to load QoS policies');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleOpenDialog = (policy?: QoSPolicy) => {
    if (policy) {
      setEditingPolicy(policy);
      setFormData(policy);
    } else {
      setEditingPolicy(null);
      setFormData({
        priority: 'P2',
        bandwidth_limit_mbps: 100,
        burst_size_kb: 1024,
        rate_limit_pps: 10000,
        dscp_marking: 0,
        token_bucket_enabled: false,
        token_bucket_rate: 1000,
        token_bucket_burst: 5000,
      });
    }
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
    setEditingPolicy(null);
  };

  const handleSavePolicy = async () => {
    try {
      if (editingPolicy && editingPolicy.id) {
        await updateQoSPolicy(editingPolicy.id, formData);
      } else {
        await createQoSPolicy(formData);
      }
      await loadPolicies();
      handleCloseDialog();
    } catch (err) {
      setError('Failed to save QoS policy');
      console.error(err);
    }
  };

  const handleDeletePolicy = async (id: number) => {
    if (!confirm('Are you sure you want to delete this QoS policy?')) return;
    try {
      await deleteQoSPolicy(id);
      await loadPolicies();
    } catch (err) {
      setError('Failed to delete QoS policy');
      console.error(err);
    }
  };

  const calculateBandwidthAllocation = (): BandwidthAllocation[] => {
    const byPriority: Record<string, number> = {
      P0: 0,
      P1: 0,
      P2: 0,
      P3: 0,
    };

    policies.forEach((policy) => {
      byPriority[policy.priority] += policy.bandwidth_limit_mbps || 0;
    });

    const total = Object.values(byPriority).reduce((sum, val) => sum + val, 0) || 1;

    return Object.entries(byPriority)
      .map(([priority, allocated_mbps]) => ({
        priority,
        allocated_mbps,
        percentage: (allocated_mbps / total) * 100,
        color: PRIORITY_COLORS[priority],
      }))
      .filter((item) => item.allocated_mbps > 0);
  };

  const bandwidthData = calculateBandwidthAllocation();

  return (
    <LicenseGate
      featureName="Traffic Shaping & QoS"
      hasAccess={hasEnterpriseAccess}
      isLoading={licenseLoading}
    >
      <Container maxWidth="xl">
        <Box py={4}>
          <Box display="flex" justifyContent="space-between" alignItems="center" mb={4}>
            <Typography variant="h4" fontWeight="bold">
              Traffic Shaping & QoS
            </Typography>
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={() => handleOpenDialog()}
            >
              Add QoS Policy
            </Button>
          </Box>

          {error && (
            <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
              {error}
            </Alert>
          )}

          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <Card>
                <CardContent>
                  <Typography variant="h6" gutterBottom>
                    Bandwidth Allocation
                  </Typography>
                  {bandwidthData.length > 0 ? (
                    <ResponsiveContainer width="100%" height={300}>
                      <PieChart>
                        <Pie
                          data={bandwidthData as any}
                          dataKey="allocated_mbps"
                          nameKey="priority"
                          cx="50%"
                          cy="50%"
                          outerRadius={100}
                          label
                        >
                          {bandwidthData.map((entry, index) => (
                            <Cell key={`cell-${index}`} fill={entry.color} />
                          ))}
                        </Pie>
                        <Tooltip />
                        <Legend />
                      </PieChart>
                    </ResponsiveContainer>
                  ) : (
                    <Box textAlign="center" py={4}>
                      <Typography variant="body2" color="text.secondary">
                        No bandwidth allocation data available
                      </Typography>
                    </Box>
                  )}
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12} md={6}>
              <Card>
                <CardContent>
                  <Typography variant="h6" gutterBottom>
                    Priority Levels
                  </Typography>
                  <Box>
                    {Object.entries(PRIORITY_COLORS).map(([priority, color]) => (
                      <Box
                        key={priority}
                        display="flex"
                        alignItems="center"
                        justifyContent="space-between"
                        py={1}
                      >
                        <Box display="flex" alignItems="center" gap={1}>
                          <Box
                            width={16}
                            height={16}
                            bgcolor={color}
                            borderRadius="50%"
                          />
                          <Typography variant="body2">{priority}</Typography>
                        </Box>
                        <Typography variant="body2" color="text.secondary">
                          {priority === 'P0' && 'Critical - Highest priority'}
                          {priority === 'P1' && 'High - Business critical'}
                          {priority === 'P2' && 'Medium - Normal traffic'}
                          {priority === 'P3' && 'Low - Best effort'}
                        </Typography>
                      </Box>
                    ))}
                  </Box>
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12}>
              <Card>
                <CardContent>
                  <Typography variant="h6" gutterBottom>
                    QoS Policies
                  </Typography>
                  <TableContainer>
                    <Table>
                      <TableHead>
                        <TableRow>
                          <TableCell>Service ID</TableCell>
                          <TableCell>Priority</TableCell>
                          <TableCell>Bandwidth Limit</TableCell>
                          <TableCell>Rate Limit</TableCell>
                          <TableCell>DSCP</TableCell>
                          <TableCell>Token Bucket</TableCell>
                          <TableCell align="right">Actions</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {policies.map((policy) => (
                          <TableRow key={policy.id}>
                            <TableCell>{policy.service_id}</TableCell>
                            <TableCell>
                              <Chip
                                label={policy.priority}
                                size="small"
                                sx={{
                                  bgcolor: PRIORITY_COLORS[policy.priority],
                                  color: 'white',
                                }}
                              />
                            </TableCell>
                            <TableCell>
                              {policy.bandwidth_limit_mbps
                                ? `${policy.bandwidth_limit_mbps} Mbps`
                                : 'Unlimited'}
                            </TableCell>
                            <TableCell>
                              {policy.rate_limit_pps
                                ? `${policy.rate_limit_pps} pps`
                                : 'Unlimited'}
                            </TableCell>
                            <TableCell>{policy.dscp_marking || 0}</TableCell>
                            <TableCell>
                              {policy.token_bucket_enabled ? 'Enabled' : 'Disabled'}
                            </TableCell>
                            <TableCell align="right">
                              <IconButton
                                size="small"
                                onClick={() => handleOpenDialog(policy)}
                              >
                                <EditIcon />
                              </IconButton>
                              <IconButton
                                size="small"
                                onClick={() => handleDeletePolicy(policy.id)}
                              >
                                <DeleteIcon />
                              </IconButton>
                            </TableCell>
                          </TableRow>
                        ))}
                        {policies.length === 0 && (
                          <TableRow>
                            <TableCell colSpan={7} align="center">
                              <Typography variant="body2" color="text.secondary" py={2}>
                                No QoS policies configured
                              </Typography>
                            </TableCell>
                          </TableRow>
                        )}
                      </TableBody>
                    </Table>
                  </TableContainer>
                </CardContent>
              </Card>
            </Grid>
          </Grid>
        </Box>

        <Dialog open={dialogOpen} onClose={handleCloseDialog} maxWidth="md" fullWidth>
          <DialogTitle>
            {editingPolicy ? 'Edit QoS Policy' : 'Create QoS Policy'}
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
                    label="Priority"
                    value={formData.priority || 'P2'}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        priority: e.target.value as 'P0' | 'P1' | 'P2' | 'P3',
                      })
                    }
                  >
                    <MenuItem value="P0">P0 - Critical</MenuItem>
                    <MenuItem value="P1">P1 - High</MenuItem>
                    <MenuItem value="P2">P2 - Medium</MenuItem>
                    <MenuItem value="P3">P3 - Low</MenuItem>
                  </TextField>
                </Grid>

                <Grid item xs={12}>
                  <Typography variant="body2" gutterBottom>
                    Bandwidth Limit (Mbps): {formData.bandwidth_limit_mbps || 0}
                  </Typography>
                  <Slider
                    value={formData.bandwidth_limit_mbps || 0}
                    onChange={(_, value) =>
                      setFormData({ ...formData, bandwidth_limit_mbps: value as number })
                    }
                    min={0}
                    max={10000}
                    step={10}
                    marks={[
                      { value: 0, label: '0' },
                      { value: 1000, label: '1 Gbps' },
                      { value: 10000, label: '10 Gbps' },
                    ]}
                  />
                </Grid>

                <Grid item xs={12} sm={6}>
                  <TextField
                    fullWidth
                    label="Burst Size (KB)"
                    type="number"
                    value={formData.burst_size_kb || ''}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        burst_size_kb: parseInt(e.target.value),
                      })
                    }
                  />
                </Grid>

                <Grid item xs={12} sm={6}>
                  <TextField
                    fullWidth
                    label="Rate Limit (pps)"
                    type="number"
                    value={formData.rate_limit_pps || ''}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        rate_limit_pps: parseInt(e.target.value),
                      })
                    }
                  />
                </Grid>

                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    select
                    label="DSCP Marking"
                    value={formData.dscp_marking || 0}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        dscp_marking: parseInt(e.target.value),
                      })
                    }
                  >
                    {DSCP_VALUES.map((option) => (
                      <MenuItem key={option.value} value={option.value}>
                        {option.label}
                      </MenuItem>
                    ))}
                  </TextField>
                </Grid>

                <Grid item xs={12}>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={formData.token_bucket_enabled || false}
                        onChange={(e) =>
                          setFormData({
                            ...formData,
                            token_bucket_enabled: e.target.checked,
                          })
                        }
                      />
                    }
                    label="Enable Token Bucket"
                  />
                </Grid>

                {formData.token_bucket_enabled && (
                  <>
                    <Grid item xs={12} sm={6}>
                      <TextField
                        fullWidth
                        label="Token Bucket Rate"
                        type="number"
                        value={formData.token_bucket_rate || ''}
                        onChange={(e) =>
                          setFormData({
                            ...formData,
                            token_bucket_rate: parseInt(e.target.value),
                          })
                        }
                      />
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <TextField
                        fullWidth
                        label="Token Bucket Burst"
                        type="number"
                        value={formData.token_bucket_burst || ''}
                        onChange={(e) =>
                          setFormData({
                            ...formData,
                            token_bucket_burst: parseInt(e.target.value),
                          })
                        }
                      />
                    </Grid>
                  </>
                )}
              </Grid>
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={handleCloseDialog}>Cancel</Button>
            <Button
              variant="contained"
              startIcon={<SaveIcon />}
              onClick={handleSavePolicy}
            >
              Save
            </Button>
          </DialogActions>
        </Dialog>
      </Container>
    </LicenseGate>
  );
};

export default TrafficShaping;

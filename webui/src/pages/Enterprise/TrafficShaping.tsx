/**
 * Traffic Shaping Page
 *
 * Enterprise feature for QoS configuration and bandwidth management.
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
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Switch,
  FormControlLabel,
  Grid,
  Alert
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  PlayArrow as EnableIcon,
  Pause as DisableIcon,
  Speed as SpeedIcon
} from '@mui/icons-material';

import LicenseGate from '../../components/Common/LicenseGate';
import {
  useTrafficShaping,
  QoSPolicyCreate,
  QoSPolicyUpdate,
  PriorityQueueConfig,
  BandwidthLimit
} from '../../hooks/useTrafficShaping';

const TrafficShaping: React.FC = () => {
  const {
    policies,
    loading,
    error,
    hasAccess,
    fetchPolicies,
    createPolicy,
    updatePolicy,
    deletePolicy,
    enablePolicy,
    disablePolicy
  } = useTrafficShaping();

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editMode, setEditMode] = useState(false);
  const [currentPolicyId, setCurrentPolicyId] = useState<number | null>(null);
  const [formData, setFormData] = useState<QoSPolicyCreate>({
    name: '',
    description: '',
    service_id: 0,
    cluster_id: 0,
    bandwidth: {
      ingress_mbps: 1000,
      egress_mbps: 1000,
      burst_size_kb: 1024
    },
    priority_config: {
      priority: 'P2',
      weight: 1,
      max_latency_ms: 100,
      dscp_marking: 'BE'
    },
    enabled: true
  });

  useEffect(() => {
    fetchPolicies();
  }, [fetchPolicies]);

  const handleOpenDialog = (policy?: any) => {
    if (policy) {
      setEditMode(true);
      setCurrentPolicyId(policy.id);
      setFormData({
        name: policy.name,
        description: policy.description || '',
        service_id: policy.service_id,
        cluster_id: policy.cluster_id,
        bandwidth: policy.bandwidth,
        priority_config: policy.priority_config,
        enabled: policy.enabled
      });
    } else {
      setEditMode(false);
      setCurrentPolicyId(null);
      setFormData({
        name: '',
        description: '',
        service_id: 0,
        cluster_id: 0,
        bandwidth: {
          ingress_mbps: 1000,
          egress_mbps: 1000,
          burst_size_kb: 1024
        },
        priority_config: {
          priority: 'P2',
          weight: 1,
          max_latency_ms: 100,
          dscp_marking: 'BE'
        },
        enabled: true
      });
    }
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
  };

  const handleSave = async () => {
    if (editMode && currentPolicyId) {
      await updatePolicy(currentPolicyId, formData);
    } else {
      await createPolicy(formData);
    }
    handleCloseDialog();
  };

  const handleDelete = async (id: number) => {
    if (window.confirm('Are you sure you want to delete this QoS policy?')) {
      await deletePolicy(id);
    }
  };

  const handleToggle = async (id: number, enabled: boolean) => {
    if (enabled) {
      await disablePolicy(id);
    } else {
      await enablePolicy(id);
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'P0': return 'error';
      case 'P1': return 'warning';
      case 'P2': return 'info';
      case 'P3': return 'default';
      default: return 'default';
    }
  };

  return (
    <LicenseGate
      featureName="Advanced Traffic Shaping & QoS"
      hasAccess={hasAccess}
      isLoading={loading && policies.length === 0}
    >
      <Container maxWidth="xl" sx={{ mt: 4, mb: 4 }}>
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
          <Box>
            <Typography variant="h4" gutterBottom>
              <SpeedIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
              Traffic Shaping & QoS
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Configure bandwidth limits, priority queues, and DSCP marking
            </Typography>
          </Box>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => handleOpenDialog()}
          >
            Create Policy
          </Button>
        </Box>

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Service ID</TableCell>
                <TableCell>Priority</TableCell>
                <TableCell>Ingress Limit</TableCell>
                <TableCell>Egress Limit</TableCell>
                <TableCell>DSCP</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {policies.map((policy) => (
                <TableRow key={policy.id}>
                  <TableCell>
                    <Typography variant="body2" fontWeight="bold">
                      {policy.name}
                    </Typography>
                    {policy.description && (
                      <Typography variant="caption" color="text.secondary">
                        {policy.description}
                      </Typography>
                    )}
                  </TableCell>
                  <TableCell>{policy.service_id}</TableCell>
                  <TableCell>
                    <Chip
                      label={policy.priority_config.priority}
                      color={getPriorityColor(policy.priority_config.priority)}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    {policy.bandwidth.ingress_mbps
                      ? `${policy.bandwidth.ingress_mbps} Mbps`
                      : 'Unlimited'}
                  </TableCell>
                  <TableCell>
                    {policy.bandwidth.egress_mbps
                      ? `${policy.bandwidth.egress_mbps} Mbps`
                      : 'Unlimited'}
                  </TableCell>
                  <TableCell>{policy.priority_config.dscp_marking}</TableCell>
                  <TableCell>
                    <Chip
                      label={policy.enabled ? 'Enabled' : 'Disabled'}
                      color={policy.enabled ? 'success' : 'default'}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    <IconButton
                      size="small"
                      onClick={() => handleToggle(policy.id, policy.enabled)}
                      title={policy.enabled ? 'Disable' : 'Enable'}
                    >
                      {policy.enabled ? <DisableIcon /> : <EnableIcon />}
                    </IconButton>
                    <IconButton
                      size="small"
                      onClick={() => handleOpenDialog(policy)}
                      title="Edit"
                    >
                      <EditIcon />
                    </IconButton>
                    <IconButton
                      size="small"
                      onClick={() => handleDelete(policy.id)}
                      title="Delete"
                    >
                      <DeleteIcon />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
              {policies.length === 0 && !loading && (
                <TableRow>
                  <TableCell colSpan={8} align="center">
                    <Typography variant="body2" color="text.secondary" py={4}>
                      No QoS policies configured. Click "Create Policy" to get started.
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>

        {/* Create/Edit Dialog - Simplified for size */}
        <Dialog open={dialogOpen} onClose={handleCloseDialog} maxWidth="md" fullWidth>
          <DialogTitle>
            {editMode ? 'Edit QoS Policy' : 'Create QoS Policy'}
          </DialogTitle>
          <DialogContent>
            <Box sx={{ pt: 2 }}>
              <Grid container spacing={2}>
                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    label="Policy Name"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  />
                </Grid>
                <Grid item xs={12} sm={6}>
                  <TextField
                    fullWidth
                    type="number"
                    label="Service ID"
                    value={formData.service_id}
                    onChange={(e) => setFormData({
                      ...formData,
                      service_id: parseInt(e.target.value)
                    })}
                  />
                </Grid>
                <Grid item xs={12} sm={6}>
                  <TextField
                    fullWidth
                    type="number"
                    label="Cluster ID"
                    value={formData.cluster_id}
                    onChange={(e) => setFormData({
                      ...formData,
                      cluster_id: parseInt(e.target.value)
                    })}
                  />
                </Grid>
                <Grid item xs={12} sm={6}>
                  <FormControl fullWidth>
                    <InputLabel>Priority</InputLabel>
                    <Select
                      value={formData.priority_config.priority}
                      label="Priority"
                      onChange={(e) => setFormData({
                        ...formData,
                        priority_config: {
                          ...formData.priority_config,
                          priority: e.target.value as any
                        }
                      })}
                    >
                      <MenuItem value="P0">P0 - Interactive (&lt;1ms)</MenuItem>
                      <MenuItem value="P1">P1 - Real-time (&lt;10ms)</MenuItem>
                      <MenuItem value="P2">P2 - Bulk (&lt;100ms)</MenuItem>
                      <MenuItem value="P3">P3 - Best Effort</MenuItem>
                    </Select>
                  </FormControl>
                </Grid>
                <Grid item xs={12} sm={6}>
                  <FormControl fullWidth>
                    <InputLabel>DSCP Marking</InputLabel>
                    <Select
                      value={formData.priority_config.dscp_marking}
                      label="DSCP Marking"
                      onChange={(e) => setFormData({
                        ...formData,
                        priority_config: {
                          ...formData.priority_config,
                          dscp_marking: e.target.value as any
                        }
                      })}
                    >
                      <MenuItem value="EF">EF - Expedited Forwarding</MenuItem>
                      <MenuItem value="AF41">AF41 - Assured Forwarding 4-1</MenuItem>
                      <MenuItem value="AF31">AF31 - Assured Forwarding 3-1</MenuItem>
                      <MenuItem value="AF21">AF21 - Assured Forwarding 2-1</MenuItem>
                      <MenuItem value="AF11">AF11 - Assured Forwarding 1-1</MenuItem>
                      <MenuItem value="BE">BE - Best Effort</MenuItem>
                    </Select>
                  </FormControl>
                </Grid>
                <Grid item xs={12} sm={6}>
                  <TextField
                    fullWidth
                    type="number"
                    label="Ingress Limit (Mbps)"
                    value={formData.bandwidth.ingress_mbps || ''}
                    onChange={(e) => setFormData({
                      ...formData,
                      bandwidth: {
                        ...formData.bandwidth,
                        ingress_mbps: e.target.value ? parseInt(e.target.value) : undefined
                      }
                    })}
                  />
                </Grid>
                <Grid item xs={12} sm={6}>
                  <TextField
                    fullWidth
                    type="number"
                    label="Egress Limit (Mbps)"
                    value={formData.bandwidth.egress_mbps || ''}
                    onChange={(e) => setFormData({
                      ...formData,
                      bandwidth: {
                        ...formData.bandwidth,
                        egress_mbps: e.target.value ? parseInt(e.target.value) : undefined
                      }
                    })}
                  />
                </Grid>
              </Grid>
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={handleCloseDialog}>Cancel</Button>
            <Button onClick={handleSave} variant="contained">
              {editMode ? 'Update' : 'Create'}
            </Button>
          </DialogActions>
        </Dialog>
      </Container>
    </LicenseGate>
  );
};

export default TrafficShaping;

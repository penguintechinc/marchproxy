/**
 * RouteEditor Component
 *
 * Form for creating/editing routes for a module.
 * Supports protocol selection, backend configuration, rate limiting, and priority.
 */

import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Grid,
  MenuItem,
  FormControlLabel,
  Switch,
  Typography,
  Box,
  Divider,
  Alert,
} from '@mui/material';
import SaveIcon from '@mui/icons-material/Save';
import type { ModuleRoute, Module } from '../../services/types';

interface RouteEditorProps {
  open: boolean;
  onClose: () => void;
  onSave: (route: Partial<ModuleRoute>) => Promise<void>;
  route?: ModuleRoute;
  moduleType: Module['type'];
  loading?: boolean;
}

const PROTOCOL_OPTIONS: Record<Module['type'], string[]> = {
  NLB: ['TCP', 'UDP', 'ICMP'],
  ALB: ['HTTP', 'HTTPS', 'WebSocket', 'HTTP/2', 'HTTP/3'],
  DBLB: ['MySQL', 'PostgreSQL', 'MongoDB', 'Redis', 'MSSQL'],
  AILB: ['OpenAI', 'Anthropic', 'Ollama', 'Custom'],
  RTMP: ['RTMP', 'RTMPS', 'HLS', 'DASH'],
};

const PRIORITY_OPTIONS: Array<{ value: ModuleRoute['priority']; label: string }> = [
  { value: 'P0', label: 'P0 - Critical' },
  { value: 'P1', label: 'P1 - High' },
  { value: 'P2', label: 'P2 - Normal' },
  { value: 'P3', label: 'P3 - Low' },
];

const RouteEditor: React.FC<RouteEditorProps> = ({
  open,
  onClose,
  onSave,
  route,
  moduleType,
  loading = false,
}) => {
  const [formData, setFormData] = useState<Partial<ModuleRoute>>({
    name: '',
    protocol: PROTOCOL_OPTIONS[moduleType]?.[0] || 'TCP',
    backend_url: '',
    backend_port: 80,
    is_active: true,
    rate_limit_rps: undefined,
    rate_limit_connections: undefined,
    rate_limit_bandwidth_mbps: undefined,
    priority: 'P2',
  });

  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (route) {
      setFormData(route);
    } else {
      setFormData({
        name: '',
        protocol: PROTOCOL_OPTIONS[moduleType]?.[0] || 'TCP',
        backend_url: '',
        backend_port: 80,
        is_active: true,
        priority: 'P2',
      });
    }
    setError(null);
  }, [route, moduleType, open]);

  const handleChange = (field: keyof ModuleRoute, value: any) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
  };

  const handleSave = async () => {
    // Validation
    if (!formData.name || !formData.backend_url) {
      setError('Name and backend URL are required');
      return;
    }

    if (formData.backend_port && (formData.backend_port < 1 || formData.backend_port > 65535)) {
      setError('Backend port must be between 1 and 65535');
      return;
    }

    try {
      setError(null);
      await onSave(formData);
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to save route');
    }
  };

  const protocols = PROTOCOL_OPTIONS[moduleType] || ['TCP'];

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>
        {route ? 'Edit Route' : 'Create New Route'} - {moduleType} Module
      </DialogTitle>

      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
            {error}
          </Alert>
        )}

        <Box pt={1}>
          <Grid container spacing={3}>
            {/* Basic Configuration */}
            <Grid item xs={12}>
              <Typography variant="subtitle2" gutterBottom fontWeight="bold">
                Basic Configuration
              </Typography>
              <Divider />
            </Grid>

            <Grid item xs={12} sm={6}>
              <TextField
                fullWidth
                label="Route Name"
                value={formData.name || ''}
                onChange={(e) => handleChange('name', e.target.value)}
                required
                helperText="Unique name for this route"
              />
            </Grid>

            <Grid item xs={12} sm={6}>
              <TextField
                fullWidth
                select
                label="Protocol"
                value={formData.protocol || protocols[0]}
                onChange={(e) => handleChange('protocol', e.target.value)}
              >
                {protocols.map((protocol) => (
                  <MenuItem key={protocol} value={protocol}>
                    {protocol}
                  </MenuItem>
                ))}
              </TextField>
            </Grid>

            <Grid item xs={12} sm={8}>
              <TextField
                fullWidth
                label="Backend URL"
                value={formData.backend_url || ''}
                onChange={(e) => handleChange('backend_url', e.target.value)}
                required
                helperText="FQDN or IP address of the backend server"
              />
            </Grid>

            <Grid item xs={12} sm={4}>
              <TextField
                fullWidth
                label="Backend Port"
                type="number"
                value={formData.backend_port || 80}
                onChange={(e) => handleChange('backend_port', parseInt(e.target.value))}
                inputProps={{ min: 1, max: 65535 }}
              />
            </Grid>

            {/* Rate Limiting */}
            <Grid item xs={12}>
              <Typography variant="subtitle2" gutterBottom fontWeight="bold" sx={{ mt: 2 }}>
                Rate Limiting (Optional)
              </Typography>
              <Divider />
            </Grid>

            <Grid item xs={12} sm={4}>
              <TextField
                fullWidth
                label="Requests/sec"
                type="number"
                value={formData.rate_limit_rps || ''}
                onChange={(e) =>
                  handleChange('rate_limit_rps', e.target.value ? parseInt(e.target.value) : undefined)
                }
                inputProps={{ min: 1 }}
                helperText="Max requests per second"
              />
            </Grid>

            <Grid item xs={12} sm={4}>
              <TextField
                fullWidth
                label="Max Connections"
                type="number"
                value={formData.rate_limit_connections || ''}
                onChange={(e) =>
                  handleChange('rate_limit_connections', e.target.value ? parseInt(e.target.value) : undefined)
                }
                inputProps={{ min: 1 }}
                helperText="Max concurrent connections"
              />
            </Grid>

            <Grid item xs={12} sm={4}>
              <TextField
                fullWidth
                label="Bandwidth (Mbps)"
                type="number"
                value={formData.rate_limit_bandwidth_mbps || ''}
                onChange={(e) =>
                  handleChange('rate_limit_bandwidth_mbps', e.target.value ? parseInt(e.target.value) : undefined)
                }
                inputProps={{ min: 1 }}
                helperText="Max bandwidth in Mbps"
              />
            </Grid>

            {/* Priority & Status */}
            <Grid item xs={12}>
              <Typography variant="subtitle2" gutterBottom fontWeight="bold" sx={{ mt: 2 }}>
                Priority & Status
              </Typography>
              <Divider />
            </Grid>

            <Grid item xs={12} sm={6}>
              <TextField
                fullWidth
                select
                label="Priority"
                value={formData.priority || 'P2'}
                onChange={(e) => handleChange('priority', e.target.value as ModuleRoute['priority'])}
              >
                {PRIORITY_OPTIONS.map((option) => (
                  <MenuItem key={option.value} value={option.value}>
                    {option.label}
                  </MenuItem>
                ))}
              </TextField>
            </Grid>

            <Grid item xs={12} sm={6}>
              <Box pt={1}>
                <FormControlLabel
                  control={
                    <Switch
                      checked={formData.is_active ?? true}
                      onChange={(e) => handleChange('is_active', e.target.checked)}
                    />
                  }
                  label="Route Active"
                />
              </Box>
            </Grid>
          </Grid>
        </Box>
      </DialogContent>

      <DialogActions>
        <Button onClick={onClose} disabled={loading}>
          Cancel
        </Button>
        <Button
          variant="contained"
          startIcon={<SaveIcon />}
          onClick={handleSave}
          disabled={loading}
        >
          {loading ? 'Saving...' : 'Save Route'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default RouteEditor;

/**
 * Kong Plugins Management Page with JSON Config Editor
 */
import React, { useEffect, useState, useCallback } from 'react';
import {
  Box,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Alert,
  IconButton,
  Tooltip,
  MenuItem,
  FormControlLabel,
  Switch,
  Typography,
  Chip,
} from '@mui/material';
import { DataGrid, GridColDef, GridActionsCellItem } from '@mui/x-data-grid';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import RefreshIcon from '@mui/icons-material/Refresh';
import { kongApi, KongPlugin, KongService, KongRoute } from '../../services/kongApi';

const KongPlugins: React.FC = () => {
  const [plugins, setPlugins] = useState<KongPlugin[]>([]);
  const [enabledPlugins, setEnabledPlugins] = useState<string[]>([]);
  const [services, setServices] = useState<KongService[]>([]);
  const [routes, setRoutes] = useState<KongRoute[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingPlugin, setEditingPlugin] = useState<KongPlugin | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    enabled: true,
    service_id: '',
    route_id: '',
    config: '{}',
  });
  const [configError, setConfigError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [pluginsRes, enabledRes, servicesRes, routesRes] = await Promise.all([
        kongApi.getPlugins(),
        kongApi.getEnabledPlugins(),
        kongApi.getServices(),
        kongApi.getRoutes(),
      ]);
      setPlugins(pluginsRes.data.data || []);
      setEnabledPlugins(enabledRes.data.enabled_plugins || []);
      setServices(servicesRes.data.data || []);
      setRoutes(routesRes.data.data || []);
      setError(null);
    } catch (err) {
      setError('Failed to load plugins');
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleOpenDialog = (plugin?: KongPlugin) => {
    if (plugin) {
      setEditingPlugin(plugin);
      setFormData({
        name: plugin.name,
        enabled: plugin.enabled ?? true,
        service_id: plugin.service?.id || '',
        route_id: plugin.route?.id || '',
        config: JSON.stringify(plugin.config || {}, null, 2),
      });
    } else {
      setEditingPlugin(null);
      setFormData({
        name: '',
        enabled: true,
        service_id: '',
        route_id: '',
        config: '{}',
      });
    }
    setConfigError(null);
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
    setEditingPlugin(null);
    setConfigError(null);
  };

  const validateConfig = (configStr: string): boolean => {
    try {
      JSON.parse(configStr);
      setConfigError(null);
      return true;
    } catch (e) {
      setConfigError('Invalid JSON configuration');
      return false;
    }
  };

  const handleSubmit = async () => {
    if (!validateConfig(formData.config)) return;

    try {
      const payload: Partial<KongPlugin> = {
        name: formData.name,
        enabled: formData.enabled,
        config: JSON.parse(formData.config),
      };

      if (formData.service_id) {
        payload.service = { id: formData.service_id };
      }
      if (formData.route_id) {
        payload.route = { id: formData.route_id };
      }

      if (editingPlugin) {
        await kongApi.updatePlugin(editingPlugin.id, payload);
      } else {
        await kongApi.createPlugin(payload);
      }
      handleCloseDialog();
      fetchData();
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to save plugin';
      setError(errorMessage);
    }
  };

  const handleDelete = async (id: string) => {
    if (window.confirm('Are you sure you want to delete this plugin?')) {
      try {
        await kongApi.deletePlugin(id);
        fetchData();
      } catch (err) {
        setError('Failed to delete plugin');
      }
    }
  };

  const getScope = (plugin: KongPlugin): string => {
    if (plugin.service?.id && plugin.route?.id) return 'Route + Service';
    if (plugin.service?.id) return 'Service';
    if (plugin.route?.id) return 'Route';
    if (plugin.consumer?.id) return 'Consumer';
    return 'Global';
  };

  const columns: GridColDef[] = [
    { field: 'name', headerName: 'Plugin', flex: 1, minWidth: 150 },
    {
      field: 'enabled',
      headerName: 'Enabled',
      width: 100,
      renderCell: (params) => (
        <Chip
          label={params.value ? 'Yes' : 'No'}
          color={params.value ? 'success' : 'default'}
          size="small"
        />
      ),
    },
    {
      field: 'scope',
      headerName: 'Scope',
      width: 150,
      valueGetter: (_, row) => getScope(row),
    },
    {
      field: 'service',
      headerName: 'Service',
      width: 150,
      valueGetter: (value) => {
        const svc = services.find((s) => s.id === value?.id);
        return svc?.name || '-';
      },
    },
    {
      field: 'route',
      headerName: 'Route',
      width: 150,
      valueGetter: (value) => {
        const rt = routes.find((r) => r.id === value?.id);
        return rt?.name || '-';
      },
    },
    {
      field: 'actions',
      type: 'actions',
      headerName: 'Actions',
      width: 100,
      getActions: (params) => [
        <GridActionsCellItem
          key="edit"
          icon={<EditIcon />}
          label="Edit"
          onClick={() => handleOpenDialog(params.row)}
        />,
        <GridActionsCellItem
          key="delete"
          icon={<DeleteIcon />}
          label="Delete"
          onClick={() => handleDelete(params.row.id)}
        />,
      ],
    },
  ];

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Box>
          <Button variant="contained" startIcon={<AddIcon />} onClick={() => handleOpenDialog()}>
            Add Plugin
          </Button>
          <Tooltip title="Refresh">
            <IconButton onClick={fetchData} sx={{ ml: 1 }}>
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </Box>
        <Typography variant="body2" color="textSecondary">
          {enabledPlugins.length} plugins available
        </Typography>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      <DataGrid
        rows={plugins}
        columns={columns}
        loading={loading}
        autoHeight
        pageSizeOptions={[10, 25, 50]}
        initialState={{ pagination: { paginationModel: { pageSize: 10 } } }}
        disableRowSelectionOnClick
      />

      <Dialog open={dialogOpen} onClose={handleCloseDialog} maxWidth="md" fullWidth>
        <DialogTitle>{editingPlugin ? 'Edit Plugin' : 'Add Plugin'}</DialogTitle>
        <DialogContent>
          <TextField
            select
            margin="dense"
            label="Plugin Name"
            fullWidth
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            disabled={!!editingPlugin}
          >
            {enabledPlugins.map((p) => (
              <MenuItem key={p} value={p}>{p}</MenuItem>
            ))}
          </TextField>

          <TextField
            select
            margin="dense"
            label="Service (Optional)"
            fullWidth
            value={formData.service_id}
            onChange={(e) => setFormData({ ...formData, service_id: e.target.value })}
          >
            <MenuItem value="">Global (No Service)</MenuItem>
            {services.map((s) => (
              <MenuItem key={s.id} value={s.id}>{s.name}</MenuItem>
            ))}
          </TextField>

          <TextField
            select
            margin="dense"
            label="Route (Optional)"
            fullWidth
            value={formData.route_id}
            onChange={(e) => setFormData({ ...formData, route_id: e.target.value })}
          >
            <MenuItem value="">No Route</MenuItem>
            {routes.map((r) => (
              <MenuItem key={r.id} value={r.id}>{r.name}</MenuItem>
            ))}
          </TextField>

          <FormControlLabel
            control={
              <Switch
                checked={formData.enabled}
                onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
              />
            }
            label="Enabled"
            sx={{ mt: 1 }}
          />

          <Typography variant="subtitle2" sx={{ mt: 2, mb: 1 }}>
            Plugin Configuration (JSON)
          </Typography>
          <TextField
            multiline
            rows={12}
            fullWidth
            value={formData.config}
            onChange={(e) => {
              setFormData({ ...formData, config: e.target.value });
              validateConfig(e.target.value);
            }}
            error={!!configError}
            helperText={configError}
            sx={{
              fontFamily: 'monospace',
              '& .MuiInputBase-input': { fontFamily: 'monospace', fontSize: '0.875rem' },
            }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button onClick={handleSubmit} variant="contained" disabled={!!configError}>
            {editingPlugin ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default KongPlugins;

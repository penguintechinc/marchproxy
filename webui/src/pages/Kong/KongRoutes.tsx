/**
 * Kong Routes Management Page
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
  FormControlLabel,
  Switch,
  Chip,
  MenuItem,
  Autocomplete,
} from '@mui/material';
import { DataGrid, GridColDef, GridActionsCellItem } from '@mui/x-data-grid';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import RefreshIcon from '@mui/icons-material/Refresh';
import { kongApi, KongRoute, KongService } from '../../services/kongApi';

const PROTOCOLS = ['http', 'https', 'grpc', 'grpcs', 'tcp', 'tls'];
const METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS', 'HEAD'];

const KongRoutes: React.FC = () => {
  const [routes, setRoutes] = useState<KongRoute[]>([]);
  const [services, setServices] = useState<KongService[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingRoute, setEditingRoute] = useState<KongRoute | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    protocols: ['http', 'https'],
    methods: [] as string[],
    hosts: [] as string[],
    paths: [] as string[],
    strip_path: true,
    preserve_host: false,
    service_id: '',
  });
  const [hostInput, setHostInput] = useState('');
  const [pathInput, setPathInput] = useState('');

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [routesRes, servicesRes] = await Promise.all([
        kongApi.getRoutes(),
        kongApi.getServices(),
      ]);
      setRoutes(routesRes.data.data || []);
      setServices(servicesRes.data.data || []);
      setError(null);
    } catch (err) {
      setError('Failed to load routes');
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleOpenDialog = (route?: KongRoute) => {
    if (route) {
      setEditingRoute(route);
      setFormData({
        name: route.name,
        protocols: route.protocols || ['http', 'https'],
        methods: route.methods || [],
        hosts: route.hosts || [],
        paths: route.paths || [],
        strip_path: route.strip_path ?? true,
        preserve_host: route.preserve_host ?? false,
        service_id: route.service?.id || '',
      });
    } else {
      setEditingRoute(null);
      setFormData({
        name: '',
        protocols: ['http', 'https'],
        methods: [],
        hosts: [],
        paths: [],
        strip_path: true,
        preserve_host: false,
        service_id: '',
      });
    }
    setHostInput('');
    setPathInput('');
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
    setEditingRoute(null);
  };

  const handleSubmit = async () => {
    try {
      const payload: Partial<KongRoute> = {
        name: formData.name,
        protocols: formData.protocols,
        methods: formData.methods.length > 0 ? formData.methods : undefined,
        hosts: formData.hosts.length > 0 ? formData.hosts : undefined,
        paths: formData.paths.length > 0 ? formData.paths : undefined,
        strip_path: formData.strip_path,
        preserve_host: formData.preserve_host,
      };

      if (formData.service_id) {
        payload.service = { id: formData.service_id };
      }

      if (editingRoute) {
        await kongApi.updateRoute(editingRoute.id, payload);
      } else {
        await kongApi.createRoute(payload);
      }
      handleCloseDialog();
      fetchData();
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to save route';
      setError(errorMessage);
    }
  };

  const handleDelete = async (id: string) => {
    if (window.confirm('Are you sure you want to delete this route?')) {
      try {
        await kongApi.deleteRoute(id);
        fetchData();
      } catch (err) {
        setError('Failed to delete route');
      }
    }
  };

  const addHost = () => {
    if (hostInput && !formData.hosts.includes(hostInput)) {
      setFormData({ ...formData, hosts: [...formData.hosts, hostInput] });
      setHostInput('');
    }
  };

  const addPath = () => {
    if (pathInput && !formData.paths.includes(pathInput)) {
      setFormData({ ...formData, paths: [...formData.paths, pathInput] });
      setPathInput('');
    }
  };

  const columns: GridColDef[] = [
    { field: 'name', headerName: 'Name', flex: 1, minWidth: 150 },
    {
      field: 'protocols',
      headerName: 'Protocols',
      width: 150,
      renderCell: (params) => (params.value || []).join(', '),
    },
    {
      field: 'hosts',
      headerName: 'Hosts',
      flex: 1,
      renderCell: (params) => (params.value || []).join(', '),
    },
    {
      field: 'paths',
      headerName: 'Paths',
      flex: 1,
      renderCell: (params) => (params.value || []).join(', '),
    },
    {
      field: 'service',
      headerName: 'Service',
      width: 150,
      valueGetter: (value) => {
        const svc = services.find((s) => s.id === value?.id);
        return svc?.name || value?.id || '-';
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
            Add Route
          </Button>
          <Tooltip title="Refresh">
            <IconButton onClick={fetchData} sx={{ ml: 1 }}>
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </Box>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      <DataGrid
        rows={routes}
        columns={columns}
        loading={loading}
        autoHeight
        pageSizeOptions={[10, 25, 50]}
        initialState={{ pagination: { paginationModel: { pageSize: 10 } } }}
        disableRowSelectionOnClick
      />

      <Dialog open={dialogOpen} onClose={handleCloseDialog} maxWidth="md" fullWidth>
        <DialogTitle>{editingRoute ? 'Edit Route' : 'Add Route'}</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            margin="dense"
            label="Name"
            fullWidth
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            disabled={!!editingRoute}
          />

          <TextField
            select
            margin="dense"
            label="Service"
            fullWidth
            value={formData.service_id}
            onChange={(e) => setFormData({ ...formData, service_id: e.target.value })}
          >
            <MenuItem value="">None</MenuItem>
            {services.map((s) => (
              <MenuItem key={s.id} value={s.id}>
                {s.name}
              </MenuItem>
            ))}
          </TextField>

          <Autocomplete
            multiple
            options={PROTOCOLS}
            value={formData.protocols}
            onChange={(_, newValue) => setFormData({ ...formData, protocols: newValue })}
            renderInput={(params) => <TextField {...params} margin="dense" label="Protocols" />}
          />

          <Autocomplete
            multiple
            options={METHODS}
            value={formData.methods}
            onChange={(_, newValue) => setFormData({ ...formData, methods: newValue })}
            renderInput={(params) => <TextField {...params} margin="dense" label="Methods" />}
          />

          <Box display="flex" gap={1} alignItems="center" mt={1}>
            <TextField
              label="Add Host"
              value={hostInput}
              onChange={(e) => setHostInput(e.target.value)}
              size="small"
              onKeyPress={(e) => e.key === 'Enter' && addHost()}
            />
            <Button onClick={addHost} size="small">Add</Button>
          </Box>
          <Box display="flex" gap={0.5} flexWrap="wrap" mt={1}>
            {formData.hosts.map((host) => (
              <Chip
                key={host}
                label={host}
                onDelete={() => setFormData({ ...formData, hosts: formData.hosts.filter((h) => h !== host) })}
                size="small"
              />
            ))}
          </Box>

          <Box display="flex" gap={1} alignItems="center" mt={2}>
            <TextField
              label="Add Path"
              value={pathInput}
              onChange={(e) => setPathInput(e.target.value)}
              size="small"
              onKeyPress={(e) => e.key === 'Enter' && addPath()}
            />
            <Button onClick={addPath} size="small">Add</Button>
          </Box>
          <Box display="flex" gap={0.5} flexWrap="wrap" mt={1}>
            {formData.paths.map((path) => (
              <Chip
                key={path}
                label={path}
                onDelete={() => setFormData({ ...formData, paths: formData.paths.filter((p) => p !== path) })}
                size="small"
              />
            ))}
          </Box>

          <Box mt={2}>
            <FormControlLabel
              control={
                <Switch
                  checked={formData.strip_path}
                  onChange={(e) => setFormData({ ...formData, strip_path: e.target.checked })}
                />
              }
              label="Strip Path"
            />
            <FormControlLabel
              control={
                <Switch
                  checked={formData.preserve_host}
                  onChange={(e) => setFormData({ ...formData, preserve_host: e.target.checked })}
                />
              }
              label="Preserve Host"
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button onClick={handleSubmit} variant="contained">
            {editingRoute ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default KongRoutes;

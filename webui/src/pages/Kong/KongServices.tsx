/**
 * Kong Services Management Page
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
  MenuItem,
} from '@mui/material';
import { DataGrid, GridColDef, GridActionsCellItem } from '@mui/x-data-grid';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import RefreshIcon from '@mui/icons-material/Refresh';
import { kongApi, KongService } from '../../services/kongApi';

const KongServices: React.FC = () => {
  const [services, setServices] = useState<KongService[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingService, setEditingService] = useState<KongService | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    protocol: 'http',
    host: '',
    port: 80,
    path: '',
    retries: 5,
    connect_timeout: 60000,
    write_timeout: 60000,
    read_timeout: 60000,
    enabled: true,
  });

  const fetchServices = useCallback(async () => {
    setLoading(true);
    try {
      const response = await kongApi.getServices();
      setServices(response.data.data || []);
      setError(null);
    } catch (err) {
      setError('Failed to load services');
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchServices();
  }, [fetchServices]);

  const handleOpenDialog = (service?: KongService) => {
    if (service) {
      setEditingService(service);
      setFormData({
        name: service.name,
        protocol: service.protocol || 'http',
        host: service.host,
        port: service.port || 80,
        path: service.path || '',
        retries: service.retries ?? 5,
        connect_timeout: service.connect_timeout ?? 60000,
        write_timeout: service.write_timeout ?? 60000,
        read_timeout: service.read_timeout ?? 60000,
        enabled: service.enabled ?? true,
      });
    } else {
      setEditingService(null);
      setFormData({
        name: '',
        protocol: 'http',
        host: '',
        port: 80,
        path: '',
        retries: 5,
        connect_timeout: 60000,
        write_timeout: 60000,
        read_timeout: 60000,
        enabled: true,
      });
    }
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
    setEditingService(null);
  };

  const handleSubmit = async () => {
    try {
      const payload = {
        ...formData,
        path: formData.path || undefined,
      };

      if (editingService) {
        await kongApi.updateService(editingService.id, payload);
      } else {
        await kongApi.createService(payload);
      }
      handleCloseDialog();
      fetchServices();
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to save service';
      setError(errorMessage);
    }
  };

  const handleDelete = async (id: string) => {
    if (window.confirm('Are you sure you want to delete this service?')) {
      try {
        await kongApi.deleteService(id);
        fetchServices();
      } catch (err) {
        setError('Failed to delete service');
      }
    }
  };

  const columns: GridColDef[] = [
    { field: 'name', headerName: 'Name', flex: 1, minWidth: 150 },
    { field: 'protocol', headerName: 'Protocol', width: 100 },
    { field: 'host', headerName: 'Host', flex: 1, minWidth: 150 },
    { field: 'port', headerName: 'Port', width: 80, type: 'number' },
    { field: 'path', headerName: 'Path', width: 120 },
    {
      field: 'enabled',
      headerName: 'Enabled',
      width: 100,
      renderCell: (params) => (params.value ? 'Yes' : 'No'),
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
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => handleOpenDialog()}
          >
            Add Service
          </Button>
          <Tooltip title="Refresh">
            <IconButton onClick={fetchServices} sx={{ ml: 1 }}>
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
        rows={services}
        columns={columns}
        loading={loading}
        autoHeight
        pageSizeOptions={[10, 25, 50]}
        initialState={{ pagination: { paginationModel: { pageSize: 10 } } }}
        disableRowSelectionOnClick
      />

      {/* Add/Edit Dialog */}
      <Dialog open={dialogOpen} onClose={handleCloseDialog} maxWidth="sm" fullWidth>
        <DialogTitle>{editingService ? 'Edit Service' : 'Add Service'}</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            margin="dense"
            label="Name"
            fullWidth
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            disabled={!!editingService}
          />
          <TextField
            select
            margin="dense"
            label="Protocol"
            fullWidth
            value={formData.protocol}
            onChange={(e) => setFormData({ ...formData, protocol: e.target.value })}
          >
            <MenuItem value="http">HTTP</MenuItem>
            <MenuItem value="https">HTTPS</MenuItem>
            <MenuItem value="grpc">gRPC</MenuItem>
            <MenuItem value="grpcs">gRPCs</MenuItem>
            <MenuItem value="tcp">TCP</MenuItem>
            <MenuItem value="tls">TLS</MenuItem>
          </TextField>
          <TextField
            margin="dense"
            label="Host"
            fullWidth
            value={formData.host}
            onChange={(e) => setFormData({ ...formData, host: e.target.value })}
          />
          <TextField
            margin="dense"
            label="Port"
            type="number"
            fullWidth
            value={formData.port}
            onChange={(e) => setFormData({ ...formData, port: parseInt(e.target.value) })}
          />
          <TextField
            margin="dense"
            label="Path (optional)"
            fullWidth
            value={formData.path}
            onChange={(e) => setFormData({ ...formData, path: e.target.value })}
          />
          <TextField
            margin="dense"
            label="Retries"
            type="number"
            fullWidth
            value={formData.retries}
            onChange={(e) => setFormData({ ...formData, retries: parseInt(e.target.value) })}
          />
          <TextField
            margin="dense"
            label="Connect Timeout (ms)"
            type="number"
            fullWidth
            value={formData.connect_timeout}
            onChange={(e) => setFormData({ ...formData, connect_timeout: parseInt(e.target.value) })}
          />
          <FormControlLabel
            control={
              <Switch
                checked={formData.enabled}
                onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
              />
            }
            label="Enabled"
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button onClick={handleSubmit} variant="contained">
            {editingService ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default KongServices;

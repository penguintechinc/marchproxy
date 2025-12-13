import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  IconButton,
  Typography,
  Alert,
  Chip,
  Tooltip,
  Switch,
  FormControlLabel,
} from '@mui/material';
import {
  DataGrid,
  GridColDef,
  GridActionsCellItem,
  GridRowParams,
} from '@mui/x-data-grid';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  Refresh as RefreshIcon,
  VpnKey as KeyIcon,
} from '@mui/icons-material';
import { useForm, Controller } from 'react-hook-form';
import { clusterApi, CreateClusterRequest, UpdateClusterRequest } from '@services/clusterApi';
import { Cluster } from '@services/types';

interface ClusterFormData {
  name: string;
  description: string;
  syslog_server: string;
  syslog_port: number;
  auth_log_enabled: boolean;
  netflow_log_enabled: boolean;
  debug_log_enabled: boolean;
}

const Clusters: React.FC = () => {
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [editMode, setEditMode] = useState(false);
  const [selectedCluster, setSelectedCluster] = useState<Cluster | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null);
  const [newApiKey, setNewApiKey] = useState<string | null>(null);

  const { control, handleSubmit, reset, formState: { errors } } = useForm<ClusterFormData>({
    defaultValues: {
      name: '',
      description: '',
      syslog_server: '',
      syslog_port: 514,
      auth_log_enabled: true,
      netflow_log_enabled: false,
      debug_log_enabled: false,
    }
  });

  useEffect(() => {
    fetchClusters();
  }, []);

  const fetchClusters = async () => {
    try {
      setLoading(true);
      const response = await clusterApi.list();
      setClusters(response.items);
      setError(null);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to load clusters');
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    setEditMode(false);
    setSelectedCluster(null);
    reset({
      name: '',
      description: '',
      syslog_server: '',
      syslog_port: 514,
      auth_log_enabled: true,
      netflow_log_enabled: false,
      debug_log_enabled: false,
    });
    setOpenDialog(true);
  };

  const handleEdit = (cluster: Cluster) => {
    setEditMode(true);
    setSelectedCluster(cluster);
    reset({
      name: cluster.name,
      description: cluster.description,
      syslog_server: cluster.syslog_server || '',
      syslog_port: cluster.syslog_port || 514,
      auth_log_enabled: cluster.auth_log_enabled,
      netflow_log_enabled: cluster.netflow_log_enabled,
      debug_log_enabled: cluster.debug_log_enabled,
    });
    setOpenDialog(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await clusterApi.delete(id);
      fetchClusters();
      setDeleteConfirm(null);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to delete cluster');
    }
  };

  const handleRotateKey = async (id: number) => {
    try {
      const result = await clusterApi.rotateApiKey(id);
      setNewApiKey(result.api_key);
      fetchClusters();
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to rotate API key');
    }
  };

  const onSubmit = async (data: ClusterFormData) => {
    try {
      if (editMode && selectedCluster) {
        await clusterApi.update(selectedCluster.id, data as UpdateClusterRequest);
      } else {
        await clusterApi.create(data as CreateClusterRequest);
      }
      setOpenDialog(false);
      fetchClusters();
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to save cluster');
    }
  };

  const columns: GridColDef[] = [
    { field: 'id', headerName: 'ID', width: 70 },
    { field: 'name', headerName: 'Name', width: 200, flex: 1 },
    { field: 'description', headerName: 'Description', width: 300, flex: 1 },
    {
      field: 'proxy_count',
      headerName: 'Proxies',
      width: 100,
      renderCell: (params) => (
        <Chip label={params.value || 0} color="primary" size="small" />
      ),
    },
    {
      field: 'syslog_server',
      headerName: 'Syslog',
      width: 150,
      renderCell: (params) => params.value || 'Not configured',
    },
    {
      field: 'actions',
      type: 'actions',
      headerName: 'Actions',
      width: 150,
      getActions: (params: GridRowParams<Cluster>) => [
        <GridActionsCellItem
          icon={<EditIcon />}
          label="Edit"
          onClick={() => handleEdit(params.row)}
        />,
        <GridActionsCellItem
          icon={<KeyIcon />}
          label="Rotate API Key"
          onClick={() => handleRotateKey(params.row.id)}
        />,
        <GridActionsCellItem
          icon={<DeleteIcon />}
          label="Delete"
          onClick={() => setDeleteConfirm(params.row.id)}
        />,
      ],
    },
  ];

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}>
        <Typography variant="h4" fontWeight="bold">
          Cluster Management
        </Typography>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={fetchClusters}
          >
            Refresh
          </Button>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={handleCreate}
          >
            Add Cluster
          </Button>
        </Box>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {newApiKey && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setNewApiKey(null)}>
          <Typography variant="body2" fontWeight="bold">
            New API Key (save this, it won't be shown again):
          </Typography>
          <Typography variant="body2" sx={{ fontFamily: 'monospace', mt: 1 }}>
            {newApiKey}
          </Typography>
        </Alert>
      )}

      <DataGrid
        rows={clusters}
        columns={columns}
        loading={loading}
        autoHeight
        pageSizeOptions={[10, 25, 50]}
        initialState={{
          pagination: { paginationModel: { pageSize: 10 } },
        }}
        disableRowSelectionOnClick
      />

      {/* Create/Edit Dialog */}
      <Dialog open={openDialog} onClose={() => setOpenDialog(false)} maxWidth="md" fullWidth>
        <DialogTitle>{editMode ? 'Edit Cluster' : 'Create Cluster'}</DialogTitle>
        <form onSubmit={handleSubmit(onSubmit)}>
          <DialogContent>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <Controller
                name="name"
                control={control}
                rules={{ required: 'Name is required' }}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Cluster Name"
                    fullWidth
                    error={!!errors.name}
                    helperText={errors.name?.message}
                  />
                )}
              />
              <Controller
                name="description"
                control={control}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Description"
                    fullWidth
                    multiline
                    rows={2}
                  />
                )}
              />
              <Controller
                name="syslog_server"
                control={control}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Syslog Server"
                    fullWidth
                    placeholder="syslog.example.com"
                  />
                )}
              />
              <Controller
                name="syslog_port"
                control={control}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Syslog Port"
                    type="number"
                    fullWidth
                  />
                )}
              />
              <Controller
                name="auth_log_enabled"
                control={control}
                render={({ field }) => (
                  <FormControlLabel
                    control={<Switch {...field} checked={field.value} />}
                    label="Authentication Logging"
                  />
                )}
              />
              <Controller
                name="netflow_log_enabled"
                control={control}
                render={({ field }) => (
                  <FormControlLabel
                    control={<Switch {...field} checked={field.value} />}
                    label="Netflow Logging"
                  />
                )}
              />
              <Controller
                name="debug_log_enabled"
                control={control}
                render={({ field }) => (
                  <FormControlLabel
                    control={<Switch {...field} checked={field.value} />}
                    label="Debug Logging"
                  />
                )}
              />
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setOpenDialog(false)}>Cancel</Button>
            <Button type="submit" variant="contained">
              {editMode ? 'Update' : 'Create'}
            </Button>
          </DialogActions>
        </form>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteConfirm !== null} onClose={() => setDeleteConfirm(null)}>
        <DialogTitle>Confirm Delete</DialogTitle>
        <DialogContent>
          <Typography>Are you sure you want to delete this cluster?</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteConfirm(null)}>Cancel</Button>
          <Button
            onClick={() => deleteConfirm && handleDelete(deleteConfirm)}
            variant="contained"
            color="error"
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Clusters;

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
  Card,
  CardContent,
  Grid,
  Divider,
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
  ContentCopy as CopyIcon,
  Info as InfoIcon,
} from '@mui/icons-material';
import { useForm, Controller } from 'react-hook-form';
import { clusterApi, CreateClusterRequest, UpdateClusterRequest } from '@services/clusterApi';
import { Cluster } from '@services/types';
import { formatDistanceToNow } from 'date-fns';

interface ClusterFormData {
  name: string;
  description: string;
  syslog_endpoint: string;
  log_auth: boolean;
  log_netflow: boolean;
  log_debug: boolean;
  max_proxies: number;
}

const ClusterManagement: React.FC = () => {
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [editMode, setEditMode] = useState(false);
  const [selectedCluster, setSelectedCluster] = useState<Cluster | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null);
  const [rotateConfirm, setRotateConfirm] = useState<number | null>(null);
  const [newApiKey, setNewApiKey] = useState<string | null>(null);
  const [viewDetailsDialog, setViewDetailsDialog] = useState(false);

  const { control, handleSubmit, reset, formState: { errors } } = useForm<ClusterFormData>({
    defaultValues: {
      name: '',
      description: '',
      syslog_endpoint: '',
      log_auth: true,
      log_netflow: false,
      log_debug: false,
      max_proxies: 3,
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
      syslog_endpoint: '',
      log_auth: true,
      log_netflow: false,
      log_debug: false,
      max_proxies: 3,
    });
    setOpenDialog(true);
  };

  const handleEdit = (cluster: Cluster) => {
    setEditMode(true);
    setSelectedCluster(cluster);
    reset({
      name: cluster.name,
      description: cluster.description || '',
      syslog_endpoint: cluster.syslog_endpoint || '',
      log_auth: cluster.log_auth,
      log_netflow: cluster.log_netflow,
      log_debug: cluster.log_debug,
      max_proxies: cluster.max_proxies,
    });
    setOpenDialog(true);
  };

  const handleViewDetails = (cluster: Cluster) => {
    setSelectedCluster(cluster);
    setViewDetailsDialog(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await clusterApi.delete(id);
      setSuccess('Cluster deleted successfully');
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
      setSuccess('API key rotated successfully. Save it now - it will not be shown again!');
      fetchClusters();
      setRotateConfirm(null);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to rotate API key');
    }
  };

  const onSubmit = async (data: ClusterFormData) => {
    try {
      if (editMode && selectedCluster) {
        await clusterApi.update(selectedCluster.id, data as UpdateClusterRequest);
        setSuccess('Cluster updated successfully');
      } else {
        const newCluster = await clusterApi.create(data as CreateClusterRequest);
        setSuccess('Cluster created successfully');
        // Check if API key is returned (it should be for new clusters)
        if ((newCluster as any).api_key) {
          setNewApiKey((newCluster as any).api_key);
        }
      }
      setOpenDialog(false);
      fetchClusters();
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to save cluster');
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setSuccess('Copied to clipboard!');
  };

  const columns: GridColDef[] = [
    { field: 'id', headerName: 'ID', width: 70 },
    {
      field: 'name',
      headerName: 'Name',
      width: 200,
      flex: 1,
      renderCell: (params) => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          {params.value}
          {params.row.is_default && (
            <Chip label="Default" size="small" color="info" />
          )}
        </Box>
      ),
    },
    { field: 'description', headerName: 'Description', width: 250, flex: 1 },
    {
      field: 'proxy_count',
      headerName: 'Proxies',
      width: 120,
      renderCell: (params) => (
        <Chip
          label={`${params.value || 0}/${params.row.max_proxies}`}
          color={params.value >= params.row.max_proxies ? 'warning' : 'primary'}
          size="small"
        />
      ),
    },
    {
      field: 'syslog_endpoint',
      headerName: 'Syslog',
      width: 180,
      renderCell: (params) => params.value || 'Not configured',
    },
    {
      field: 'is_active',
      headerName: 'Status',
      width: 100,
      renderCell: (params) => (
        <Chip
          label={params.value ? 'Active' : 'Inactive'}
          color={params.value ? 'success' : 'default'}
          size="small"
        />
      ),
    },
    {
      field: 'created_at',
      headerName: 'Created',
      width: 150,
      renderCell: (params) => {
        try {
          return formatDistanceToNow(new Date(params.value), { addSuffix: true });
        } catch {
          return params.value;
        }
      },
    },
    {
      field: 'actions',
      type: 'actions',
      headerName: 'Actions',
      width: 180,
      getActions: (params: GridRowParams<Cluster>) => [
        <GridActionsCellItem
          icon={
            <Tooltip title="View Details">
              <InfoIcon />
            </Tooltip>
          }
          label="Details"
          onClick={() => handleViewDetails(params.row)}
        />,
        <GridActionsCellItem
          icon={
            <Tooltip title="Edit Cluster">
              <EditIcon />
            </Tooltip>
          }
          label="Edit"
          onClick={() => handleEdit(params.row)}
        />,
        <GridActionsCellItem
          icon={
            <Tooltip title="Rotate API Key">
              <KeyIcon />
            </Tooltip>
          }
          label="Rotate API Key"
          onClick={() => setRotateConfirm(params.row.id)}
        />,
        <GridActionsCellItem
          icon={
            <Tooltip title={params.row.is_default ? 'Cannot delete default cluster' : 'Delete Cluster'}>
              <DeleteIcon />
            </Tooltip>
          }
          label="Delete"
          onClick={() => setDeleteConfirm(params.row.id)}
          disabled={params.row.is_default}
        />,
      ],
    },
  ];

  // Calculate statistics
  const stats = {
    total: clusters.length,
    active: clusters.filter(c => c.is_active).length,
    inactive: clusters.filter(c => !c.is_active).length,
    totalProxies: clusters.reduce((sum, c) => sum + (c.proxy_count || 0), 0),
  };

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

      {success && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setSuccess(null)}>
          {success}
        </Alert>
      )}

      {newApiKey && (
        <Alert
          severity="warning"
          sx={{ mb: 2 }}
          action={
            <IconButton
              color="inherit"
              size="small"
              onClick={() => copyToClipboard(newApiKey)}
            >
              <CopyIcon />
            </IconButton>
          }
          onClose={() => setNewApiKey(null)}
        >
          <Typography variant="body2" fontWeight="bold">
            New API Key (save this, it won't be shown again):
          </Typography>
          <Typography variant="body2" sx={{ fontFamily: 'monospace', mt: 1, wordBreak: 'break-all' }}>
            {newApiKey}
          </Typography>
        </Alert>
      )}

      {/* Statistics Cards */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom variant="body2">
                Total Clusters
              </Typography>
              <Typography variant="h4">{stats.total}</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom variant="body2">
                Active Clusters
              </Typography>
              <Typography variant="h4" color="success.main">
                {stats.active}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom variant="body2">
                Inactive Clusters
              </Typography>
              <Typography variant="h4" color="text.secondary">
                {stats.inactive}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom variant="body2">
                Total Proxies
              </Typography>
              <Typography variant="h4" color="primary.main">
                {stats.totalProxies}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <DataGrid
        rows={clusters}
        columns={columns}
        loading={loading}
        autoHeight
        pageSizeOptions={[10, 25, 50, 100]}
        initialState={{
          pagination: { paginationModel: { pageSize: 10 } },
        }}
        disableRowSelectionOnClick
        sx={{
          '& .MuiDataGrid-cell:focus': {
            outline: 'none',
          },
        }}
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
                rules={{ required: 'Name is required', minLength: { value: 1, message: 'Name cannot be empty' } }}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Cluster Name"
                    fullWidth
                    required
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
                name="syslog_endpoint"
                control={control}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Syslog Endpoint"
                    fullWidth
                    placeholder="syslog.example.com:514"
                    helperText="Format: hostname:port"
                  />
                )}
              />
              <Controller
                name="max_proxies"
                control={control}
                rules={{ min: { value: 1, message: 'Must be at least 1' } }}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Max Proxies"
                    type="number"
                    fullWidth
                    error={!!errors.max_proxies}
                    helperText={errors.max_proxies?.message || 'Community tier limited to 3'}
                    inputProps={{ min: 1 }}
                  />
                )}
              />
              <Divider sx={{ my: 1 }} />
              <Typography variant="subtitle2" color="text.secondary">
                Logging Configuration
              </Typography>
              <Controller
                name="log_auth"
                control={control}
                render={({ field }) => (
                  <FormControlLabel
                    control={<Switch {...field} checked={field.value} />}
                    label="Authentication Logging"
                  />
                )}
              />
              <Controller
                name="log_netflow"
                control={control}
                render={({ field }) => (
                  <FormControlLabel
                    control={<Switch {...field} checked={field.value} />}
                    label="Netflow Logging"
                  />
                )}
              />
              <Controller
                name="log_debug"
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

      {/* View Details Dialog */}
      <Dialog
        open={viewDetailsDialog}
        onClose={() => setViewDetailsDialog(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>Cluster Details</DialogTitle>
        <DialogContent>
          {selectedCluster && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <Box>
                <Typography variant="subtitle2" color="text.secondary">ID</Typography>
                <Typography variant="body1">{selectedCluster.id}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2" color="text.secondary">Name</Typography>
                <Typography variant="body1">{selectedCluster.name}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2" color="text.secondary">Description</Typography>
                <Typography variant="body1">{selectedCluster.description || 'N/A'}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2" color="text.secondary">API Key Hash</Typography>
                <Typography variant="body2" sx={{ fontFamily: 'monospace', wordBreak: 'break-all' }}>
                  {selectedCluster.api_key_hash}
                </Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2" color="text.secondary">Proxies</Typography>
                <Typography variant="body1">
                  {selectedCluster.proxy_count} / {selectedCluster.max_proxies}
                </Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2" color="text.secondary">Syslog Endpoint</Typography>
                <Typography variant="body1">{selectedCluster.syslog_endpoint || 'Not configured'}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2" color="text.secondary">Logging</Typography>
                <Box sx={{ display: 'flex', gap: 1, mt: 1 }}>
                  <Chip
                    label="Auth"
                    color={selectedCluster.log_auth ? 'success' : 'default'}
                    size="small"
                  />
                  <Chip
                    label="Netflow"
                    color={selectedCluster.log_netflow ? 'success' : 'default'}
                    size="small"
                  />
                  <Chip
                    label="Debug"
                    color={selectedCluster.log_debug ? 'success' : 'default'}
                    size="small"
                  />
                </Box>
              </Box>
              <Box>
                <Typography variant="subtitle2" color="text.secondary">Status</Typography>
                <Chip
                  label={selectedCluster.is_active ? 'Active' : 'Inactive'}
                  color={selectedCluster.is_active ? 'success' : 'default'}
                  size="small"
                />
              </Box>
              <Box>
                <Typography variant="subtitle2" color="text.secondary">Created</Typography>
                <Typography variant="body1">
                  {new Date(selectedCluster.created_at).toLocaleString()}
                </Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2" color="text.secondary">Last Updated</Typography>
                <Typography variant="body1">
                  {new Date(selectedCluster.updated_at).toLocaleString()}
                </Typography>
              </Box>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setViewDetailsDialog(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteConfirm !== null} onClose={() => setDeleteConfirm(null)}>
        <DialogTitle>Confirm Delete</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to delete this cluster? This action cannot be undone.
          </Typography>
          <Alert severity="warning" sx={{ mt: 2 }}>
            All proxies and services associated with this cluster will be affected.
          </Alert>
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

      {/* Rotate API Key Confirmation Dialog */}
      <Dialog open={rotateConfirm !== null} onClose={() => setRotateConfirm(null)}>
        <DialogTitle>Confirm API Key Rotation</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to rotate the API key for this cluster?
          </Typography>
          <Alert severity="warning" sx={{ mt: 2 }}>
            The old API key will be immediately invalidated. All proxies must be reconfigured with the new key.
          </Alert>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRotateConfirm(null)}>Cancel</Button>
          <Button
            onClick={() => rotateConfirm && handleRotateKey(rotateConfirm)}
            variant="contained"
            color="warning"
          >
            Rotate Key
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default ClusterManagement;

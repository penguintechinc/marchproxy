import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Typography,
  Alert,
  Chip,
  Switch,
  FormControlLabel,
  MenuItem,
  Select,
  FormControl,
  InputLabel,
  IconButton,
  Tooltip,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  Divider,
  Paper,
  Grid,
  InputAdornment,
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
  VpnKey as TokenIcon,
  ContentCopy as CopyIcon,
  Info as InfoIcon,
  Link as LinkIcon,
  Close as CloseIcon,
  Search as SearchIcon,
} from '@mui/icons-material';
import { useForm, Controller } from 'react-hook-form';
import { serviceApi, CreateServiceRequest, UpdateServiceRequest } from '@services/serviceApi';
import { clusterApi } from '@services/clusterApi';
import { Service, Cluster } from '@services/types';

interface ServiceFormData {
  cluster_id: number;
  name: string;
  description: string;
  destination_fqdn: string;
  destination_port: string;
  protocol: 'TCP' | 'UDP' | 'ICMP' | 'HTTPS' | 'HTTP3';
  auth_method: 'base64_token' | 'jwt';
  token_ttl: number;
  is_active: boolean;
}

const Services: React.FC = () => {
  const [services, setServices] = useState<Service[]>([]);
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [editMode, setEditMode] = useState(false);
  const [selectedService, setSelectedService] = useState<Service | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null);
  const [newToken, setNewToken] = useState<string | null>(null);
  const [filterCluster, setFilterCluster] = useState<number | null>(null);
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [detailsService, setDetailsService] = useState<Service | null>(null);
  const [mappingsDialog, setMappingsDialog] = useState<Service | null>(null);
  const [serviceMappings, setServiceMappings] = useState<Service[]>([]);
  const [availableServices, setAvailableServices] = useState<Service[]>([]);
  const [copySuccess, setCopySuccess] = useState<string | null>(null);

  const { control, handleSubmit, reset, watch, formState: { errors } } = useForm<ServiceFormData>({
    defaultValues: {
      cluster_id: 0,
      name: '',
      description: '',
      destination_fqdn: '',
      destination_port: '443',
      protocol: 'HTTPS',
      auth_method: 'jwt',
      token_ttl: 3600,
      is_active: true,
    }
  });

  const authMethod = watch('auth_method');

  useEffect(() => {
    fetchClusters();
    fetchServices();
  }, [filterCluster, searchQuery]);

  const fetchClusters = async () => {
    try {
      const response = await clusterApi.list();
      setClusters(response.items);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to load clusters');
    }
  };

  const fetchServices = async () => {
    try {
      setLoading(true);
      const response = await serviceApi.list({
        cluster_id: filterCluster || undefined,
        search: searchQuery || undefined,
      });
      setServices(response.items);
      setError(null);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to load services');
    } finally {
      setLoading(false);
    }
  };

  const fetchServiceMappings = async (serviceId: number) => {
    try {
      const mappings = await serviceApi.getMappings(serviceId);
      setServiceMappings(mappings);

      // Get available services (all services except the selected one and already mapped ones)
      const allServices = await serviceApi.list({ cluster_id: mappingsDialog?.cluster_id });
      const mappedIds = new Set(mappings.map(m => m.id));
      const available = allServices.items.filter(
        s => s.id !== serviceId && !mappedIds.has(s.id)
      );
      setAvailableServices(available);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to load service mappings');
    }
  };

  const handleCreate = () => {
    setEditMode(false);
    setSelectedService(null);
    reset({
      cluster_id: filterCluster || 0,
      name: '',
      description: '',
      destination_fqdn: '',
      destination_port: '443',
      protocol: 'HTTPS',
      auth_method: 'jwt',
      token_ttl: 3600,
      is_active: true,
    });
    setOpenDialog(true);
  };

  const handleEdit = (service: Service) => {
    setEditMode(true);
    setSelectedService(service);
    reset({
      cluster_id: service.cluster_id,
      name: service.name,
      description: service.description,
      destination_fqdn: service.destination_fqdn,
      destination_port: service.destination_port,
      protocol: service.protocol,
      auth_method: service.auth_method,
      token_ttl: service.token_ttl || 3600,
      is_active: service.is_active,
    });
    setOpenDialog(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await serviceApi.delete(id);
      fetchServices();
      setDeleteConfirm(null);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to delete service');
    }
  };

  const handleRegenerateToken = async (id: number) => {
    try {
      const result = await serviceApi.regenerateToken(id);
      setNewToken(result.token);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to regenerate token');
    }
  };

  const handleCopyToken = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopySuccess('Token copied to clipboard!');
      setTimeout(() => setCopySuccess(null), 3000);
    } catch (err) {
      setError('Failed to copy to clipboard');
    }
  };

  const handleOpenMappings = async (service: Service) => {
    setMappingsDialog(service);
    await fetchServiceMappings(service.id);
  };

  const handleAddMapping = async (targetServiceId: number) => {
    if (!mappingsDialog) return;
    try {
      await serviceApi.addMapping(mappingsDialog.id, targetServiceId);
      await fetchServiceMappings(mappingsDialog.id);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to add service mapping');
    }
  };

  const handleRemoveMapping = async (targetServiceId: number) => {
    if (!mappingsDialog) return;
    try {
      await serviceApi.removeMapping(mappingsDialog.id, targetServiceId);
      await fetchServiceMappings(mappingsDialog.id);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to remove service mapping');
    }
  };

  const getClusterName = (clusterId: number): string => {
    const cluster = clusters.find(c => c.id === clusterId);
    return cluster?.name || `Cluster ${clusterId}`;
  };

  const validatePort = (value: string): boolean | string => {
    // Single port: 443
    // Range: 8000-8100
    // Comma-separated: 80,443,8080
    const portRegex = /^(\d+(-\d+)?)(,\d+(-\d+)?)*$/;
    if (!portRegex.test(value)) {
      return 'Invalid port format. Use single (443), range (8000-8100), or comma-separated (80,443,8080)';
    }

    // Validate individual ports are in valid range (1-65535)
    const parts = value.split(',');
    for (const part of parts) {
      if (part.includes('-')) {
        const [start, end] = part.split('-').map(Number);
        if (start < 1 || start > 65535 || end < 1 || end > 65535 || start >= end) {
          return 'Port numbers must be between 1-65535 and start < end in ranges';
        }
      } else {
        const port = Number(part);
        if (port < 1 || port > 65535) {
          return 'Port numbers must be between 1-65535';
        }
      }
    }
    return true;
  };

  const onSubmit = async (data: ServiceFormData) => {
    try {
      if (editMode && selectedService) {
        await serviceApi.update(selectedService.id, data as UpdateServiceRequest);
      } else {
        await serviceApi.create(data as CreateServiceRequest);
      }
      setOpenDialog(false);
      fetchServices();
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to save service');
    }
  };

  const columns: GridColDef[] = [
    { field: 'id', headerName: 'ID', width: 70 },
    { field: 'name', headerName: 'Service Name', width: 200, flex: 1 },
    {
      field: 'cluster_id',
      headerName: 'Cluster',
      width: 150,
      renderCell: (params) => (
        <Chip label={getClusterName(params.value)} size="small" variant="outlined" />
      ),
    },
    {
      field: 'destination_fqdn',
      headerName: 'Destination',
      width: 250,
      flex: 1,
      renderCell: (params) => `${params.value}:${params.row.destination_port}`,
    },
    {
      field: 'protocol',
      headerName: 'Protocol',
      width: 100,
      renderCell: (params) => (
        <Chip label={params.value} size="small" color="primary" variant="outlined" />
      ),
    },
    {
      field: 'auth_method',
      headerName: 'Auth Method',
      width: 130,
      renderCell: (params) => (
        <Chip
          label={params.value === 'jwt' ? 'JWT' : 'Base64'}
          size="small"
          color={params.value === 'jwt' ? 'success' : 'default'}
        />
      ),
    },
    {
      field: 'is_active',
      headerName: 'Status',
      width: 100,
      renderCell: (params) => (
        <Chip
          label={params.value ? 'Active' : 'Inactive'}
          size="small"
          color={params.value ? 'success' : 'default'}
        />
      ),
    },
    {
      field: 'actions',
      type: 'actions',
      headerName: 'Actions',
      width: 200,
      getActions: (params: GridRowParams<Service>) => [
        <GridActionsCellItem
          icon={<InfoIcon />}
          label="Details"
          onClick={() => setDetailsService(params.row)}
        />,
        <GridActionsCellItem
          icon={<LinkIcon />}
          label="Service Mappings"
          onClick={() => handleOpenMappings(params.row)}
        />,
        <GridActionsCellItem
          icon={<EditIcon />}
          label="Edit"
          onClick={() => handleEdit(params.row)}
        />,
        <GridActionsCellItem
          icon={<TokenIcon />}
          label="Regenerate Token"
          onClick={() => handleRegenerateToken(params.row.id)}
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
          Service Management
        </Typography>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <TextField
            size="small"
            placeholder="Search services..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon />
                </InputAdornment>
              ),
            }}
            sx={{ minWidth: 250 }}
          />
          <FormControl size="small" sx={{ minWidth: 200 }}>
            <InputLabel>Filter by Cluster</InputLabel>
            <Select
              value={filterCluster || ''}
              onChange={(e) => setFilterCluster(e.target.value as number || null)}
              label="Filter by Cluster"
            >
              <MenuItem value="">All Clusters</MenuItem>
              {clusters.map((cluster) => (
                <MenuItem key={cluster.id} value={cluster.id}>
                  {cluster.name}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={fetchServices}
          >
            Refresh
          </Button>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={handleCreate}
          >
            Add Service
          </Button>
        </Box>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {copySuccess && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setCopySuccess(null)}>
          {copySuccess}
        </Alert>
      )}

      {newToken && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setNewToken(null)}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Box>
              <Typography variant="body2" fontWeight="bold">
                New Service Token (save this, it won't be shown again):
              </Typography>
              <Typography variant="body2" sx={{ fontFamily: 'monospace', mt: 1 }}>
                {newToken}
              </Typography>
            </Box>
            <Tooltip title="Copy to clipboard">
              <IconButton onClick={() => handleCopyToken(newToken)} size="small">
                <CopyIcon />
              </IconButton>
            </Tooltip>
          </Box>
        </Alert>
      )}

      <DataGrid
        rows={services}
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
        <DialogTitle>{editMode ? 'Edit Service' : 'Create Service'}</DialogTitle>
        <form onSubmit={handleSubmit(onSubmit)}>
          <DialogContent>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <Controller
                name="cluster_id"
                control={control}
                rules={{ required: 'Cluster is required', min: { value: 1, message: 'Please select a cluster' } }}
                render={({ field }) => (
                  <FormControl fullWidth error={!!errors.cluster_id}>
                    <InputLabel>Cluster</InputLabel>
                    <Select {...field} label="Cluster">
                      <MenuItem value={0}>Select a cluster</MenuItem>
                      {clusters.map((cluster) => (
                        <MenuItem key={cluster.id} value={cluster.id}>
                          {cluster.name}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                )}
              />
              <Controller
                name="name"
                control={control}
                rules={{ required: 'Name is required' }}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Service Name"
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
                name="destination_fqdn"
                control={control}
                rules={{ required: 'Destination FQDN is required' }}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Destination FQDN"
                    fullWidth
                    placeholder="api.example.com"
                    error={!!errors.destination_fqdn}
                    helperText={errors.destination_fqdn?.message}
                  />
                )}
              />
              <Controller
                name="destination_port"
                control={control}
                rules={{
                  required: 'Port is required',
                  validate: validatePort
                }}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Destination Port"
                    fullWidth
                    placeholder="443 or 8000-8100 or 80,443,8080"
                    error={!!errors.destination_port}
                    helperText={errors.destination_port?.message || 'Single port, range, or comma-separated'}
                  />
                )}
              />
              <Controller
                name="protocol"
                control={control}
                render={({ field }) => (
                  <FormControl fullWidth>
                    <InputLabel>Protocol</InputLabel>
                    <Select {...field} label="Protocol">
                      <MenuItem value="TCP">TCP</MenuItem>
                      <MenuItem value="UDP">UDP</MenuItem>
                      <MenuItem value="ICMP">ICMP</MenuItem>
                      <MenuItem value="HTTPS">HTTPS</MenuItem>
                      <MenuItem value="HTTP3">HTTP3/QUIC</MenuItem>
                    </Select>
                  </FormControl>
                )}
              />
              <Controller
                name="auth_method"
                control={control}
                render={({ field }) => (
                  <FormControl fullWidth>
                    <InputLabel>Authentication Method</InputLabel>
                    <Select {...field} label="Authentication Method">
                      <MenuItem value="jwt">JWT Token</MenuItem>
                      <MenuItem value="base64_token">Base64 Token</MenuItem>
                    </Select>
                  </FormControl>
                )}
              />
              {authMethod === 'jwt' && (
                <Controller
                  name="token_ttl"
                  control={control}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label="Token TTL (seconds)"
                      type="number"
                      fullWidth
                    />
                  )}
                />
              )}
              <Controller
                name="is_active"
                control={control}
                render={({ field }) => (
                  <FormControlLabel
                    control={<Switch {...field} checked={field.value} />}
                    label="Active"
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
          <Typography>Are you sure you want to delete this service?</Typography>
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

      {/* Service Details Dialog */}
      <Dialog open={detailsService !== null} onClose={() => setDetailsService(null)} maxWidth="md" fullWidth>
        <DialogTitle>
          Service Details
          <IconButton
            onClick={() => setDetailsService(null)}
            sx={{ position: 'absolute', right: 8, top: 8 }}
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent>
          {detailsService && (
            <Grid container spacing={2} sx={{ mt: 1 }}>
              <Grid item xs={12} sm={6}>
                <Paper sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary">Service Name</Typography>
                  <Typography variant="body1" fontWeight="bold">{detailsService.name}</Typography>
                </Paper>
              </Grid>
              <Grid item xs={12} sm={6}>
                <Paper sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary">Cluster</Typography>
                  <Typography variant="body1" fontWeight="bold">{getClusterName(detailsService.cluster_id)}</Typography>
                </Paper>
              </Grid>
              <Grid item xs={12}>
                <Paper sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary">Description</Typography>
                  <Typography variant="body1">{detailsService.description || 'No description'}</Typography>
                </Paper>
              </Grid>
              <Grid item xs={12} sm={6}>
                <Paper sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary">Destination</Typography>
                  <Typography variant="body1" fontWeight="bold">
                    {detailsService.destination_fqdn}:{detailsService.destination_port}
                  </Typography>
                </Paper>
              </Grid>
              <Grid item xs={12} sm={6}>
                <Paper sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary">Protocol</Typography>
                  <Typography variant="body1">
                    <Chip label={detailsService.protocol} color="primary" size="small" />
                  </Typography>
                </Paper>
              </Grid>
              <Grid item xs={12} sm={6}>
                <Paper sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary">Authentication Method</Typography>
                  <Typography variant="body1">
                    <Chip
                      label={detailsService.auth_method === 'jwt' ? 'JWT Token' : 'Base64 Token'}
                      color={detailsService.auth_method === 'jwt' ? 'success' : 'default'}
                      size="small"
                    />
                  </Typography>
                </Paper>
              </Grid>
              <Grid item xs={12} sm={6}>
                <Paper sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary">Status</Typography>
                  <Typography variant="body1">
                    <Chip
                      label={detailsService.is_active ? 'Active' : 'Inactive'}
                      color={detailsService.is_active ? 'success' : 'default'}
                      size="small"
                    />
                  </Typography>
                </Paper>
              </Grid>
              {detailsService.auth_method === 'jwt' && detailsService.token_ttl && (
                <Grid item xs={12} sm={6}>
                  <Paper sx={{ p: 2 }}>
                    <Typography variant="caption" color="text.secondary">Token TTL</Typography>
                    <Typography variant="body1">{detailsService.token_ttl} seconds</Typography>
                  </Paper>
                </Grid>
              )}
              <Grid item xs={12} sm={6}>
                <Paper sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary">Created</Typography>
                  <Typography variant="body1">{new Date(detailsService.created_at).toLocaleString()}</Typography>
                </Paper>
              </Grid>
              <Grid item xs={12} sm={6}>
                <Paper sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary">Last Updated</Typography>
                  <Typography variant="body1">{new Date(detailsService.updated_at).toLocaleString()}</Typography>
                </Paper>
              </Grid>
            </Grid>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDetailsService(null)}>Close</Button>
        </DialogActions>
      </Dialog>

      {/* Service-to-Service Mappings Dialog */}
      <Dialog
        open={mappingsDialog !== null}
        onClose={() => setMappingsDialog(null)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          Service-to-Service Mappings for {mappingsDialog?.name}
          <IconButton
            onClick={() => setMappingsDialog(null)}
            sx={{ position: 'absolute', right: 8, top: 8 }}
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Configure which services can access this service. Only mapped services will be allowed to connect.
          </Typography>

          <Typography variant="h6" sx={{ mt: 2, mb: 1 }}>
            Current Mappings
          </Typography>
          {serviceMappings.length === 0 ? (
            <Alert severity="info" sx={{ mb: 2 }}>
              No service mappings configured. All services in the cluster can access this service.
            </Alert>
          ) : (
            <Paper variant="outlined" sx={{ mb: 3 }}>
              <List>
                {serviceMappings.map((service, index) => (
                  <React.Fragment key={service.id}>
                    {index > 0 && <Divider />}
                    <ListItem>
                      <ListItemText
                        primary={service.name}
                        secondary={`${service.destination_fqdn}:${service.destination_port} (${service.protocol})`}
                      />
                      <ListItemSecondaryAction>
                        <IconButton
                          edge="end"
                          onClick={() => handleRemoveMapping(service.id)}
                          color="error"
                        >
                          <DeleteIcon />
                        </IconButton>
                      </ListItemSecondaryAction>
                    </ListItem>
                  </React.Fragment>
                ))}
              </List>
            </Paper>
          )}

          <Typography variant="h6" sx={{ mt: 3, mb: 1 }}>
            Add Service Mapping
          </Typography>
          {availableServices.length === 0 ? (
            <Alert severity="info">
              No additional services available to map in this cluster.
            </Alert>
          ) : (
            <Paper variant="outlined">
              <List>
                {availableServices.map((service, index) => (
                  <React.Fragment key={service.id}>
                    {index > 0 && <Divider />}
                    <ListItem>
                      <ListItemText
                        primary={service.name}
                        secondary={`${service.destination_fqdn}:${service.destination_port} (${service.protocol})`}
                      />
                      <ListItemSecondaryAction>
                        <Button
                          size="small"
                          variant="outlined"
                          startIcon={<AddIcon />}
                          onClick={() => handleAddMapping(service.id)}
                        >
                          Add
                        </Button>
                      </ListItemSecondaryAction>
                    </ListItem>
                  </React.Fragment>
                ))}
              </List>
            </Paper>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setMappingsDialog(null)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Services;

import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Typography,
  Alert,
  Chip,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Card,
  CardContent,
  Grid,
} from '@mui/material';
import {
  DataGrid,
  GridColDef,
  GridActionsCellItem,
  GridRowParams,
} from '@mui/x-data-grid';
import {
  Delete as DeleteIcon,
  Refresh as RefreshIcon,
  CheckCircle as OnlineIcon,
  Cancel as OfflineIcon,
  Error as ErrorIcon,
} from '@mui/icons-material';
import { proxyApi, ProxyListParams } from '@services/proxyApi';
import { clusterApi } from '@services/clusterApi';
import { Proxy, Cluster } from '@services/types';
import { formatDistanceToNow } from 'date-fns';

const Proxies: React.FC = () => {
  const [proxies, setProxies] = useState<Proxy[]>([]);
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null);
  const [filterCluster, setFilterCluster] = useState<number | null>(null);
  const [filterStatus, setFilterStatus] = useState<string>('');

  useEffect(() => {
    fetchClusters();
    fetchProxies();

    // Set up real-time updates every 10 seconds
    const interval = setInterval(fetchProxies, 10000);
    return () => clearInterval(interval);
  }, [filterCluster, filterStatus]);

  const fetchClusters = async () => {
    try {
      const response = await clusterApi.list();
      setClusters(response.items);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to load clusters');
    }
  };

  const fetchProxies = async () => {
    try {
      setLoading(true);
      const params: ProxyListParams = {
        cluster_id: filterCluster || undefined,
        status: filterStatus as any || undefined,
      };
      const response = await proxyApi.list(params);
      setProxies(response.items);
      setError(null);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to load proxies');
    } finally {
      setLoading(false);
    }
  };

  const handleDeregister = async (id: number) => {
    try {
      await proxyApi.deregister(id);
      fetchProxies();
      setDeleteConfirm(null);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to deregister proxy');
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active':
        return <OnlineIcon sx={{ color: 'success.main' }} />;
      case 'inactive':
        return <OfflineIcon sx={{ color: 'text.disabled' }} />;
      case 'error':
        return <ErrorIcon sx={{ color: 'error.main' }} />;
      default:
        return null;
    }
  };

  const getStatusColor = (status: string): 'success' | 'default' | 'error' => {
    switch (status) {
      case 'active':
        return 'success';
      case 'error':
        return 'error';
      default:
        return 'default';
    }
  };

  const columns: GridColDef[] = [
    { field: 'id', headerName: 'ID', width: 70 },
    {
      field: 'hostname',
      headerName: 'Hostname',
      width: 200,
      flex: 1,
    },
    {
      field: 'ip_address',
      headerName: 'IP Address',
      width: 150,
    },
    {
      field: 'status',
      headerName: 'Status',
      width: 120,
      renderCell: (params) => {
        const statusIcon = getStatusIcon(params.value);
        return (
          <Chip
            icon={statusIcon || undefined}
            label={params.value}
            size="small"
            color={getStatusColor(params.value)}
            sx={{ textTransform: 'capitalize' }}
          />
        );
      },
    },
    {
      field: 'version',
      headerName: 'Version',
      width: 120,
    },
    {
      field: 'last_heartbeat',
      headerName: 'Last Heartbeat',
      width: 180,
      renderCell: (params) => {
        if (!params.value) return 'Never';
        try {
          return formatDistanceToNow(new Date(params.value), { addSuffix: true });
        } catch {
          return params.value;
        }
      },
    },
    {
      field: 'capabilities',
      headerName: 'Capabilities',
      width: 200,
      flex: 1,
      renderCell: (params) => {
        const caps = params.value as string[];
        if (!caps || caps.length === 0) return 'None';
        return (
          <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
            {caps.slice(0, 2).map((cap: string) => (
              <Chip key={cap} label={cap} size="small" variant="outlined" />
            ))}
            {caps.length > 2 && (
              <Chip label={`+${caps.length - 2}`} size="small" variant="outlined" />
            )}
          </Box>
        );
      },
    },
    {
      field: 'actions',
      type: 'actions',
      headerName: 'Actions',
      width: 100,
      getActions: (params: GridRowParams<Proxy>) => [
        <GridActionsCellItem
          icon={<DeleteIcon />}
          label="Deregister"
          onClick={() => setDeleteConfirm(params.row.id)}
        />,
      ],
    },
  ];

  // Calculate statistics
  const stats = {
    total: proxies.length,
    active: proxies.filter(p => p.status === 'active').length,
    inactive: proxies.filter(p => p.status === 'inactive').length,
    error: proxies.filter(p => p.status === 'error').length,
  };

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}>
        <Typography variant="h4" fontWeight="bold">
          Proxy Management
        </Typography>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <FormControl sx={{ minWidth: 200 }}>
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
          <FormControl sx={{ minWidth: 150 }}>
            <InputLabel>Status</InputLabel>
            <Select
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value)}
              label="Status"
            >
              <MenuItem value="">All</MenuItem>
              <MenuItem value="active">Active</MenuItem>
              <MenuItem value="inactive">Inactive</MenuItem>
              <MenuItem value="error">Error</MenuItem>
            </Select>
          </FormControl>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={fetchProxies}
          >
            Refresh
          </Button>
        </Box>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* Statistics Cards */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                Total Proxies
              </Typography>
              <Typography variant="h4">{stats.total}</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                Active
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
              <Typography color="text.secondary" gutterBottom>
                Inactive
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
              <Typography color="text.secondary" gutterBottom>
                Errors
              </Typography>
              <Typography variant="h4" color="error.main">
                {stats.error}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <DataGrid
        rows={proxies}
        columns={columns}
        loading={loading}
        autoHeight
        pageSizeOptions={[10, 25, 50]}
        initialState={{
          pagination: { paginationModel: { pageSize: 10 } },
        }}
        disableRowSelectionOnClick
      />

      {/* Deregister Confirmation Dialog */}
      <Dialog open={deleteConfirm !== null} onClose={() => setDeleteConfirm(null)}>
        <DialogTitle>Confirm Deregister</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to deregister this proxy?
            It can re-register automatically on next heartbeat.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteConfirm(null)}>Cancel</Button>
          <Button
            onClick={() => deleteConfirm && handleDeregister(deleteConfirm)}
            variant="contained"
            color="error"
          >
            Deregister
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Proxies;

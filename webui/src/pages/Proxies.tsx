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
  IconButton,
  Divider,
  CircularProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableRow,
  Paper,
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
  Visibility as ViewIcon,
  Close as CloseIcon,
  Warning as WarningIcon,
  Speed as SpeedIcon,
  Memory as MemoryIcon,
  Computer as ComputerIcon,
  NetworkCheck as NetworkIcon,
} from '@mui/icons-material';
import { proxyApi, ProxyListParams, ProxyMetrics } from '@services/proxyApi';
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
  const [selectedProxy, setSelectedProxy] = useState<Proxy | null>(null);
  const [proxyMetrics, setProxyMetrics] = useState<ProxyMetrics | null>(null);
  const [metricsLoading, setMetricsLoading] = useState(false);
  const [proxyMetricsMap, setProxyMetricsMap] = useState<Map<number, ProxyMetrics>>(new Map());

  useEffect(() => {
    fetchClusters();
    fetchProxies();

    // Set up real-time updates every 10 seconds
    const interval = setInterval(() => {
      fetchProxies();
      fetchAllMetrics();
    }, 10000);
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
      // Fetch metrics for all proxies
      fetchAllMetrics(response.items);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to load proxies');
    } finally {
      setLoading(false);
    }
  };

  const fetchAllMetrics = async (proxyList?: Proxy[]) => {
    const proxiesToFetch = proxyList || proxies;
    const metricsMap = new Map<number, ProxyMetrics>();

    // Fetch metrics for all active proxies in parallel
    await Promise.allSettled(
      proxiesToFetch
        .filter(p => p.status === 'active')
        .map(async (proxy) => {
          try {
            const metrics = await proxyApi.getMetrics(proxy.id);
            metricsMap.set(proxy.id, metrics);
          } catch (err) {
            // Silently fail for individual metrics
            console.warn(`Failed to fetch metrics for proxy ${proxy.id}`);
          }
        })
    );

    setProxyMetricsMap(metricsMap);
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

  const handleViewDetails = async (proxy: Proxy) => {
    setSelectedProxy(proxy);
    setMetricsLoading(true);
    try {
      const metrics = await proxyApi.getMetrics(proxy.id);
      setProxyMetrics(metrics);
    } catch (err: any) {
      setError(err.response?.data?.detail || 'Failed to load proxy metrics');
      setProxyMetrics(null);
    } finally {
      setMetricsLoading(false);
    }
  };

  const handleCloseDetails = () => {
    setSelectedProxy(null);
    setProxyMetrics(null);
  };

  const getClusterName = (clusterId: number): string => {
    const cluster = clusters.find(c => c.id === clusterId);
    return cluster?.name || `Cluster ${clusterId}`;
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  const formatUptime = (seconds: number): string => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);

    if (days > 0) return `${days}d ${hours}h`;
    if (hours > 0) return `${hours}h ${minutes}m`;
    return `${minutes}m`;
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

  const getHealthIndicator = (proxyId: number, status: string) => {
    if (status !== 'active') {
      return <OfflineIcon sx={{ color: 'text.disabled' }} />;
    }

    const metrics = proxyMetricsMap.get(proxyId);
    if (!metrics) {
      return <OnlineIcon sx={{ color: 'success.main' }} />;
    }

    // Determine health based on CPU and memory usage
    const cpuHigh = metrics.cpu_usage > 80;
    const memoryHigh = metrics.memory_usage > 80;
    const hasErrors = metrics.errors > 0;

    if (cpuHigh || memoryHigh || hasErrors) {
      return <WarningIcon sx={{ color: 'warning.main' }} />;
    }

    return <OnlineIcon sx={{ color: 'success.main' }} />;
  };

  const columns: GridColDef[] = [
    { field: 'id', headerName: 'ID', width: 70 },
    {
      field: 'hostname',
      headerName: 'Hostname',
      width: 180,
      flex: 1,
    },
    {
      field: 'cluster_id',
      headerName: 'Cluster',
      width: 130,
      renderCell: (params) => (
        <Chip
          label={getClusterName(params.value as number)}
          size="small"
          variant="outlined"
        />
      ),
    },
    {
      field: 'status',
      headerName: 'Status',
      width: 120,
      renderCell: (params) => {
        const healthIcon = getHealthIndicator(params.row.id, params.value);
        return (
          <Chip
            icon={healthIcon || undefined}
            label={params.value}
            size="small"
            color={getStatusColor(params.value)}
            sx={{ textTransform: 'capitalize' }}
          />
        );
      },
    },
    {
      field: 'cpu_usage',
      headerName: 'CPU',
      width: 90,
      renderCell: (params) => {
        const metrics = proxyMetricsMap.get(params.row.id);
        if (!metrics || params.row.status !== 'active') return '-';
        const cpu = metrics.cpu_usage;
        const color = cpu > 80 ? 'error.main' : cpu > 60 ? 'warning.main' : 'success.main';
        return (
          <Typography variant="body2" sx={{ color }}>
            {cpu.toFixed(1)}%
          </Typography>
        );
      },
    },
    {
      field: 'memory_usage',
      headerName: 'Memory',
      width: 90,
      renderCell: (params) => {
        const metrics = proxyMetricsMap.get(params.row.id);
        if (!metrics || params.row.status !== 'active') return '-';
        const memory = metrics.memory_usage;
        const color = memory > 80 ? 'error.main' : memory > 60 ? 'warning.main' : 'success.main';
        return (
          <Typography variant="body2" sx={{ color }}>
            {memory.toFixed(1)}%
          </Typography>
        );
      },
    },
    {
      field: 'connections',
      headerName: 'Connections',
      width: 110,
      renderCell: (params) => {
        const metrics = proxyMetricsMap.get(params.row.id);
        if (!metrics || params.row.status !== 'active') return '-';
        return (
          <Typography variant="body2">
            {metrics.connections_active.toLocaleString()}
          </Typography>
        );
      },
    },
    {
      field: 'uptime',
      headerName: 'Uptime',
      width: 100,
      renderCell: (params) => {
        const metrics = proxyMetricsMap.get(params.row.id);
        if (!metrics || params.row.status !== 'active') return '-';
        return (
          <Typography variant="body2">
            {formatUptime(metrics.uptime)}
          </Typography>
        );
      },
    },
    {
      field: 'last_heartbeat',
      headerName: 'Last Heartbeat',
      width: 140,
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
      field: 'actions',
      type: 'actions',
      headerName: 'Actions',
      width: 100,
      getActions: (params: GridRowParams<Proxy>) => [
        <GridActionsCellItem
          icon={<ViewIcon />}
          label="View Details"
          onClick={() => handleViewDetails(params.row)}
        />,
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

      {/* Proxy Details Dialog */}
      <Dialog
        open={selectedProxy !== null}
        onClose={handleCloseDetails}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <ComputerIcon sx={{ fontSize: 32 }} />
              <Box>
                <Typography variant="h6">{selectedProxy?.hostname}</Typography>
                <Typography variant="body2" color="text.secondary">
                  {selectedProxy?.ip_address}
                </Typography>
              </Box>
            </Box>
            <IconButton onClick={handleCloseDetails}>
              <CloseIcon />
            </IconButton>
          </Box>
        </DialogTitle>
        <DialogContent dividers>
          {metricsLoading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
              <CircularProgress />
            </Box>
          ) : (
            <Box>
              {/* Proxy Information */}
              <Typography variant="h6" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <NetworkIcon /> Proxy Information
              </Typography>
              <TableContainer component={Paper} variant="outlined" sx={{ mb: 3 }}>
                <Table size="small">
                  <TableBody>
                    <TableRow>
                      <TableCell sx={{ fontWeight: 'bold' }}>ID</TableCell>
                      <TableCell>{selectedProxy?.id}</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell sx={{ fontWeight: 'bold' }}>Cluster</TableCell>
                      <TableCell>
                        <Chip
                          label={selectedProxy ? getClusterName(selectedProxy.cluster_id) : ''}
                          size="small"
                          variant="outlined"
                        />
                      </TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell sx={{ fontWeight: 'bold' }}>Status</TableCell>
                      <TableCell>
                        <Chip
                          icon={selectedProxy ? getStatusIcon(selectedProxy.status) || undefined : undefined}
                          label={selectedProxy?.status}
                          size="small"
                          color={selectedProxy ? getStatusColor(selectedProxy.status) : 'default'}
                          sx={{ textTransform: 'capitalize' }}
                        />
                      </TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell sx={{ fontWeight: 'bold' }}>Version</TableCell>
                      <TableCell>{selectedProxy?.version}</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell sx={{ fontWeight: 'bold' }}>Last Heartbeat</TableCell>
                      <TableCell>
                        {selectedProxy?.last_heartbeat
                          ? formatDistanceToNow(new Date(selectedProxy.last_heartbeat), { addSuffix: true })
                          : 'Never'}
                      </TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell sx={{ fontWeight: 'bold' }}>Capabilities</TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
                          {selectedProxy?.capabilities && selectedProxy.capabilities.length > 0 ? (
                            selectedProxy.capabilities.map((cap: string) => (
                              <Chip key={cap} label={cap} size="small" variant="outlined" />
                            ))
                          ) : (
                            'None'
                          )}
                        </Box>
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </TableContainer>

              {/* Metrics Summary */}
              {proxyMetrics && (
                <>
                  <Typography variant="h6" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <SpeedIcon /> Performance Metrics
                  </Typography>
                  <Grid container spacing={2} sx={{ mb: 3 }}>
                    <Grid item xs={12} sm={6}>
                      <Card variant="outlined">
                        <CardContent>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                            <ComputerIcon color="primary" />
                            <Typography color="text.secondary" variant="body2">
                              CPU Usage
                            </Typography>
                          </Box>
                          <Typography variant="h5">
                            {proxyMetrics.cpu_usage.toFixed(1)}%
                          </Typography>
                        </CardContent>
                      </Card>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <Card variant="outlined">
                        <CardContent>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                            <MemoryIcon color="secondary" />
                            <Typography color="text.secondary" variant="body2">
                              Memory Usage
                            </Typography>
                          </Box>
                          <Typography variant="h5">
                            {proxyMetrics.memory_usage.toFixed(1)}%
                          </Typography>
                        </CardContent>
                      </Card>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <Card variant="outlined">
                        <CardContent>
                          <Typography color="text.secondary" variant="body2" gutterBottom>
                            Active Connections
                          </Typography>
                          <Typography variant="h5">
                            {proxyMetrics.connections_active.toLocaleString()}
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            Total: {proxyMetrics.connections_total.toLocaleString()}
                          </Typography>
                        </CardContent>
                      </Card>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <Card variant="outlined">
                        <CardContent>
                          <Typography color="text.secondary" variant="body2" gutterBottom>
                            Uptime
                          </Typography>
                          <Typography variant="h5">
                            {formatUptime(proxyMetrics.uptime)}
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            Errors: {proxyMetrics.errors}
                          </Typography>
                        </CardContent>
                      </Card>
                    </Grid>
                  </Grid>

                  <Typography variant="h6" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <NetworkIcon /> Network Throughput
                  </Typography>
                  <TableContainer component={Paper} variant="outlined">
                    <Table size="small">
                      <TableBody>
                        <TableRow>
                          <TableCell sx={{ fontWeight: 'bold' }}>Bytes Sent</TableCell>
                          <TableCell>{formatBytes(proxyMetrics.bytes_sent)}</TableCell>
                        </TableRow>
                        <TableRow>
                          <TableCell sx={{ fontWeight: 'bold' }}>Bytes Received</TableCell>
                          <TableCell>{formatBytes(proxyMetrics.bytes_received)}</TableCell>
                        </TableRow>
                        <TableRow>
                          <TableCell sx={{ fontWeight: 'bold' }}>Packets Sent</TableCell>
                          <TableCell>{proxyMetrics.packets_sent.toLocaleString()}</TableCell>
                        </TableRow>
                        <TableRow>
                          <TableCell sx={{ fontWeight: 'bold' }}>Packets Received</TableCell>
                          <TableCell>{proxyMetrics.packets_received.toLocaleString()}</TableCell>
                        </TableRow>
                      </TableBody>
                    </Table>
                  </TableContainer>
                </>
              )}
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDetails}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Proxies;

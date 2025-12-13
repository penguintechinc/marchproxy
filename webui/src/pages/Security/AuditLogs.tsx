/**
 * Audit Log Viewer
 *
 * Displays immutable audit logs with advanced filtering, export,
 * real-time streaming, and tamper detection.
 */

import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  TextField,
  Grid,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Alert,
  Snackbar,
  Chip,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Card,
  CardContent,
  Switch,
  FormControlLabel
} from '@mui/material';
import { DataGrid, GridColDef, GridPaginationModel } from '@mui/x-data-grid';
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import {
  Download as ExportIcon,
  Refresh as RefreshIcon,
  Security as SecurityIcon,
  Warning as WarningIcon,
  Search as SearchIcon,
  Visibility as ViewIcon
} from '@mui/icons-material';
import {
  AuditLog,
  AuditLogFilter,
  getAuditLogs,
  exportAuditLogs,
  verifyAuditLogIntegrity
} from '../../services/securityApi';
import LicenseGate from '../../components/Common/LicenseGate';

const AuditLogs: React.FC = () => {
  const hasEnterpriseAccess = true; // TODO: Get from license check
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [totalLogs, setTotalLogs] = useState(0);
  const [paginationModel, setPaginationModel] = useState<GridPaginationModel>({
    page: 0,
    pageSize: 25
  });
  const [filters, setFilters] = useState<AuditLogFilter>({
    page: 0,
    page_size: 25
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null);
  const [detailsDialogOpen, setDetailsDialogOpen] = useState(false);
  const [exportDialogOpen, setExportDialogOpen] = useState(false);
  const [exportFormat, setExportFormat] = useState<'csv' | 'json'>('csv');
  const [realTimeEnabled, setRealTimeEnabled] = useState(false);
  const [integrityStatus, setIntegrityStatus] = useState<{
    verified: boolean;
    total_logs: number;
    tampered_logs: number[];
    last_verified: string;
  } | null>(null);

  const columns: GridColDef[] = [
    {
      field: 'timestamp',
      headerName: 'Timestamp',
      width: 180,
      valueFormatter: (params) => new Date(params).toLocaleString()
    },
    {
      field: 'username',
      headerName: 'User',
      width: 120
    },
    {
      field: 'action',
      headerName: 'Action',
      width: 150
    },
    {
      field: 'resource_type',
      headerName: 'Resource Type',
      width: 130
    },
    {
      field: 'resource_id',
      headerName: 'Resource ID',
      width: 120
    },
    {
      field: 'status',
      headerName: 'Status',
      width: 100,
      renderCell: (params) => (
        <Chip
          label={params.value}
          color={params.value === 'success' ? 'success' : 'error'}
          size="small"
        />
      )
    },
    {
      field: 'ip_address',
      headerName: 'IP Address',
      width: 130
    },
    {
      field: 'tamper_detected',
      headerName: 'Tamper',
      width: 80,
      renderCell: (params) =>
        params.value ? (
          <Tooltip title="Tamper detected!">
            <WarningIcon color="error" />
          </Tooltip>
        ) : null
    },
    {
      field: 'actions',
      headerName: 'Actions',
      width: 100,
      sortable: false,
      renderCell: (params) => (
        <IconButton
          size="small"
          onClick={() => handleViewDetails(params.row as AuditLog)}
        >
          <ViewIcon fontSize="small" />
        </IconButton>
      )
    }
  ];

  useEffect(() => {
    loadAuditLogs();
  }, [paginationModel, filters]);

  useEffect(() => {
    let interval: NodeJS.Timeout | null = null;

    if (realTimeEnabled) {
      interval = setInterval(() => {
        loadAuditLogs(true);
      }, 5000); // Refresh every 5 seconds
    }

    return () => {
      if (interval) clearInterval(interval);
    };
  }, [realTimeEnabled, filters]);

  const loadAuditLogs = async (silent = false) => {
    try {
      if (!silent) setLoading(true);

      const response = await getAuditLogs({
        ...filters,
        page: paginationModel.page,
        page_size: paginationModel.pageSize
      });

      setLogs(response.items);
      setTotalLogs(response.total);
    } catch (err: any) {
      if (!silent) setError(err.message || 'Failed to load audit logs');
    } finally {
      if (!silent) setLoading(false);
    }
  };

  const handleViewDetails = (log: AuditLog) => {
    setSelectedLog(log);
    setDetailsDialogOpen(true);
  };

  const handleExport = async () => {
    try {
      setLoading(true);
      const blob = await exportAuditLogs({
        format: exportFormat,
        filters
      });

      // Create download link
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `audit-logs-${new Date().toISOString()}.${exportFormat}`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);

      setSuccess('Audit logs exported successfully');
      setExportDialogOpen(false);
    } catch (err: any) {
      setError(err.message || 'Failed to export logs');
    } finally {
      setLoading(false);
    }
  };

  const handleVerifyIntegrity = async () => {
    try {
      setLoading(true);
      const result = await verifyAuditLogIntegrity();
      setIntegrityStatus(result);

      if (result.verified) {
        setSuccess('Audit log integrity verified - no tampering detected');
      } else {
        setError(
          `Tamper detected! ${result.tampered_logs.length} log(s) have been compromised`
        );
      }
    } catch (err: any) {
      setError(err.message || 'Failed to verify integrity');
    } finally {
      setLoading(false);
    }
  };

  const handleFilterChange = (key: keyof AuditLogFilter, value: any) => {
    setFilters({ ...filters, [key]: value });
    setPaginationModel({ ...paginationModel, page: 0 });
  };

  const handleClearFilters = () => {
    setFilters({ page: 0, page_size: 25 });
    setPaginationModel({ page: 0, pageSize: 25 });
  };

  return (
    <LicenseGate
      featureName="Zero-Trust Security"
      hasAccess={hasEnterpriseAccess}
      isLoading={false}
    >
      <Box sx={{ p: 3 }}>
        <Box sx={{ mb: 3, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography variant="h4">Audit Logs</Typography>
          <Box>
            <FormControlLabel
              control={
                <Switch
                  checked={realTimeEnabled}
                  onChange={(e) => setRealTimeEnabled(e.target.checked)}
                />
              }
              label="Real-time"
              sx={{ mr: 2 }}
            />
            <Button
              variant="outlined"
              startIcon={<SecurityIcon />}
              onClick={handleVerifyIntegrity}
              disabled={loading}
              sx={{ mr: 1 }}
            >
              Verify Integrity
            </Button>
            <Button
              variant="outlined"
              startIcon={<ExportIcon />}
              onClick={() => setExportDialogOpen(true)}
              disabled={loading}
              sx={{ mr: 1 }}
            >
              Export
            </Button>
            <IconButton onClick={() => loadAuditLogs()} disabled={loading}>
              <RefreshIcon />
            </IconButton>
          </Box>
        </Box>

        {/* Integrity Status Alert */}
        {integrityStatus && !integrityStatus.verified && (
          <Alert severity="error" sx={{ mb: 2 }}>
            <Typography variant="subtitle2">
              Tamper Detection Alert
            </Typography>
            <Typography variant="body2">
              {integrityStatus.tampered_logs.length} audit log(s) have failed
              integrity verification. Log IDs: {integrityStatus.tampered_logs.join(', ')}
            </Typography>
          </Alert>
        )}

        {integrityStatus && integrityStatus.verified && (
          <Alert severity="success" sx={{ mb: 2 }}>
            Audit log integrity verified. All {integrityStatus.total_logs} logs
            are unmodified. Last verified:{' '}
            {new Date(integrityStatus.last_verified).toLocaleString()}
          </Alert>
        )}

        {/* Filters */}
        <Paper sx={{ p: 2, mb: 3 }}>
          <Typography variant="h6" gutterBottom>
            Filters
          </Typography>
          <Grid container spacing={2}>
            <Grid item xs={12} md={3}>
              <TextField
                fullWidth
                label="Search"
                placeholder="User, action, resource..."
                value={filters.search || ''}
                onChange={(e) => handleFilterChange('search', e.target.value)}
                InputProps={{
                  startAdornment: <SearchIcon sx={{ mr: 1, color: 'action.disabled' }} />
                }}
              />
            </Grid>
            <Grid item xs={12} md={2}>
              <FormControl fullWidth>
                <InputLabel>Status</InputLabel>
                <Select
                  value={filters.status || ''}
                  onChange={(e) => handleFilterChange('status', e.target.value || undefined)}
                >
                  <MenuItem value="">All</MenuItem>
                  <MenuItem value="success">Success</MenuItem>
                  <MenuItem value="failure">Failure</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12} md={2}>
              <TextField
                fullWidth
                label="Action"
                value={filters.action || ''}
                onChange={(e) => handleFilterChange('action', e.target.value || undefined)}
              />
            </Grid>
            <Grid item xs={12} md={2}>
              <LocalizationProvider dateAdapter={AdapterDateFns}>
                <DateTimePicker
                  label="Start Date"
                  value={filters.start_date ? new Date(filters.start_date) : null}
                  onChange={(date) =>
                    handleFilterChange('start_date', date?.toISOString())
                  }
                  slotProps={{ textField: { fullWidth: true } }}
                />
              </LocalizationProvider>
            </Grid>
            <Grid item xs={12} md={2}>
              <LocalizationProvider dateAdapter={AdapterDateFns}>
                <DateTimePicker
                  label="End Date"
                  value={filters.end_date ? new Date(filters.end_date) : null}
                  onChange={(date) =>
                    handleFilterChange('end_date', date?.toISOString())
                  }
                  slotProps={{ textField: { fullWidth: true } }}
                />
              </LocalizationProvider>
            </Grid>
            <Grid item xs={12} md={1}>
              <Button
                fullWidth
                variant="outlined"
                onClick={handleClearFilters}
                sx={{ height: '56px' }}
              >
                Clear
              </Button>
            </Grid>
          </Grid>
        </Paper>

        {/* Data Grid */}
        <Paper sx={{ height: 600, width: '100%' }}>
          <DataGrid
            rows={logs}
            columns={columns}
            loading={loading}
            paginationModel={paginationModel}
            onPaginationModelChange={setPaginationModel}
            pageSizeOptions={[10, 25, 50, 100]}
            rowCount={totalLogs}
            paginationMode="server"
            disableRowSelectionOnClick
          />
        </Paper>

        {/* Details Dialog */}
        <Dialog
          open={detailsDialogOpen}
          onClose={() => setDetailsDialogOpen(false)}
          maxWidth="md"
          fullWidth
        >
          <DialogTitle>Audit Log Details</DialogTitle>
          <DialogContent>
            {selectedLog && (
              <Grid container spacing={2}>
                <Grid item xs={12}>
                  {selectedLog.tamper_detected && (
                    <Alert severity="error" sx={{ mb: 2 }}>
                      <Typography variant="subtitle2">
                        TAMPER DETECTED
                      </Typography>
                      This log entry has failed integrity verification.
                    </Alert>
                  )}
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="subtitle2" color="textSecondary">
                    Timestamp
                  </Typography>
                  <Typography>
                    {new Date(selectedLog.timestamp).toLocaleString()}
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="subtitle2" color="textSecondary">
                    User
                  </Typography>
                  <Typography>{selectedLog.username}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="subtitle2" color="textSecondary">
                    Action
                  </Typography>
                  <Typography>{selectedLog.action}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="subtitle2" color="textSecondary">
                    Status
                  </Typography>
                  <Chip
                    label={selectedLog.status}
                    color={selectedLog.status === 'success' ? 'success' : 'error'}
                    size="small"
                  />
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="subtitle2" color="textSecondary">
                    Resource Type
                  </Typography>
                  <Typography>{selectedLog.resource_type}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="subtitle2" color="textSecondary">
                    Resource ID
                  </Typography>
                  <Typography>{selectedLog.resource_id || 'N/A'}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="subtitle2" color="textSecondary">
                    IP Address
                  </Typography>
                  <Typography>{selectedLog.ip_address}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="subtitle2" color="textSecondary">
                    User Agent
                  </Typography>
                  <Typography variant="caption">
                    {selectedLog.user_agent || 'N/A'}
                  </Typography>
                </Grid>
                <Grid item xs={12}>
                  <Typography variant="subtitle2" color="textSecondary" gutterBottom>
                    Details
                  </Typography>
                  <Card variant="outlined">
                    <CardContent>
                      <pre style={{ margin: 0, fontSize: '0.85rem', overflow: 'auto' }}>
                        {JSON.stringify(selectedLog.details, null, 2)}
                      </pre>
                    </CardContent>
                  </Card>
                </Grid>
                <Grid item xs={12}>
                  <Typography variant="subtitle2" color="textSecondary">
                    Integrity Hash
                  </Typography>
                  <Typography
                    variant="caption"
                    sx={{ fontFamily: 'monospace', wordBreak: 'break-all' }}
                  >
                    {selectedLog.hash}
                  </Typography>
                </Grid>
                {selectedLog.previous_hash && (
                  <Grid item xs={12}>
                    <Typography variant="subtitle2" color="textSecondary">
                      Previous Hash
                    </Typography>
                    <Typography
                      variant="caption"
                      sx={{ fontFamily: 'monospace', wordBreak: 'break-all' }}
                    >
                      {selectedLog.previous_hash}
                    </Typography>
                  </Grid>
                )}
              </Grid>
            )}
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setDetailsDialogOpen(false)}>Close</Button>
          </DialogActions>
        </Dialog>

        {/* Export Dialog */}
        <Dialog open={exportDialogOpen} onClose={() => setExportDialogOpen(false)}>
          <DialogTitle>Export Audit Logs</DialogTitle>
          <DialogContent>
            <Typography variant="body2" paragraph>
              Export audit logs with current filters applied.
            </Typography>
            <FormControl fullWidth sx={{ mt: 2 }}>
              <InputLabel>Format</InputLabel>
              <Select
                value={exportFormat}
                onChange={(e) => setExportFormat(e.target.value as 'csv' | 'json')}
              >
                <MenuItem value="csv">CSV</MenuItem>
                <MenuItem value="json">JSON</MenuItem>
              </Select>
            </FormControl>
            <Alert severity="info" sx={{ mt: 2 }}>
              Total logs to export: {totalLogs}
            </Alert>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setExportDialogOpen(false)}>Cancel</Button>
            <Button
              variant="contained"
              onClick={handleExport}
              disabled={loading}
            >
              Export
            </Button>
          </DialogActions>
        </Dialog>

        {/* Snackbar Messages */}
        <Snackbar
          open={!!error}
          autoHideDuration={6000}
          onClose={() => setError(null)}
        >
          <Alert severity="error" onClose={() => setError(null)}>
            {error}
          </Alert>
        </Snackbar>
        <Snackbar
          open={!!success}
          autoHideDuration={4000}
          onClose={() => setSuccess(null)}
        >
          <Alert severity="success" onClose={() => setSuccess(null)}>
            {success}
          </Alert>
        </Snackbar>
      </Box>
    </LicenseGate>
  );
};

export default AuditLogs;

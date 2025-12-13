import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  TextField,
  Alert,
  Stack,
  Typography,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  IconButton,
  Tooltip,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
} from '@mui/material';
import {
  Search,
  Download,
  Visibility,
  CheckCircle,
  Cancel,
  Shield,
} from '@mui/icons-material';
import { DateTimePicker } from '@mui/x-date-pickers';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { apiClient } from '../../services/api';

interface AuditEvent {
  event_id: number;
  timestamp: string;
  event_type: string;
  service?: string;
  user?: string;
  action: string;
  resource: string;
  source_ip: string;
  allowed: boolean;
  reason?: string;
  policy_name?: string;
  duration?: number;
  prev_hash: string;
  current_hash: string;
}

const AuditLogViewer: React.FC = () => {
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [totalCount, setTotalCount] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Filters
  const [startDate, setStartDate] = useState<Date | null>(new Date(Date.now() - 24 * 60 * 60 * 1000));
  const [endDate, setEndDate] = useState<Date | null>(new Date());
  const [filterService, setFilterService] = useState<string>('');
  const [filterUser, setFilterUser] = useState<string>('');
  const [filterAction, setFilterAction] = useState<string>('');
  const [filterAllowed, setFilterAllowed] = useState<string>('all');

  // Detail dialog
  const [selectedEvent, setSelectedEvent] = useState<AuditEvent | null>(null);
  const [detailDialog, setDetailDialog] = useState(false);

  // Chain verification
  const [chainStatus, setChainStatus] = useState<{ valid: boolean; message: string } | null>(null);

  useEffect(() => {
    fetchEvents();
  }, [page, rowsPerPage]);

  const fetchEvents = async () => {
    try {
      setLoading(true);
      setError(null);

      const params = new URLSearchParams({
        offset: (page * rowsPerPage).toString(),
        limit: rowsPerPage.toString(),
      });

      if (startDate) {
        params.append('start_time', startDate.toISOString());
      }
      if (endDate) {
        params.append('end_time', endDate.toISOString());
      }
      if (filterService) {
        params.append('service', filterService);
      }
      if (filterUser) {
        params.append('user', filterUser);
      }
      if (filterAction) {
        params.append('action', filterAction);
      }
      if (filterAllowed !== 'all') {
        params.append('allowed', filterAllowed);
      }

      const response = await apiClient.get(`/api/v1/zerotrust/audit-logs?${params}`);
      setEvents(response.data.events || []);
      setTotalCount(response.data.total || 0);
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to fetch audit logs');
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = () => {
    setPage(0);
    fetchEvents();
  };

  const handleExport = async (format: 'json' | 'csv') => {
    try {
      setLoading(true);
      setError(null);

      const params = new URLSearchParams();
      if (startDate) params.append('start_time', startDate.toISOString());
      if (endDate) params.append('end_time', endDate.toISOString());
      params.append('format', format);

      const response = await apiClient.get(`/api/v1/zerotrust/audit-logs/export?${params}`, {
        responseType: 'blob',
      });

      // Create download link
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', `audit-logs-${Date.now()}.${format}`);
      document.body.appendChild(link);
      link.click();
      link.remove();

      setSuccess(`Audit logs exported as ${format.toUpperCase()}`);
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to export audit logs');
    } finally {
      setLoading(false);
    }
  };

  const handleVerifyChain = async () => {
    try {
      setLoading(true);
      setError(null);
      setSuccess(null);

      const response = await apiClient.post('/api/v1/zerotrust/audit-logs/verify');
      setChainStatus(response.data);

      if (response.data.valid) {
        setSuccess('Audit chain integrity verified successfully');
      } else {
        setError(`Audit chain integrity check failed: ${response.data.message}`);
      }
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to verify audit chain');
    } finally {
      setLoading(false);
    }
  };

  const handleViewDetails = (event: AuditEvent) => {
    setSelectedEvent(event);
    setDetailDialog(true);
  };

  return (
    <LocalizationProvider dateAdapter={AdapterDateFns}>
      <Box>
        {/* Filters */}
        <Card sx={{ mb: 2 }}>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              Filters
            </Typography>

            <Grid container spacing={2}>
              <Grid item xs={12} md={3}>
                <DateTimePicker
                  label="Start Date"
                  value={startDate}
                  onChange={setStartDate}
                  renderInput={(params) => <TextField {...params} fullWidth />}
                />
              </Grid>

              <Grid item xs={12} md={3}>
                <DateTimePicker
                  label="End Date"
                  value={endDate}
                  onChange={setEndDate}
                  renderInput={(params) => <TextField {...params} fullWidth />}
                />
              </Grid>

              <Grid item xs={12} md={2}>
                <TextField
                  label="Service"
                  value={filterService}
                  onChange={(e) => setFilterService(e.target.value)}
                  fullWidth
                />
              </Grid>

              <Grid item xs={12} md={2}>
                <TextField
                  label="User"
                  value={filterUser}
                  onChange={(e) => setFilterUser(e.target.value)}
                  fullWidth
                />
              </Grid>

              <Grid item xs={12} md={2}>
                <FormControl fullWidth>
                  <InputLabel>Access Result</InputLabel>
                  <Select
                    value={filterAllowed}
                    onChange={(e) => setFilterAllowed(e.target.value)}
                    label="Access Result"
                  >
                    <MenuItem value="all">All</MenuItem>
                    <MenuItem value="true">Allowed</MenuItem>
                    <MenuItem value="false">Denied</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
            </Grid>

            <Stack direction="row" spacing={2} sx={{ mt: 2 }}>
              <Button
                variant="contained"
                startIcon={<Search />}
                onClick={handleSearch}
                disabled={loading}
              >
                Search
              </Button>
              <Button
                startIcon={<Download />}
                onClick={() => handleExport('json')}
                disabled={loading}
              >
                Export JSON
              </Button>
              <Button
                startIcon={<Download />}
                onClick={() => handleExport('csv')}
                disabled={loading}
              >
                Export CSV
              </Button>
              <Button
                startIcon={<Shield />}
                onClick={handleVerifyChain}
                disabled={loading}
                color="secondary"
              >
                Verify Chain
              </Button>
            </Stack>
          </CardContent>
        </Card>

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

        {/* Audit Log Table */}
        <Card>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Timestamp</TableCell>
                  <TableCell>Event Type</TableCell>
                  <TableCell>Service / User</TableCell>
                  <TableCell>Action</TableCell>
                  <TableCell>Resource</TableCell>
                  <TableCell>Source IP</TableCell>
                  <TableCell>Result</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {events.map((event) => (
                  <TableRow key={event.event_id}>
                    <TableCell>
                      {new Date(event.timestamp).toLocaleString()}
                    </TableCell>
                    <TableCell>
                      <Chip label={event.event_type} size="small" />
                    </TableCell>
                    <TableCell>
                      {event.service || event.user || '-'}
                    </TableCell>
                    <TableCell>{event.action}</TableCell>
                    <TableCell>{event.resource}</TableCell>
                    <TableCell>{event.source_ip}</TableCell>
                    <TableCell>
                      {event.allowed ? (
                        <Chip
                          icon={<CheckCircle />}
                          label="Allowed"
                          color="success"
                          size="small"
                        />
                      ) : (
                        <Chip
                          icon={<Cancel />}
                          label="Denied"
                          color="error"
                          size="small"
                        />
                      )}
                    </TableCell>
                    <TableCell>
                      <Tooltip title="View Details">
                        <IconButton
                          size="small"
                          onClick={() => handleViewDetails(event)}
                        >
                          <Visibility />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>

          <TablePagination
            component="div"
            count={totalCount}
            page={page}
            onPageChange={(e, newPage) => setPage(newPage)}
            rowsPerPage={rowsPerPage}
            onRowsPerPageChange={(e) => setRowsPerPage(parseInt(e.target.value, 10))}
            rowsPerPageOptions={[10, 25, 50, 100]}
          />
        </Card>

        {/* Event Detail Dialog */}
        <Dialog open={detailDialog} onClose={() => setDetailDialog(false)} maxWidth="md" fullWidth>
          <DialogTitle>Audit Event Details</DialogTitle>
          <DialogContent>
            {selectedEvent && (
              <Box sx={{ fontFamily: 'monospace', fontSize: '0.875rem' }}>
                <pre>{JSON.stringify(selectedEvent, null, 2)}</pre>
              </Box>
            )}
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setDetailDialog(false)}>Close</Button>
          </DialogActions>
        </Dialog>
      </Box>
    </LocalizationProvider>
  );
};

export default AuditLogViewer;

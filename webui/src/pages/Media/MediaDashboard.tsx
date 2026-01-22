/**
 * Media Dashboard Page
 *
 * Main page for managing media streaming module including
 * active streams, configuration, and statistics.
 */

import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Grid,
  Card,
  CardContent,
  Tabs,
  Tab,
  Alert,
  CircularProgress,
  Chip,
  IconButton,
  Tooltip,
  Button,
  Paper,
} from '@mui/material';
import { DataGrid, GridColDef, GridActionsCellItem } from '@mui/x-data-grid';
import RefreshIcon from '@mui/icons-material/Refresh';
import StopIcon from '@mui/icons-material/Stop';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import SettingsIcon from '@mui/icons-material/Settings';
import InfoIcon from '@mui/icons-material/Info';
import VideocamIcon from '@mui/icons-material/Videocam';
import CloudUploadIcon from '@mui/icons-material/CloudUpload';
import SpeedIcon from '@mui/icons-material/Speed';
import {
  getActiveStreams,
  getMediaStats,
  getCapabilities,
  stopStream,
  MediaStream,
  MediaStats,
  MediaCapabilities,
  getProtocolLabel,
  getCodecLabel,
  formatBytes,
  formatBitrate,
  getResolutionLabel,
} from '@services/mediaApi';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

const TabPanel: React.FC<TabPanelProps> = ({ children, value, index }) => (
  <div role="tabpanel" hidden={value !== index}>
    {value === index && <Box sx={{ py: 3 }}>{children}</Box>}
  </div>
);

const MediaDashboard: React.FC = () => {
  const [tabValue, setTabValue] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [streams, setStreams] = useState<MediaStream[]>([]);
  const [stats, setStats] = useState<MediaStats | null>(null);
  const [capabilities, setCapabilities] = useState<MediaCapabilities | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  useEffect(() => {
    fetchData();
    // Refresh every 10 seconds
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, []);

  const fetchData = async () => {
    try {
      setRefreshing(true);
      const [streamsData, statsData, capsData] = await Promise.all([
        getActiveStreams(),
        getMediaStats(),
        getCapabilities(),
      ]);
      setStreams(streamsData);
      setStats(statsData);
      setCapabilities(capsData);
      setError(null);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to load media data';
      setError(message);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const handleStopStream = async (streamKey: string) => {
    if (!confirm(`Stop stream ${streamKey}?`)) return;
    try {
      await stopStream(streamKey);
      await fetchData();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to stop stream';
      setError(message);
    }
  };

  const streamColumns: GridColDef[] = [
    { field: 'stream_key', headerName: 'Stream Key', flex: 1 },
    {
      field: 'protocol',
      headerName: 'Protocol',
      width: 100,
      renderCell: (params) => (
        <Chip
          label={getProtocolLabel(params.value)}
          size="small"
          color={params.value === 'rtmp' ? 'primary' : params.value === 'srt' ? 'secondary' : 'info'}
        />
      ),
    },
    {
      field: 'codec',
      headerName: 'Codec',
      width: 100,
      renderCell: (params) => params.value ? getCodecLabel(params.value) : '-',
    },
    { field: 'resolution', headerName: 'Resolution', width: 120 },
    {
      field: 'bitrate_kbps',
      headerName: 'Bitrate',
      width: 100,
      renderCell: (params) => params.value ? formatBitrate(params.value) : '-',
    },
    {
      field: 'status',
      headerName: 'Status',
      width: 100,
      renderCell: (params) => (
        <Chip
          label={params.value}
          size="small"
          color={params.value === 'active' ? 'success' : params.value === 'error' ? 'error' : 'default'}
        />
      ),
    },
    { field: 'client_ip', headerName: 'Client IP', width: 130 },
    {
      field: 'bytes_in',
      headerName: 'Data In',
      width: 100,
      renderCell: (params) => formatBytes(params.value),
    },
    {
      field: 'actions',
      type: 'actions',
      headerName: 'Actions',
      width: 80,
      getActions: (params) => [
        <GridActionsCellItem
          key="stop"
          icon={<StopIcon color="error" />}
          label="Stop"
          onClick={() => handleStopStream(params.row.stream_key)}
        />,
      ],
    },
  ];

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4" fontWeight="bold">
          Media Streaming
        </Typography>
        <Tooltip title="Refresh">
          <IconButton onClick={fetchData} disabled={refreshing}>
            <RefreshIcon className={refreshing ? 'spin' : ''} />
          </IconButton>
        </Tooltip>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* Stats Overview */}
      <Grid container spacing={3} mb={3}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" gap={1}>
                <VideocamIcon color="primary" />
                <Typography variant="subtitle2" color="text.secondary">
                  Active Streams
                </Typography>
              </Box>
              <Typography variant="h4" mt={1}>
                {stats?.active_streams ?? 0}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" gap={1}>
                <CloudUploadIcon color="primary" />
                <Typography variant="subtitle2" color="text.secondary">
                  Data Received
                </Typography>
              </Box>
              <Typography variant="h4" mt={1}>
                {formatBytes(stats?.total_bytes_in ?? 0)}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" gap={1}>
                <SpeedIcon color="primary" />
                <Typography variant="subtitle2" color="text.secondary">
                  Max Resolution
                </Typography>
              </Box>
              <Typography variant="h4" mt={1}>
                {capabilities ? getResolutionLabel(capabilities.effective_max_resolution) : '-'}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" gap={1}>
                <SettingsIcon color="primary" />
                <Typography variant="subtitle2" color="text.secondary">
                  GPU
                </Typography>
              </Box>
              <Typography variant="h5" mt={1}>
                {capabilities?.hardware.gpu_type !== 'none'
                  ? capabilities?.hardware.gpu_model?.split(' ').slice(-2).join(' ')
                  : 'CPU Only'}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Hardware Info Alert */}
      {capabilities && (
        <Alert severity="info" sx={{ mb: 3 }} icon={<InfoIcon />}>
          <strong>Hardware:</strong> {capabilities.hardware.gpu_type !== 'none' ? capabilities.hardware.gpu_model : 'CPU-only mode'} |{' '}
          <strong>VRAM:</strong> {capabilities.hardware.vram_gb}GB |{' '}
          <strong>AV1:</strong> {capabilities.hardware.av1_supported ? 'Supported' : 'Not Available'} |{' '}
          <strong>Max:</strong> {getResolutionLabel(capabilities.effective_max_resolution)}
          {capabilities.settings.admin_max_resolution && (
            <> (Admin limit: {getResolutionLabel(capabilities.settings.admin_max_resolution)})</>
          )}
        </Alert>
      )}

      {/* Tabs */}
      <Paper sx={{ mb: 3 }}>
        <Tabs value={tabValue} onChange={(_, v) => setTabValue(v)}>
          <Tab icon={<PlayArrowIcon />} label="Active Streams" />
          <Tab icon={<SettingsIcon />} label="Configuration" />
        </Tabs>
      </Paper>

      {/* Active Streams Tab */}
      <TabPanel value={tabValue} index={0}>
        <Card>
          <CardContent>
            <Typography variant="h6" mb={2}>
              Active Streams
            </Typography>
            <DataGrid
              rows={streams}
              columns={streamColumns}
              autoHeight
              pageSizeOptions={[10, 25, 50]}
              initialState={{
                pagination: { paginationModel: { pageSize: 10 } },
              }}
              disableRowSelectionOnClick
              sx={{
                '& .MuiDataGrid-cell': { borderBottom: '1px solid rgba(255,255,255,0.1)' },
              }}
            />
            {streams.length === 0 && (
              <Typography color="text.secondary" textAlign="center" py={4}>
                No active streams
              </Typography>
            )}
          </CardContent>
        </Card>
      </TabPanel>

      {/* Configuration Tab */}
      <TabPanel value={tabValue} index={1}>
        <Grid container spacing={3}>
          <Grid item xs={12} md={6}>
            <Card>
              <CardContent>
                <Typography variant="h6" mb={2}>
                  Transcode Ladder
                </Typography>
                <Typography variant="body2" color="text.secondary" mb={2}>
                  Pre-transcode all incoming streams to these resolutions for ABR playback.
                </Typography>
                <Box display="flex" flexWrap="wrap" gap={1}>
                  {capabilities?.settings.transcode_ladder_resolutions.map((res) => (
                    <Chip key={res} label={getResolutionLabel(res)} variant="outlined" />
                  ))}
                </Box>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={6}>
            <Card>
              <CardContent>
                <Typography variant="h6" mb={2}>
                  Protocol Support
                </Typography>
                <Box display="flex" flexDirection="column" gap={1}>
                  <Box display="flex" justifyContent="space-between">
                    <Typography>RTMP (Port 1935)</Typography>
                    <Chip label="Enabled" color="success" size="small" />
                  </Box>
                  <Box display="flex" justifyContent="space-between">
                    <Typography>SRT (Port 8890)</Typography>
                    <Chip label="Disabled" color="default" size="small" />
                  </Box>
                  <Box display="flex" justifyContent="space-between">
                    <Typography>WebRTC/WHIP (Port 8080)</Typography>
                    <Chip label="Disabled" color="default" size="small" />
                  </Box>
                </Box>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={6}>
            <Card>
              <CardContent>
                <Typography variant="h6" mb={2}>
                  Codec Support
                </Typography>
                <Box display="flex" flexDirection="column" gap={1}>
                  <Box display="flex" justifyContent="space-between">
                    <Typography>H.264</Typography>
                    <Chip
                      label={capabilities?.hardware.gpu_type !== 'none' ? 'Hardware' : 'CPU'}
                      color="success"
                      size="small"
                    />
                  </Box>
                  <Box display="flex" justifyContent="space-between">
                    <Typography>H.265/HEVC</Typography>
                    <Chip
                      label={capabilities?.hardware.gpu_type !== 'none' ? 'Hardware' : 'CPU'}
                      color="success"
                      size="small"
                    />
                  </Box>
                  <Box display="flex" justifyContent="space-between">
                    <Typography>AV1</Typography>
                    <Chip
                      label={
                        capabilities?.hardware.av1_supported
                          ? 'Hardware'
                          : capabilities?.hardware.gpu_type !== 'none'
                          ? 'CPU Only'
                          : 'CPU'
                      }
                      color={capabilities?.hardware.av1_supported ? 'success' : 'warning'}
                      size="small"
                    />
                  </Box>
                </Box>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={6}>
            <Card>
              <CardContent>
                <Typography variant="h6" mb={2}>
                  Output Formats
                </Typography>
                <Box display="flex" flexDirection="column" gap={1}>
                  <Box display="flex" justifyContent="space-between">
                    <Typography>HLS</Typography>
                    <Chip label="Enabled" color="success" size="small" />
                  </Box>
                  <Box display="flex" justifyContent="space-between">
                    <Typography>DASH</Typography>
                    <Chip label="Enabled" color="success" size="small" />
                  </Box>
                </Box>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      </TabPanel>
    </Box>
  );
};

export default MediaDashboard;

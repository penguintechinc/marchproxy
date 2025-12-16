/**
 * Metrics Dashboard Page
 *
 * Prometheus metrics visualization with query builder, time-series charts,
 * and customizable dashboards.
 */

import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Paper,
  Typography,
  Grid,
  TextField,
  Button,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  CircularProgress,
  Alert,
  Card,
  CardContent,
  IconButton,
  Tooltip,
  ToggleButton,
  ToggleButtonGroup,
  Chip,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import DownloadIcon from '@mui/icons-material/Download';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip as RechartsTooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import {
  queryRangeMetrics,
  getMetricNames,
  exportMetrics,
} from '../../services/observabilityApi';
import {
  MetricQuery,
  MetricQueryResponse,
  TimeSeriesData,
  ChartDataPoint,
} from '../../services/observabilityTypes';

// Predefined quick queries
const QUICK_QUERIES = [
  {
    name: 'Request Rate',
    query: 'rate(http_requests_total[5m])',
    description: 'HTTP requests per second',
  },
  {
    name: 'Error Rate',
    query: 'rate(http_requests_total{status=~"5.."}[5m])',
    description: 'HTTP 5xx errors per second',
  },
  {
    name: 'P95 Latency',
    query: 'histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))',
    description: '95th percentile request latency',
  },
  {
    name: 'P99 Latency',
    query: 'histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))',
    description: '99th percentile request latency',
  },
  {
    name: 'Throughput',
    query: 'rate(bytes_sent_total[5m])',
    description: 'Bytes sent per second',
  },
  {
    name: 'Active Connections',
    query: 'connections_active',
    description: 'Currently active connections',
  },
];

// Time range presets
const TIME_RANGES = [
  { label: '5m', value: 300 },
  { label: '15m', value: 900 },
  { label: '1h', value: 3600 },
  { label: '6h', value: 21600 },
  { label: '24h', value: 86400 },
  { label: '7d', value: 604800 },
];

export const Metrics: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState<string>('');
  const [metricNames, setMetricNames] = useState<string[]>([]);
  const [metricsData, setMetricsData] = useState<TimeSeriesData[]>([]);
  const [timeRange, setTimeRange] = useState<number>(3600); // Default 1h
  const [autoRefresh, setAutoRefresh] = useState<boolean>(false);
  const [refreshInterval, setRefreshInterval] = useState<number>(30); // seconds
  const [step, setStep] = useState<string>('15s');

  // Load metric names
  useEffect(() => {
    loadMetricNames();
  }, []);

  // Auto-refresh
  useEffect(() => {
    if (autoRefresh && Number(refreshInterval) > 0 && query) {
      const interval = setInterval(() => {
        executeQuery();
      }, Number(refreshInterval) * 1000);
      return () => clearInterval(interval);
    }
  }, [autoRefresh, refreshInterval, query]);

  const loadMetricNames = async () => {
    try {
      const names = await getMetricNames();
      setMetricNames(names);
    } catch (err: any) {
      console.error('Failed to load metric names:', err);
    }
  };

  const executeQuery = async () => {
    if (!query.trim()) {
      setError('Please enter a PromQL query');
      return;
    }

    try {
      setLoading(true);
      setError(null);

      const now = Math.floor(Date.now() / 1000);
      const start = now - timeRange;

      const queryParams: MetricQuery = {
        query: query.trim(),
        start: new Date(start * 1000).toISOString(),
        end: new Date(now * 1000).toISOString(),
        step,
      };

      const response: MetricQueryResponse = await queryRangeMetrics(queryParams);

      // Transform response to chart data
      const chartData: TimeSeriesData[] = response.result.map((series) => {
        const seriesLabel = Object.entries(series.metric)
          .map(([k, v]) => `${k}="${v}"`)
          .join(', ');

        return {
          name: seriesLabel || 'default',
          data: series.values.map((v) => ({
            timestamp: v.timestamp * 1000,
            value: v.value,
          })),
        };
      });

      setMetricsData(chartData);
    } catch (err: any) {
      setError(err.message || 'Failed to execute query');
      console.error('Error executing query:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleQuickQuery = (quickQuery: string) => {
    setQuery(quickQuery);
  };

  const handleExport = async () => {
    try {
      const now = Math.floor(Date.now() / 1000);
      const start = now - timeRange;

      const queryParams: MetricQuery = {
        query: query.trim(),
        start: new Date(start * 1000).toISOString(),
        end: new Date(now * 1000).toISOString(),
        step,
      };

      const blob = await exportMetrics(queryParams);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `metrics-${new Date().toISOString()}.csv`;
      a.click();
      window.URL.revokeObjectURL(url);
    } catch (err: any) {
      console.error('Failed to export metrics:', err);
    }
  };

  // Merge all series data for chart display
  const getChartData = (): any[] => {
    if (!metricsData.length) return [];

    const allTimestamps = new Set<number>();
    metricsData.forEach((series) => {
      series.data.forEach((point) => {
        allTimestamps.add(point.timestamp);
      });
    });

    const sortedTimestamps = Array.from(allTimestamps).sort((a, b) => a - b);

    return sortedTimestamps.map((timestamp) => {
      const dataPoint: any = {
        timestamp,
        time: new Date(timestamp).toLocaleTimeString(),
      };

      metricsData.forEach((series) => {
        const point = series.data.find((p) => p.timestamp === timestamp);
        dataPoint[series.name] = point?.value || null;
      });

      return dataPoint;
    });
  };

  // Calculate statistics
  const getStatistics = () => {
    if (!metricsData.length) return null;

    const primarySeries = metricsData[0];
    const values = primarySeries.data.map((p) => p.value);

    const avg = values.reduce((sum, v) => sum + v, 0) / values.length;
    const min = Math.min(...values);
    const max = Math.max(...values);
    const current = values[values.length - 1] || 0;

    return { avg, min, max, current };
  };

  const stats = getStatistics();

  return (
    <Box>
      <Typography variant="h4" gutterBottom>
        Metrics Dashboard
      </Typography>

      {/* Query Builder */}
      <Paper sx={{ p: 3, mb: 3 }}>
        <Grid container spacing={2}>
          <Grid item xs={12}>
            <TextField
              fullWidth
              label="PromQL Query"
              placeholder="Enter PromQL query..."
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              multiline
              rows={2}
            />
          </Grid>

          <Grid item xs={12}>
            <Box display="flex" gap={1} flexWrap="wrap">
              {QUICK_QUERIES.map((q) => (
                <Tooltip key={q.name} title={q.description}>
                  <Chip
                    label={q.name}
                    onClick={() => handleQuickQuery(q.query)}
                    variant="outlined"
                    size="small"
                  />
                </Tooltip>
              ))}
            </Box>
          </Grid>

          <Grid item xs={12} md={3}>
            <FormControl fullWidth>
              <InputLabel>Time Range</InputLabel>
              <Select
                value={timeRange}
                onChange={(e) => setTimeRange(e.target.value as number)}
                label="Time Range"
              >
                {TIME_RANGES.map((range) => (
                  <MenuItem key={range.label} value={range.value}>
                    {range.label}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>

          <Grid item xs={12} md={2}>
            <TextField
              fullWidth
              label="Step"
              value={step}
              onChange={(e) => setStep(e.target.value)}
              placeholder="e.g., 15s"
            />
          </Grid>

          <Grid item xs={12} md={2}>
            <FormControl fullWidth>
              <InputLabel>Auto-Refresh</InputLabel>
              <Select
                value={refreshInterval}
                onChange={(e) => {
                  const val = e.target.value as number;
                  setRefreshInterval(val);
                  setAutoRefresh(Number(val) > 0);
                }}
                label="Auto-Refresh"
              >
                <MenuItem value={0}>Off</MenuItem>
                <MenuItem value={10}>10s</MenuItem>
                <MenuItem value={30}>30s</MenuItem>
                <MenuItem value={60}>1m</MenuItem>
                <MenuItem value={300}>5m</MenuItem>
              </Select>
            </FormControl>
          </Grid>

          <Grid item xs={12} md={5}>
            <Box display="flex" gap={1}>
              <Button
                variant="contained"
                onClick={executeQuery}
                disabled={loading}
                startIcon={<PlayArrowIcon />}
                fullWidth
              >
                Execute Query
              </Button>
              <Tooltip title="Refresh">
                <IconButton onClick={executeQuery} disabled={loading}>
                  <RefreshIcon />
                </IconButton>
              </Tooltip>
              <Tooltip title="Export">
                <IconButton onClick={handleExport} disabled={!metricsData.length}>
                  <DownloadIcon />
                </IconButton>
              </Tooltip>
            </Box>
          </Grid>
        </Grid>
      </Paper>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      {loading && (
        <Box display="flex" justifyContent="center" p={4}>
          <CircularProgress />
        </Box>
      )}

      {!loading && metricsData.length > 0 && (
        <>
          {/* Statistics Cards */}
          {stats && (
            <Grid container spacing={2} sx={{ mb: 3 }}>
              <Grid item xs={12} md={3}>
                <Card>
                  <CardContent>
                    <Typography color="textSecondary" gutterBottom>
                      Current Value
                    </Typography>
                    <Typography variant="h4">
                      {stats.current.toFixed(2)}
                    </Typography>
                  </CardContent>
                </Card>
              </Grid>
              <Grid item xs={12} md={3}>
                <Card>
                  <CardContent>
                    <Typography color="textSecondary" gutterBottom>
                      Average
                    </Typography>
                    <Typography variant="h4">{stats.avg.toFixed(2)}</Typography>
                  </CardContent>
                </Card>
              </Grid>
              <Grid item xs={12} md={3}>
                <Card>
                  <CardContent>
                    <Typography color="textSecondary" gutterBottom>
                      Minimum
                    </Typography>
                    <Typography variant="h4">{stats.min.toFixed(2)}</Typography>
                  </CardContent>
                </Card>
              </Grid>
              <Grid item xs={12} md={3}>
                <Card>
                  <CardContent>
                    <Typography color="textSecondary" gutterBottom>
                      Maximum
                    </Typography>
                    <Typography variant="h4">{stats.max.toFixed(2)}</Typography>
                  </CardContent>
                </Card>
              </Grid>
            </Grid>
          )}

          {/* Time Series Chart */}
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
              Time Series
            </Typography>
            <ResponsiveContainer width="100%" height={400}>
              <LineChart data={getChartData()}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis />
                <RechartsTooltip />
                <Legend />
                {metricsData.map((series, index) => (
                  <Line
                    key={series.name}
                    type="monotone"
                    dataKey={series.name}
                    stroke={series.color || `hsl(${index * 60}, 70%, 50%)`}
                    dot={false}
                  />
                ))}
              </LineChart>
            </ResponsiveContainer>
          </Paper>
        </>
      )}

      {!loading && metricsData.length === 0 && query && (
        <Alert severity="info">
          No data available for the selected query and time range
        </Alert>
      )}
    </Box>
  );
};

export default Metrics;

/**
 * Tracing Dashboard Page
 *
 * Displays distributed tracing information with Jaeger UI integration,
 * trace search, latency histograms, and error rate tracking.
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
  Chip,
  CircularProgress,
  Alert,
  Card,
  CardContent,
  IconButton,
  Tooltip,
  Tabs,
  Tab,
} from '@mui/material';
import { DateTimePicker } from '@mui/x-date-pickers';
import SearchIcon from '@mui/icons-material/Search';
import RefreshIcon from '@mui/icons-material/Refresh';
import DownloadIcon from '@mui/icons-material/Download';
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip as RechartsTooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import {
  getTraces,
  getJaegerUrl,
  getServices,
  getOperations,
  exportTraces,
} from '../../services/observabilityApi';
import { Trace, TraceSearchParams } from '../../services/observabilityTypes';
import { ServiceGraph } from '../../components/Observability/ServiceGraph';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

const TabPanel: React.FC<TabPanelProps> = ({ children, value, index }) => {
  return (
    <div hidden={value !== index}>
      {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
    </div>
  );
};

export const Tracing: React.FC = () => {
  const [tabValue, setTabValue] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [traces, setTraces] = useState<Trace[]>([]);
  const [jaegerUrl, setJaegerUrl] = useState<string>('');
  const [services, setServices] = useState<string[]>([]);
  const [operations, setOperations] = useState<string[]>([]);

  // Search parameters
  const [searchParams, setSearchParams] = useState<TraceSearchParams>({
    start: new Date(Date.now() - 3600000).toISOString(),
    end: new Date().toISOString(),
    limit: 50,
  });

  const [selectedService, setSelectedService] = useState<string>('');
  const [selectedOperation, setSelectedOperation] = useState<string>('');
  const [minDuration, setMinDuration] = useState<string>('');
  const [maxDuration, setMaxDuration] = useState<string>('');

  // Load initial data
  useEffect(() => {
    loadJaegerUrl();
    loadServices();
  }, []);

  // Load operations when service changes
  useEffect(() => {
    if (selectedService) {
      loadOperations(selectedService);
    }
  }, [selectedService]);

  const loadJaegerUrl = async () => {
    try {
      const url = await getJaegerUrl();
      setJaegerUrl(url);
    } catch (err: any) {
      console.error('Failed to load Jaeger URL:', err);
    }
  };

  const loadServices = async () => {
    try {
      const svc = await getServices();
      setServices(svc);
    } catch (err: any) {
      console.error('Failed to load services:', err);
    }
  };

  const loadOperations = async (service: string) => {
    try {
      const ops = await getOperations(service);
      setOperations(ops);
    } catch (err: any) {
      console.error('Failed to load operations:', err);
    }
  };

  const handleSearch = async () => {
    try {
      setLoading(true);
      setError(null);

      const params: TraceSearchParams = {
        ...searchParams,
        service: selectedService || undefined,
        operation: selectedOperation || undefined,
        minDuration: minDuration || undefined,
        maxDuration: maxDuration || undefined,
      };

      const results = await getTraces(params);
      setTraces(results);
    } catch (err: any) {
      setError(err.message || 'Failed to search traces');
      console.error('Error searching traces:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleExport = async () => {
    try {
      const params: TraceSearchParams = {
        ...searchParams,
        service: selectedService || undefined,
        operation: selectedOperation || undefined,
      };

      const blob = await exportTraces(params);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `traces-${new Date().toISOString()}.csv`;
      a.click();
      window.URL.revokeObjectURL(url);
    } catch (err: any) {
      console.error('Failed to export traces:', err);
    }
  };

  // Calculate latency histogram data
  const getLatencyHistogram = () => {
    if (!traces.length) return [];

    const buckets = [0, 10, 50, 100, 500, 1000, 5000, 10000];
    const histogram = buckets.map((bucket, i) => {
      const nextBucket = buckets[i + 1] || Infinity;
      const count = traces.filter((trace) => {
        const duration = trace.spans[0]?.duration || 0;
        const durationMs = duration / 1000;
        return durationMs >= bucket && durationMs < nextBucket;
      }).length;

      return {
        range: i < buckets.length - 1 ? `${bucket}-${nextBucket}ms` : `${bucket}ms+`,
        count,
      };
    });

    return histogram;
  };

  // Calculate error rate over time
  const getErrorRateData = () => {
    if (!traces.length) return [];

    const timeGroups: { [key: string]: { total: number; errors: number } } = {};

    traces.forEach((trace) => {
      const timestamp = new Date(trace.spans[0]?.startTime || 0);
      const timeKey = new Date(
        Math.floor(timestamp.getTime() / 60000) * 60000
      ).toISOString();

      if (!timeGroups[timeKey]) {
        timeGroups[timeKey] = { total: 0, errors: 0 };
      }

      timeGroups[timeKey].total++;

      const hasError = trace.spans.some((span) =>
        span.tags.some((tag) => tag.key === 'error' && tag.value === true)
      );

      if (hasError) {
        timeGroups[timeKey].errors++;
      }
    });

    return Object.entries(timeGroups).map(([time, data]) => ({
      time: new Date(time).toLocaleTimeString(),
      errorRate: (data.errors / data.total) * 100,
      total: data.total,
    }));
  };

  return (
    <Box>
      <Typography variant="h4" gutterBottom>
        Distributed Tracing
      </Typography>

      <Tabs value={tabValue} onChange={(_, v) => setTabValue(v)} sx={{ mb: 2 }}>
        <Tab label="Trace Search" />
        <Tab label="Jaeger UI" />
        <Tab label="Service Graph" />
      </Tabs>

      <TabPanel value={tabValue} index={0}>
        {/* Search Filters */}
        <Paper sx={{ p: 3, mb: 3 }}>
          <Grid container spacing={2}>
            <Grid item xs={12} md={3}>
              <FormControl fullWidth>
                <InputLabel>Service</InputLabel>
                <Select
                  value={selectedService}
                  onChange={(e) => setSelectedService(e.target.value)}
                  label="Service"
                >
                  <MenuItem value="">All Services</MenuItem>
                  {services.map((svc) => (
                    <MenuItem key={svc} value={svc}>
                      {svc}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>

            <Grid item xs={12} md={3}>
              <FormControl fullWidth>
                <InputLabel>Operation</InputLabel>
                <Select
                  value={selectedOperation}
                  onChange={(e) => setSelectedOperation(e.target.value)}
                  label="Operation"
                  disabled={!selectedService}
                >
                  <MenuItem value="">All Operations</MenuItem>
                  {operations.map((op) => (
                    <MenuItem key={op} value={op}>
                      {op}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>

            <Grid item xs={12} md={2}>
              <TextField
                fullWidth
                label="Min Duration"
                placeholder="e.g., 100ms"
                value={minDuration}
                onChange={(e) => setMinDuration(e.target.value)}
              />
            </Grid>

            <Grid item xs={12} md={2}>
              <TextField
                fullWidth
                label="Max Duration"
                placeholder="e.g., 5s"
                value={maxDuration}
                onChange={(e) => setMaxDuration(e.target.value)}
              />
            </Grid>

            <Grid item xs={12} md={2}>
              <Box display="flex" gap={1}>
                <Button
                  variant="contained"
                  onClick={handleSearch}
                  disabled={loading}
                  startIcon={<SearchIcon />}
                  fullWidth
                >
                  Search
                </Button>
                <Tooltip title="Export">
                  <IconButton onClick={handleExport} disabled={!traces.length}>
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

        {!loading && traces.length > 0 && (
          <>
            {/* Statistics Cards */}
            <Grid container spacing={2} sx={{ mb: 3 }}>
              <Grid item xs={12} md={3}>
                <Card>
                  <CardContent>
                    <Typography color="textSecondary" gutterBottom>
                      Total Traces
                    </Typography>
                    <Typography variant="h4">{traces.length}</Typography>
                  </CardContent>
                </Card>
              </Grid>
              <Grid item xs={12} md={3}>
                <Card>
                  <CardContent>
                    <Typography color="textSecondary" gutterBottom>
                      Avg Duration
                    </Typography>
                    <Typography variant="h4">
                      {(
                        traces.reduce(
                          (sum, t) => sum + (t.spans[0]?.duration || 0),
                          0
                        ) /
                        traces.length /
                        1000
                      ).toFixed(0)}
                      ms
                    </Typography>
                  </CardContent>
                </Card>
              </Grid>
            </Grid>

            {/* Latency Histogram */}
            <Paper sx={{ p: 3, mb: 3 }}>
              <Typography variant="h6" gutterBottom>
                Latency Distribution
              </Typography>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={getLatencyHistogram()}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="range" />
                  <YAxis />
                  <RechartsTooltip />
                  <Legend />
                  <Bar dataKey="count" fill="#8884d8" />
                </BarChart>
              </ResponsiveContainer>
            </Paper>

            {/* Error Rate Chart */}
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>
                Error Rate Over Time
              </Typography>
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={getErrorRateData()}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="time" />
                  <YAxis />
                  <RechartsTooltip />
                  <Legend />
                  <Line
                    type="monotone"
                    dataKey="errorRate"
                    stroke="#f44336"
                    name="Error Rate (%)"
                  />
                </LineChart>
              </ResponsiveContainer>
            </Paper>
          </>
        )}
      </TabPanel>

      <TabPanel value={tabValue} index={1}>
        {jaegerUrl ? (
          <Paper sx={{ height: 800 }}>
            <iframe
              src={jaegerUrl}
              style={{ width: '100%', height: '100%', border: 'none' }}
              title="Jaeger UI"
            />
          </Paper>
        ) : (
          <Alert severity="info">Jaeger UI not configured</Alert>
        )}
      </TabPanel>

      <TabPanel value={tabValue} index={2}>
        <ServiceGraph
          timeRange={{
            start: searchParams.start,
            end: searchParams.end,
          }}
          autoRefresh={true}
          refreshInterval={30}
        />
      </TabPanel>
    </Box>
  );
};

export default Tracing;

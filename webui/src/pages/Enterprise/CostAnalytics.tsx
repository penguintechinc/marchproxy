/**
 * Cost Analytics Dashboard
 *
 * Enterprise feature for cloud egress cost tracking and optimization:
 * - Time-series cost charts
 * - Cost breakdown by provider, service, region
 * - Cost optimization recommendations
 * - Budget alerts and thresholds
 * - Monthly/yearly cost comparisons
 * - Export reports (CSV, PDF)
 */

import React, { useState, useEffect } from 'react';
import {
  Box,
  Container,
  Typography,
  Button,
  Card,
  CardContent,
  Grid,
  TextField,
  Alert,
  Chip,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  MenuItem,
  CircularProgress,
} from '@mui/material';
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip as RechartsTooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import TrendingDownIcon from '@mui/icons-material/TrendingDown';
import SaveAltIcon from '@mui/icons-material/SaveAlt';
import LightbulbIcon from '@mui/icons-material/Lightbulb';
import AttachMoneyIcon from '@mui/icons-material/AttachMoney';
import LicenseGate from '../../components/Common/LicenseGate';
import {
  getCostAnalytics,
  getCostTimeSeries,
  getCostOptimizations,
  exportCostReport,
} from '../../services/enterpriseApi';
import type {
  CostAnalytics,
  CostTimeSeries,
  CostOptimization,
  CostBreakdown,
} from '../../services/types';
import { useLicense } from '../../hooks/useLicense';

const PROVIDER_COLORS: Record<string, string> = {
  AWS: '#FF9900',
  GCP: '#4285F4',
  Azure: '#0078D4',
  Custom: '#757575',
};

const CostAnalyticsDashboard: React.FC = () => {
  const { isEnterprise, hasFeature, loading: licenseLoading } = useLicense();
  const hasEnterpriseAccess = isEnterprise || hasFeature('multi_cloud_routing');
  const [analytics, setAnalytics] = useState<CostAnalytics | null>(null);
  const [timeSeries, setTimeSeries] = useState<CostTimeSeries[]>([]);
  const [optimizations, setOptimizations] = useState<CostOptimization[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dateRange, setDateRange] = useState<'week' | 'month' | 'year'>('month');
  const [exportFormat, setExportFormat] = useState<'csv' | 'pdf'>('csv');

  useEffect(() => {
    loadCostData();
  }, [dateRange]);

  const loadCostData = async () => {
    try {
      setLoading(true);
      const endDate = new Date();
      const startDate = new Date();

      switch (dateRange) {
        case 'week':
          startDate.setDate(endDate.getDate() - 7);
          break;
        case 'month':
          startDate.setMonth(endDate.getMonth() - 1);
          break;
        case 'year':
          startDate.setFullYear(endDate.getFullYear() - 1);
          break;
      }

      const [analyticsData, timeSeriesData, optimizationsData] = await Promise.all([
        getCostAnalytics(
          startDate.toISOString().split('T')[0],
          endDate.toISOString().split('T')[0]
        ),
        getCostTimeSeries(
          startDate.toISOString().split('T')[0],
          endDate.toISOString().split('T')[0],
          dateRange === 'week' ? 'day' : dateRange === 'month' ? 'day' : 'month'
        ),
        getCostOptimizations(),
      ]);

      setAnalytics(analyticsData);
      setTimeSeries(timeSeriesData);
      setOptimizations(optimizationsData);
      setError(null);
    } catch (err) {
      setError('Failed to load cost analytics');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleExport = async () => {
    try {
      const endDate = new Date();
      const startDate = new Date();
      startDate.setMonth(endDate.getMonth() - 1);

      const blob = await exportCostReport(
        startDate.toISOString().split('T')[0],
        endDate.toISOString().split('T')[0],
        exportFormat
      );

      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `cost-report-${Date.now()}.${exportFormat}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err) {
      setError('Failed to export cost report');
      console.error(err);
    }
  };

  const formatCurrency = (value: number): string => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
    }).format(value);
  };

  const getPriorityColor = (priority: string): 'error' | 'warning' | 'info' => {
    switch (priority) {
      case 'high':
        return 'error';
      case 'medium':
        return 'warning';
      case 'low':
        return 'info';
      default:
        return 'info';
    }
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  return (
    <LicenseGate
      featureName="Cost Analytics & Optimization"
      hasAccess={hasEnterpriseAccess}
      isLoading={licenseLoading}
    >
      <Container maxWidth="xl">
        <Box py={4}>
          <Box display="flex" justifyContent="space-between" alignItems="center" mb={4}>
            <Typography variant="h4" fontWeight="bold">
              Cost Analytics
            </Typography>
            <Box display="flex" gap={2}>
              <TextField
                select
                size="small"
                label="Date Range"
                value={dateRange}
                onChange={(e) => setDateRange(e.target.value as any)}
                sx={{ minWidth: 150 }}
              >
                <MenuItem value="week">Last 7 Days</MenuItem>
                <MenuItem value="month">Last 30 Days</MenuItem>
                <MenuItem value="year">Last Year</MenuItem>
              </TextField>
              <TextField
                select
                size="small"
                label="Export Format"
                value={exportFormat}
                onChange={(e) => setExportFormat(e.target.value as any)}
                sx={{ minWidth: 120 }}
              >
                <MenuItem value="csv">CSV</MenuItem>
                <MenuItem value="pdf">PDF</MenuItem>
              </TextField>
              <Button
                variant="contained"
                startIcon={<SaveAltIcon />}
                onClick={handleExport}
              >
                Export Report
              </Button>
            </Box>
          </Box>

          {error && (
            <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
              {error}
            </Alert>
          )}

          {analytics && (
            <>
              <Grid container spacing={3} mb={3}>
                <Grid item xs={12} sm={6} md={3}>
                  <Card>
                    <CardContent>
                      <Box display="flex" alignItems="center" gap={1} mb={1}>
                        <AttachMoneyIcon color="primary" />
                        <Typography variant="caption" color="text.secondary">
                          Total Cost
                        </Typography>
                      </Box>
                      <Typography variant="h4" fontWeight="bold">
                        {formatCurrency(analytics.total_cost_usd)}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {analytics.period_start} to {analytics.period_end}
                      </Typography>
                    </CardContent>
                  </Card>
                </Grid>

                <Grid item xs={12} sm={6} md={3}>
                  <Card>
                    <CardContent>
                      <Box display="flex" alignItems="center" gap={1} mb={1}>
                        <TrendingUpIcon color="warning" />
                        <Typography variant="caption" color="text.secondary">
                          Monthly Projection
                        </Typography>
                      </Box>
                      <Typography variant="h4" fontWeight="bold">
                        {formatCurrency(analytics.monthly_projection)}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        Based on current usage
                      </Typography>
                    </CardContent>
                  </Card>
                </Grid>

                <Grid item xs={12} sm={6} md={3}>
                  <Card>
                    <CardContent>
                      <Box display="flex" alignItems="center" gap={1} mb={1}>
                        <TrendingUpIcon color="info" />
                        <Typography variant="caption" color="text.secondary">
                          Yearly Projection
                        </Typography>
                      </Box>
                      <Typography variant="h4" fontWeight="bold">
                        {formatCurrency(analytics.yearly_projection)}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        Estimated annual cost
                      </Typography>
                    </CardContent>
                  </Card>
                </Grid>

                <Grid item xs={12} sm={6} md={3}>
                  <Card>
                    <CardContent>
                      <Box display="flex" alignItems="center" gap={1} mb={1}>
                        <LightbulbIcon color="success" />
                        <Typography variant="caption" color="text.secondary">
                          Optimization Potential
                        </Typography>
                      </Box>
                      <Typography variant="h4" fontWeight="bold">
                        {formatCurrency(
                          optimizations.reduce(
                            (sum, opt) => sum + opt.potential_savings_usd,
                            0
                          )
                        )}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {optimizations.length} recommendations
                      </Typography>
                    </CardContent>
                  </Card>
                </Grid>
              </Grid>

              <Grid container spacing={3} mb={3}>
                <Grid item xs={12} lg={8}>
                  <Card>
                    <CardContent>
                      <Typography variant="h6" gutterBottom>
                        Cost Over Time
                      </Typography>
                      <ResponsiveContainer width="100%" height={300}>
                        <LineChart data={timeSeries}>
                          <CartesianGrid strokeDasharray="3 3" />
                          <XAxis
                            dataKey="timestamp"
                            tickFormatter={(value) =>
                              new Date(value).toLocaleDateString()
                            }
                          />
                          <YAxis
                            tickFormatter={(value) => `$${value}`}
                            label={{
                              value: 'Cost (USD)',
                              angle: -90,
                              position: 'insideLeft',
                            }}
                          />
                          <RechartsTooltip
                            formatter={(value: number) => formatCurrency(value)}
                            labelFormatter={(label) =>
                              new Date(label).toLocaleDateString()
                            }
                          />
                          <Legend />
                          <Line
                            type="monotone"
                            dataKey="cost_usd"
                            name="Cost"
                            stroke="#8884d8"
                            strokeWidth={2}
                          />
                        </LineChart>
                      </ResponsiveContainer>
                    </CardContent>
                  </Card>
                </Grid>

                <Grid item xs={12} lg={4}>
                  <Card>
                    <CardContent>
                      <Typography variant="h6" gutterBottom>
                        Cost by Provider
                      </Typography>
                      <ResponsiveContainer width="100%" height={300}>
                        <PieChart>
                          <Pie
                            data={analytics.breakdown_by_provider as any}
                            dataKey="cost_usd"
                            nameKey="label"
                            cx="50%"
                            cy="50%"
                            outerRadius={80}
                            label
                          >
                            {analytics.breakdown_by_provider.map((entry, index) => (
                              <Cell
                                key={`cell-${index}`}
                                fill={PROVIDER_COLORS[entry.label] || '#757575'}
                              />
                            ))}
                          </Pie>
                          <RechartsTooltip formatter={(value: number) => formatCurrency(value)} />
                        </PieChart>
                      </ResponsiveContainer>
                    </CardContent>
                  </Card>
                </Grid>
              </Grid>

              <Grid container spacing={3}>
                <Grid item xs={12} md={6}>
                  <Card>
                    <CardContent>
                      <Typography variant="h6" gutterBottom>
                        Cost by Region
                      </Typography>
                      <ResponsiveContainer width="100%" height={300}>
                        <BarChart data={analytics.breakdown_by_region}>
                          <CartesianGrid strokeDasharray="3 3" />
                          <XAxis dataKey="label" />
                          <YAxis tickFormatter={(value) => `$${value}`} />
                          <RechartsTooltip
                            formatter={(value: number) => formatCurrency(value)}
                          />
                          <Legend />
                          <Bar dataKey="cost_usd" name="Cost" fill="#8884d8" />
                        </BarChart>
                      </ResponsiveContainer>
                    </CardContent>
                  </Card>
                </Grid>

                <Grid item xs={12} md={6}>
                  <Card>
                    <CardContent>
                      <Typography variant="h6" gutterBottom>
                        Cost Optimization Recommendations
                      </Typography>
                      <List>
                        {optimizations.map((opt, index) => (
                          <ListItem key={index} divider>
                            <ListItemIcon>
                              <LightbulbIcon color={getPriorityColor(opt.priority)} />
                            </ListItemIcon>
                            <ListItemText
                              primary={opt.recommendation}
                              secondary={
                                <Box display="flex" gap={1} mt={1}>
                                  <Chip
                                    label={`Save ${formatCurrency(
                                      opt.potential_savings_usd
                                    )}`}
                                    size="small"
                                    color="success"
                                  />
                                  <Chip
                                    label={opt.priority.toUpperCase()}
                                    size="small"
                                    color={getPriorityColor(opt.priority)}
                                  />
                                  <Chip
                                    label={opt.implementation_effort}
                                    size="small"
                                    variant="outlined"
                                  />
                                </Box>
                              }
                            />
                          </ListItem>
                        ))}
                        {optimizations.length === 0 && (
                          <ListItem>
                            <ListItemText
                              primary="No optimization recommendations available"
                              secondary="Your configuration is already optimized"
                            />
                          </ListItem>
                        )}
                      </List>
                    </CardContent>
                  </Card>
                </Grid>
              </Grid>
            </>
          )}
        </Box>
      </Container>
    </LicenseGate>
  );
};

export default CostAnalyticsDashboard;

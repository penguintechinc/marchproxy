import React, { useState } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Alert,
  Stack,
  Typography,
  Grid,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  LinearProgress,
} from '@mui/material';
import {
  Assessment,
  Download,
  CheckCircle,
  Warning,
  Error,
} from '@mui/icons-material';
import { DateTimePicker } from '@mui/x-date-pickers';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { apiClient } from '../../services/api';

interface ComplianceReport {
  report_id: string;
  standard: string;
  generated_at: string;
  start_time: string;
  end_time: string;
  total_events: number;
  summary: {
    access_attempts: number;
    successful_access: number;
    failed_access: number;
    failure_rate: number;
    unique_users: number;
    unique_services: number;
    policy_violations: number;
    certificate_issues: number;
    chain_integrity_valid: boolean;
  };
  findings: Array<{
    severity: string;
    category: string;
    description: string;
    count: number;
  }>;
  recommendations: string[];
}

const ComplianceReports: React.FC = () => {
  const [standard, setStandard] = useState<string>('SOC2');
  const [startDate, setStartDate] = useState<Date | null>(new Date(Date.now() - 30 * 24 * 60 * 60 * 1000));
  const [endDate, setEndDate] = useState<Date | null>(new Date());
  const [report, setReport] = useState<ComplianceReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const handleGenerate = async () => {
    if (!startDate || !endDate) {
      setError('Please select start and end dates');
      return;
    }

    try {
      setLoading(true);
      setError(null);
      setSuccess(null);

      const response = await apiClient.post('/api/v1/zerotrust/compliance-reports/generate', {
        standard: standard,
        start_time: startDate.toISOString(),
        end_time: endDate.toISOString(),
      });

      setReport(response.data.report);
      setSuccess('Compliance report generated successfully');
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to generate compliance report');
    } finally {
      setLoading(false);
    }
  };

  const handleExport = async (format: 'json' | 'html' | 'pdf') => {
    if (!report) {
      setError('No report to export');
      return;
    }

    try {
      setLoading(true);
      setError(null);

      const response = await apiClient.post(
        `/api/v1/zerotrust/compliance-reports/export`,
        {
          report_id: report.report_id,
          format: format,
        },
        {
          responseType: 'blob',
        }
      );

      // Create download link
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', `${report.report_id}.${format}`);
      document.body.appendChild(link);
      link.click();
      link.remove();

      setSuccess(`Report exported as ${format.toUpperCase()}`);
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to export report');
    } finally {
      setLoading(false);
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity.toLowerCase()) {
      case 'critical':
        return 'error';
      case 'high':
        return 'warning';
      case 'medium':
        return 'info';
      case 'low':
        return 'success';
      default:
        return 'default';
    }
  };

  const getSeverityIcon = (severity: string) => {
    switch (severity.toLowerCase()) {
      case 'critical':
        return <Error />;
      case 'high':
        return <Warning />;
      default:
        return <CheckCircle />;
    }
  };

  return (
    <LocalizationProvider dateAdapter={AdapterDateFns}>
      <Box>
        {/* Generation Controls */}
        <Card sx={{ mb: 2 }}>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              Generate Compliance Report
            </Typography>

            <Grid container spacing={2}>
              <Grid item xs={12} md={4}>
                <FormControl fullWidth>
                  <InputLabel>Compliance Standard</InputLabel>
                  <Select
                    value={standard}
                    onChange={(e) => setStandard(e.target.value)}
                    label="Compliance Standard"
                  >
                    <MenuItem value="SOC2">SOC2</MenuItem>
                    <MenuItem value="HIPAA">HIPAA</MenuItem>
                    <MenuItem value="PCI-DSS">PCI-DSS</MenuItem>
                  </Select>
                </FormControl>
              </Grid>

              <Grid item xs={12} md={4}>
                <DateTimePicker
                  label="Start Date"
                  value={startDate}
                  onChange={setStartDate}
                  renderInput={(params) => <TextField {...params} fullWidth />}
                />
              </Grid>

              <Grid item xs={12} md={4}>
                <DateTimePicker
                  label="End Date"
                  value={endDate}
                  onChange={setEndDate}
                  renderInput={(params) => <TextField {...params} fullWidth />}
                />
              </Grid>
            </Grid>

            <Stack direction="row" spacing={2} sx={{ mt: 2 }}>
              <Button
                variant="contained"
                startIcon={<Assessment />}
                onClick={handleGenerate}
                disabled={loading}
              >
                Generate Report
              </Button>

              {report && (
                <>
                  <Button
                    startIcon={<Download />}
                    onClick={() => handleExport('json')}
                    disabled={loading}
                  >
                    Export JSON
                  </Button>
                  <Button
                    startIcon={<Download />}
                    onClick={() => handleExport('html')}
                    disabled={loading}
                  >
                    Export HTML
                  </Button>
                  <Button
                    startIcon={<Download />}
                    onClick={() => handleExport('pdf')}
                    disabled={loading}
                  >
                    Export PDF
                  </Button>
                </>
              )}
            </Stack>

            {loading && <LinearProgress sx={{ mt: 2 }} />}
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

        {/* Report Display */}
        {report && (
          <>
            {/* Report Header */}
            <Card sx={{ mb: 2 }}>
              <CardContent>
                <Typography variant="h5" gutterBottom>
                  {report.standard} Compliance Report
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs={12} md={4}>
                    <Typography variant="body2" color="text.secondary">
                      Report ID
                    </Typography>
                    <Typography variant="body1">{report.report_id}</Typography>
                  </Grid>
                  <Grid item xs={12} md={4}>
                    <Typography variant="body2" color="text.secondary">
                      Generated
                    </Typography>
                    <Typography variant="body1">
                      {new Date(report.generated_at).toLocaleString()}
                    </Typography>
                  </Grid>
                  <Grid item xs={12} md={4}>
                    <Typography variant="body2" color="text.secondary">
                      Period
                    </Typography>
                    <Typography variant="body1">
                      {new Date(report.start_time).toLocaleDateString()} -{' '}
                      {new Date(report.end_time).toLocaleDateString()}
                    </Typography>
                  </Grid>
                </Grid>
              </CardContent>
            </Card>

            {/* Summary Statistics */}
            <Grid container spacing={2} sx={{ mb: 2 }}>
              <Grid item xs={12} md={3}>
                <Card>
                  <CardContent>
                    <Typography variant="body2" color="text.secondary">
                      Total Events
                    </Typography>
                    <Typography variant="h4">{report.total_events}</Typography>
                  </CardContent>
                </Card>
              </Grid>

              <Grid item xs={12} md={3}>
                <Card>
                  <CardContent>
                    <Typography variant="body2" color="text.secondary">
                      Success Rate
                    </Typography>
                    <Typography variant="h4">
                      {((1 - report.summary.failure_rate) * 100).toFixed(1)}%
                    </Typography>
                  </CardContent>
                </Card>
              </Grid>

              <Grid item xs={12} md={3}>
                <Card>
                  <CardContent>
                    <Typography variant="body2" color="text.secondary">
                      Policy Violations
                    </Typography>
                    <Typography variant="h4" color="error.main">
                      {report.summary.policy_violations}
                    </Typography>
                  </CardContent>
                </Card>
              </Grid>

              <Grid item xs={12} md={3}>
                <Card>
                  <CardContent>
                    <Typography variant="body2" color="text.secondary">
                      Chain Integrity
                    </Typography>
                    <Typography variant="h4">
                      {report.summary.chain_integrity_valid ? (
                        <Chip label="Valid" color="success" />
                      ) : (
                        <Chip label="Invalid" color="error" />
                      )}
                    </Typography>
                  </CardContent>
                </Card>
              </Grid>
            </Grid>

            {/* Findings */}
            {report.findings && report.findings.length > 0 && (
              <Card sx={{ mb: 2 }}>
                <CardContent>
                  <Typography variant="h6" gutterBottom>
                    Findings
                  </Typography>

                  <TableContainer>
                    <Table>
                      <TableHead>
                        <TableRow>
                          <TableCell>Severity</TableCell>
                          <TableCell>Category</TableCell>
                          <TableCell>Description</TableCell>
                          <TableCell>Count</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {report.findings.map((finding, index) => (
                          <TableRow key={index}>
                            <TableCell>
                              <Chip
                                icon={getSeverityIcon(finding.severity)}
                                label={finding.severity}
                                color={getSeverityColor(finding.severity) as any}
                                size="small"
                              />
                            </TableCell>
                            <TableCell>{finding.category}</TableCell>
                            <TableCell>{finding.description}</TableCell>
                            <TableCell>{finding.count}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                </CardContent>
              </Card>
            )}

            {/* Recommendations */}
            {report.recommendations && report.recommendations.length > 0 && (
              <Card>
                <CardContent>
                  <Typography variant="h6" gutterBottom>
                    Recommendations
                  </Typography>

                  <Stack spacing={1}>
                    {report.recommendations.map((rec, index) => (
                      <Alert key={index} severity="info">
                        {rec}
                      </Alert>
                    ))}
                  </Stack>
                </CardContent>
              </Card>
            )}
          </>
        )}
      </Box>
    </LocalizationProvider>
  );
};

export default ComplianceReports;

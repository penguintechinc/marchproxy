/**
 * Compliance Reports
 *
 * SOC2, HIPAA, and PCI-DSS compliance dashboard with automated checks,
 * evidence collection, and report generation.
 */

import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  Grid,
  Card,
  CardContent,
  LinearProgress,
  Chip,
  Alert,
  Snackbar,
  Tabs,
  Tab,
  List,
  ListItem,
  ListItemText,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  CircularProgress,
  Divider
} from '@mui/material';
import {
  CheckCircle as CompliantIcon,
  Cancel as NonCompliantIcon,
  RemoveCircle as NotApplicableIcon,
  Download as DownloadIcon,
  Upload as UploadIcon,
  Refresh as RefreshIcon,
  Assessment as ReportIcon,
  Verified as VerifiedIcon
} from '@mui/icons-material';
import { DatePicker } from '@mui/x-date-pickers/DatePicker';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import {
  ComplianceStatus,
  ComplianceRequirement,
  ComplianceReportRequest,
  getComplianceStatus,
  runComplianceCheck,
  uploadComplianceEvidence,
  generateComplianceReport
} from '../../services/securityApi';
import LicenseGate from '../../components/Common/LicenseGate';
import { useLicense } from '../../hooks/useLicense';

type Framework = 'soc2' | 'hipaa' | 'pci_dss';

const Compliance: React.FC = () => {
  const { isEnterprise, hasFeature, loading: licenseLoading } = useLicense();
  const hasEnterpriseAccess = isEnterprise || hasFeature('zero_trust');
  const [activeTab, setActiveTab] = useState<Framework>('soc2');
  const [complianceStatus, setComplianceStatus] = useState<ComplianceStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [evidenceDialogOpen, setEvidenceDialogOpen] = useState(false);
  const [reportDialogOpen, setReportDialogOpen] = useState(false);
  const [selectedRequirement, setSelectedRequirement] = useState<ComplianceRequirement | null>(null);
  const [reportRequest, setReportRequest] = useState<ComplianceReportRequest>({
    framework: 'soc2',
    start_date: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
    end_date: new Date().toISOString(),
    include_evidence: true
  });

  useEffect(() => {
    loadComplianceStatus(activeTab);
  }, [activeTab]);

  const loadComplianceStatus = async (framework: Framework) => {
    try {
      setLoading(true);
      const data = await getComplianceStatus(framework);
      setComplianceStatus(data);
    } catch (err: any) {
      setError(err.message || 'Failed to load compliance status');
    } finally {
      setLoading(false);
    }
  };

  const handleRunCheck = async () => {
    try {
      setLoading(true);
      const data = await runComplianceCheck(activeTab);
      setComplianceStatus(data);
      setSuccess('Compliance check completed successfully');
    } catch (err: any) {
      setError(err.message || 'Failed to run compliance check');
    } finally {
      setLoading(false);
    }
  };

  const handleUploadEvidence = async (file: File) => {
    if (!selectedRequirement) return;

    try {
      setLoading(true);
      const formData = new FormData();
      formData.append('file', file);
      formData.append('description', `Evidence for ${selectedRequirement.name}`);
      formData.append('evidence_type', 'document');

      await uploadComplianceEvidence(
        activeTab,
        selectedRequirement.id,
        formData
      );

      setSuccess('Evidence uploaded successfully');
      setEvidenceDialogOpen(false);
      loadComplianceStatus(activeTab);
    } catch (err: any) {
      setError(err.message || 'Failed to upload evidence');
    } finally {
      setLoading(false);
    }
  };

  const handleGenerateReport = async () => {
    try {
      setLoading(true);
      const blob = await generateComplianceReport({
        ...reportRequest,
        framework: activeTab
      });

      // Create download link
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `compliance-report-${activeTab}-${new Date().toISOString()}.pdf`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);

      setSuccess('Compliance report generated successfully');
      setReportDialogOpen(false);
    } catch (err: any) {
      setError(err.message || 'Failed to generate report');
    } finally {
      setLoading(false);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'compliant':
        return <CompliantIcon color="success" />;
      case 'non_compliant':
        return <NonCompliantIcon color="error" />;
      case 'not_applicable':
        return <NotApplicableIcon color="disabled" />;
      default:
        return null;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'compliant':
        return 'success';
      case 'non_compliant':
        return 'error';
      case 'not_applicable':
        return 'default';
      default:
        return 'default';
    }
  };

  const getFrameworkName = (framework: Framework) => {
    switch (framework) {
      case 'soc2':
        return 'SOC 2';
      case 'hipaa':
        return 'HIPAA';
      case 'pci_dss':
        return 'PCI-DSS';
    }
  };

  return (
    <LicenseGate
      featureName="Zero-Trust Security"
      hasAccess={hasEnterpriseAccess}
      isLoading={licenseLoading}
    >
      <Box sx={{ p: 3 }}>
        <Box sx={{ mb: 3, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography variant="h4">Compliance Management</Typography>
          <Box>
            <Button
              variant="outlined"
              startIcon={<ReportIcon />}
              onClick={() => setReportDialogOpen(true)}
              disabled={loading}
              sx={{ mr: 1 }}
            >
              Generate Report
            </Button>
            <Button
              variant="contained"
              startIcon={<RefreshIcon />}
              onClick={handleRunCheck}
              disabled={loading}
            >
              Run Check
            </Button>
          </Box>
        </Box>

        <Paper sx={{ mb: 3 }}>
          <Tabs
            value={activeTab}
            onChange={(e, newValue) => setActiveTab(newValue)}
            variant="fullWidth"
          >
            <Tab label="SOC 2" value="soc2" />
            <Tab label="HIPAA" value="hipaa" />
            <Tab label="PCI-DSS" value="pci_dss" />
          </Tabs>
        </Paper>

        {complianceStatus && (
          <>
            {/* Overall Status */}
            <Grid container spacing={3} sx={{ mb: 3 }}>
              <Grid item xs={12} md={3}>
                <Card>
                  <CardContent>
                    <Typography variant="h6" gutterBottom>
                      Overall Score
                    </Typography>
                    <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                      <CircularProgress
                        variant="determinate"
                        value={complianceStatus.overall_score}
                        size={80}
                        thickness={4}
                        color={
                          complianceStatus.overall_score >= 90
                            ? 'success'
                            : complianceStatus.overall_score >= 70
                            ? 'warning'
                            : 'error'
                        }
                      />
                      <Typography variant="h3" sx={{ ml: 2 }}>
                        {complianceStatus.overall_score}%
                      </Typography>
                    </Box>
                    <Typography variant="caption" color="textSecondary">
                      Last Assessment:{' '}
                      {new Date(complianceStatus.last_assessment).toLocaleDateString()}
                    </Typography>
                  </CardContent>
                </Card>
              </Grid>

              <Grid item xs={12} md={3}>
                <Card>
                  <CardContent>
                    <Typography variant="h6" gutterBottom>
                      Controls
                    </Typography>
                    <Typography variant="h3" color="success.main">
                      {complianceStatus.passing_controls}
                    </Typography>
                    <Typography variant="body2" color="textSecondary">
                      of {complianceStatus.total_controls} passing
                    </Typography>
                    <LinearProgress
                      variant="determinate"
                      value={
                        (complianceStatus.passing_controls /
                          complianceStatus.total_controls) *
                        100
                      }
                      sx={{ mt: 2 }}
                      color="success"
                    />
                  </CardContent>
                </Card>
              </Grid>

              <Grid item xs={12} md={3}>
                <Card>
                  <CardContent>
                    <Typography variant="h6" gutterBottom>
                      Framework
                    </Typography>
                    <Typography variant="h4" gutterBottom>
                      {getFrameworkName(activeTab)}
                    </Typography>
                    <Chip
                      icon={<VerifiedIcon />}
                      label={complianceStatus.overall_score >= 90 ? 'Compliant' : 'Non-Compliant'}
                      color={complianceStatus.overall_score >= 90 ? 'success' : 'warning'}
                    />
                  </CardContent>
                </Card>
              </Grid>

              <Grid item xs={12} md={3}>
                <Card>
                  <CardContent>
                    <Typography variant="h6" gutterBottom>
                      Next Assessment
                    </Typography>
                    <Typography variant="h5" gutterBottom>
                      {new Date(complianceStatus.next_assessment).toLocaleDateString()}
                    </Typography>
                    <Typography variant="caption" color="textSecondary">
                      {Math.ceil(
                        (new Date(complianceStatus.next_assessment).getTime() -
                          Date.now()) /
                          (1000 * 60 * 60 * 24)
                      )}{' '}
                      days remaining
                    </Typography>
                  </CardContent>
                </Card>
              </Grid>
            </Grid>

            {/* Requirements List */}
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>
                Compliance Requirements
              </Typography>
              <Divider sx={{ mb: 2 }} />
              <List>
                {complianceStatus.requirements.map((req) => (
                  <Card key={req.id} sx={{ mb: 2 }}>
                    <CardContent>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                        <Box sx={{ flex: 1 }}>
                          <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                            {getStatusIcon(req.status)}
                            <Typography variant="h6" sx={{ ml: 1 }}>
                              {req.name}
                            </Typography>
                            <Chip
                              label={req.status.replace('_', ' ').toUpperCase()}
                              color={getStatusColor(req.status) as any}
                              size="small"
                              sx={{ ml: 2 }}
                            />
                            {req.automated && (
                              <Chip
                                label="Automated"
                                size="small"
                                variant="outlined"
                                sx={{ ml: 1 }}
                              />
                            )}
                          </Box>
                          <Typography variant="body2" color="textSecondary" paragraph>
                            {req.description}
                          </Typography>
                          <Grid container spacing={2}>
                            <Grid item>
                              <Typography variant="caption" color="textSecondary">
                                Evidence: {req.evidence_count} item(s)
                              </Typography>
                            </Grid>
                            <Grid item>
                              <Typography variant="caption" color="textSecondary">
                                Last Verified:{' '}
                                {new Date(req.last_verified).toLocaleDateString()}
                              </Typography>
                            </Grid>
                          </Grid>
                        </Box>
                        <Box>
                          <Tooltip title="Upload Evidence">
                            <IconButton
                              onClick={() => {
                                setSelectedRequirement(req);
                                setEvidenceDialogOpen(true);
                              }}
                            >
                              <UploadIcon />
                            </IconButton>
                          </Tooltip>
                        </Box>
                      </Box>
                    </CardContent>
                  </Card>
                ))}
              </List>
            </Paper>
          </>
        )}

        {/* Evidence Upload Dialog */}
        <Dialog
          open={evidenceDialogOpen}
          onClose={() => setEvidenceDialogOpen(false)}
          maxWidth="sm"
          fullWidth
        >
          <DialogTitle>Upload Compliance Evidence</DialogTitle>
          <DialogContent>
            {selectedRequirement && (
              <>
                <Typography variant="subtitle2" gutterBottom>
                  Requirement: {selectedRequirement.name}
                </Typography>
                <Typography variant="body2" color="textSecondary" paragraph>
                  {selectedRequirement.description}
                </Typography>
                <Alert severity="info" sx={{ mb: 2 }}>
                  Upload documentation, screenshots, or other evidence to support
                  this compliance requirement.
                </Alert>
                <input
                  type="file"
                  onChange={(e) => {
                    const file = e.target.files?.[0];
                    if (file) handleUploadEvidence(file);
                  }}
                  style={{ marginTop: 16 }}
                />
              </>
            )}
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setEvidenceDialogOpen(false)}>Cancel</Button>
          </DialogActions>
        </Dialog>

        {/* Report Generation Dialog */}
        <Dialog
          open={reportDialogOpen}
          onClose={() => setReportDialogOpen(false)}
          maxWidth="sm"
          fullWidth
        >
          <DialogTitle>Generate Compliance Report</DialogTitle>
          <DialogContent>
            <Typography variant="body2" paragraph>
              Generate a comprehensive compliance report for {getFrameworkName(activeTab)}.
            </Typography>
            <Grid container spacing={2} sx={{ mt: 1 }}>
              <Grid item xs={12}>
                <LocalizationProvider dateAdapter={AdapterDateFns}>
                  <DatePicker
                    label="Start Date"
                    value={new Date(reportRequest.start_date)}
                    onChange={(date) =>
                      setReportRequest({
                        ...reportRequest,
                        start_date: date?.toISOString() || reportRequest.start_date
                      })
                    }
                    slotProps={{ textField: { fullWidth: true } }}
                  />
                </LocalizationProvider>
              </Grid>
              <Grid item xs={12}>
                <LocalizationProvider dateAdapter={AdapterDateFns}>
                  <DatePicker
                    label="End Date"
                    value={new Date(reportRequest.end_date)}
                    onChange={(date) =>
                      setReportRequest({
                        ...reportRequest,
                        end_date: date?.toISOString() || reportRequest.end_date
                      })
                    }
                    slotProps={{ textField: { fullWidth: true } }}
                  />
                </LocalizationProvider>
              </Grid>
              <Grid item xs={12}>
                <FormControl fullWidth>
                  <InputLabel>Include Evidence</InputLabel>
                  <Select
                    value={reportRequest.include_evidence ? 'yes' : 'no'}
                    onChange={(e) =>
                      setReportRequest({
                        ...reportRequest,
                        include_evidence: e.target.value === 'yes'
                      })
                    }
                  >
                    <MenuItem value="yes">Yes - Include all evidence</MenuItem>
                    <MenuItem value="no">No - Summary only</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
            </Grid>
            <Alert severity="info" sx={{ mt: 2 }}>
              Report will be generated as PDF and downloaded to your device.
            </Alert>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setReportDialogOpen(false)}>Cancel</Button>
            <Button
              variant="contained"
              startIcon={<DownloadIcon />}
              onClick={handleGenerateReport}
              disabled={loading}
            >
              Generate PDF
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

export default Compliance;
